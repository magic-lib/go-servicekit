package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"
	"unicode"
)

//go:embed assets.tpl
var templateFS embed.FS

func main() {
	var (
		dir    string
		goOut  string
		goFile string
		pkg    string
	)
	flag.StringVar(&dir, "dir", "", "Input directory(s) to scan for files, comma-separated (required)")
	flag.StringVar(&goOut, "go_out", "", "Output directory for generated Go file (default: first input directory)")
	flag.StringVar(&goFile, "go_file", "", "Output Go file name (default: assets.go)")
	flag.StringVar(&pkg, "pkg", "", "Package name for generated Go file (default: basename of go_out or first input directory)")
	flag.Parse()

	// Backward compatibility: if first non-flag argument is provided, treat as dir
	if dir == "" && flag.NArg() > 0 {
		dir = flag.Arg(0)
		// Second non-flag argument as package name
		if pkg == "" && flag.NArg() > 1 {
			pkg = flag.Arg(1)
		}
	}

	if dir == "" {
		fmt.Fprintf(os.Stderr, "Error: input directory is required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <directory> [pkgName]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Split comma-separated directories and tpl_paths
	dirs := splitCommaSeparated(dir)

	// Ensure all input directories exist
	for i, d := range dirs {
		info, err := os.Stat(d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading directory %s: %v\n", d, err)
			os.Exit(1)
		}
		if !info.IsDir() {
			fmt.Fprintf(os.Stderr, "%s is not a directory\n", d)
			os.Exit(1)
		}
		// Normalize path
		absPath, err := filepath.Abs(d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting absolute path for %s: %v\n", d, err)
			os.Exit(1)
		}
		dirs[i] = absPath
	}

	// Determine output file name early (used to skip the generated file itself during walk)
	outputFileName := goFile
	if outputFileName == "" {
		outputFileName = "assets.go"
	}

	// Collect files from all directories
	type collectedFile struct {
		RelPath   string // Relative path within its source directory
		SourceDir string // Absolute path of source directory
		IsDirWild bool   // True if this entry represents a directory wildcard (e.g. "js/*")
	}
	var allFiles []collectedFile

	// Track directories that contain files, to add wildcard entries
	dirsWithFiles := make(map[string]string) // key: relative dir path, value: sourceDir

	for _, sourceDir := range dirs {
		err := filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			// Skip the generated Go file itself
			if filepath.Base(path) == outputFileName {
				return nil
			}
			// Skip temporary files starting with ~
			baseName := filepath.Base(path)
			if strings.HasPrefix(baseName, "~") {
				return nil
			}

			// Get relative path from source directory
			rel, err := filepath.Rel(sourceDir, path)
			if err != nil {
				return err
			}

			// Track parent directory for wildcard entries
			relDir := filepath.Dir(rel)
			if relDir != "." {
				dirsWithFiles[relDir] = sourceDir
			}

			allFiles = append(allFiles, collectedFile{
				RelPath:   rel,
				SourceDir: sourceDir,
			})
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error walking directory %s: %v\n", sourceDir, err)
			os.Exit(1)
		}
	}

	// Prepend directory wildcard entries (e.g. "js/*") to allFiles,
	// so they appear first in the generated Go code.
	// This allows using fs.Sub(embedFS, "js") to get the entire subdirectory.
	for _, sourceDir := range dirs {
		// Collect dirs that belong to this sourceDir, sorted for stable output
		var dirsForSource []string
		for relDir, sd := range dirsWithFiles {
			if sd == sourceDir {
				dirsForSource = append(dirsForSource, relDir)
			}
		}
		// Sort to ensure consistent order
		sort.Strings(dirsForSource)

		for _, relDir := range dirsForSource {
			allFiles = append([]collectedFile{{
				RelPath:   relDir + "/*",
				SourceDir: sourceDir,
				IsDirWild: true,
			}}, allFiles...)
		}
	}

	if len(allFiles) == 0 {
		fmt.Fprintf(os.Stderr, "No files found in any input directory\n")
		os.Exit(1)
	}

	// Determine output directory
	outputDir := dirs[0] // Default to first input directory
	if goOut != "" {
		outputDir = goOut
		// Create output directory if it doesn't exist
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output directory %s: %v\n", outputDir, err)
			os.Exit(1)
		}
	}
	// Normalize outputDir to absolute path for Rel() computation
	if absOutputDir, err := filepath.Abs(outputDir); err == nil {
		outputDir = absOutputDir
	}

	// Determine package name
	if pkg == "" {
		// Use basename of go_out if provided, otherwise basename of first input directory
		if goOut != "" {
			pkg = filepath.Base(goOut)
		} else {
			pkg = filepath.Base(dirs[0])
		}
	}
	if !isValidIdentifier(pkg) {
		panic(fmt.Sprintf("Invalid package name: %s", pkg))
	}

	// Prepare file infos
	type fileInfo struct {
		EmbedFS   string
		EmbedPath string
		ConstName string
		Name      string    // 文件名（含后缀）
		BaseName  string    // 不含后缀的文件名
		Ext       string    // 文件后缀（如 .xlsx）
		Dir       string    // 目录路径
		Size      int64     // 文件大小
		Content   []byte    // 文件内容
		ModTime   time.Time // 修改时间
		IsDirWild bool      // True if this entry represents a directory wildcard (e.g. "js/*")
	}
	var infos []fileInfo
	for i, cf := range allFiles {
		// Get file full path (strip wildcard for directory wildcard entries)
		relForPath := cf.RelPath
		if cf.IsDirWild {
			relForPath = strings.TrimSuffix(cf.RelPath, "/*")
		}
		fullPath := filepath.Join(cf.SourceDir, relForPath)

		// EmbedPath is the relative path from outputDir (where generated .go lives) to the source
		embedPath, err := filepath.Rel(outputDir, fullPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error computing relative path from %s to %s: %v\n", outputDir, fullPath, err)
			os.Exit(1)
		}
		// For directory wildcard entries, append "/*" to the embed path
		if cf.IsDirWild {
			embedPath = embedPath + "/*"
		}

		// Generate constant name from EmbedPath to ensure uniqueness across directories
		constName := embedPathToConst(embedPath)

		// For wildcard dir entries, we don't stat an actual file
		var (
			baseName      string
			ext2          string
			nameWithoutExt string
			dirTemp       string
			fileSize      int64
			modTime       time.Time
		)

		if cf.IsDirWild {
			// Wildcard entry: use the dir name as the "Name"
			dirTemp = filepath.Base(filepath.Dir(embedPath))
			if dirTemp == "." {
				dirTemp = ""
			}
			baseName = dirTemp
			nameWithoutExt = dirTemp
			ext2 = ""
			fileSize = 0
			modTime = time.Time{}
		} else {
			// Read file info and content
			fileInfoStat, err := os.Stat(fullPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", fullPath, err)
				os.Exit(1)
			}

			baseName = filepath.Base(fullPath)
			ext := filepath.Ext(baseName)
			ext2 = ext
			if len(ext) > 0 {
				ext2 = ext[1:]
			}
			nameWithoutExt = baseName[:len(baseName)-len(ext)]

			// Dir is the directory portion of EmbedPath (e.g. "js" for "js/aaa.js", "" for root-level files)
			dirTemp = filepath.Dir(embedPath)
			if dirTemp == "." {
				dirTemp = ""
			}
			fileSize = fileInfoStat.Size()
			modTime = fileInfoStat.ModTime()
		}

		infos = append(infos, fileInfo{
			EmbedPath: embedPath,
			ConstName: constName,
			Name:      baseName,
			BaseName:  nameWithoutExt,
			Ext:       ext2,
			Dir:       dirTemp,
			Size:      fileSize,
			Content:   nil,
			EmbedFS:   fmt.Sprintf("file%03d", i+1),
			ModTime:   modTime,
			IsDirWild: cf.IsDirWild,
		})
	}

	// Load embedded template
	tmplContent, err := templateFS.ReadFile("assets.tpl")
	if err != nil {
		panic(err)
	}
	tmpl, err := template.New("assets").Parse(string(tmplContent))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing template: %v\n", err)
		os.Exit(1)
	}

	// Execute template
	data := struct {
		Package string
		Files   []fileInfo
	}{
		Package: pkg,
		Files:   infos,
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing template: %v\n", err)
		os.Exit(1)
	}

	// Write generated Go file to output directory
	outputPath := filepath.Join(outputDir, outputFileName)
	err = os.WriteFile(outputPath, buf.Bytes(), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outputFileName, err)
		os.Exit(1)
	}
	fmt.Printf("Generated %s with %d embedded files (package: %s)\n", outputPath, len(allFiles), pkg)
}

func filenameToConst(filename string) string {
	// Remove extension
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	// Replace _ and - with spaces
	replacer := strings.NewReplacer("_", " ", "-", " ")
	words := strings.Fields(replacer.Replace(name))
	// Capitalize each word
	for i, w := range words {
		// Ensure word is not empty
		if len(w) == 0 {
			continue
		}
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		// Lowercase the rest? Keep as is for consistency with existing (BussinessIncomeTemplate)
		// We'll just capitalize first letter, leave others unchanged
		words[i] = string(runes)
	}
	constName := strings.Join(words, "")
	// Ensure first character is letter
	if len(constName) == 0 {
		constName = "File"
	} else if !unicode.IsLetter([]rune(constName)[0]) {
		constName = "File" + constName
	}
	return constName
}

func isValidIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, r := range s {
		if i == 0 && !unicode.IsLetter(r) && r != '_' {
			return false
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

func splitCommaSeparated(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	// Trim spaces
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// embedPathToConst converts an embed path like "js/aaa.js" or "css/sub/style.css" to a
// unique Go constant name like "JsAaaJs" or "CssSubStyleCss".
// The full path (including directories) is used to guarantee uniqueness across all files.
func embedPathToConst(embedPath string) string {
	// Normalize separators: replace /, \, _, -, ., * with space, and split on common delimiters
	normalized := strings.NewReplacer("/", " ", "\\", " ", "_", " ", "-", " ", ".", " ", "*", " ").Replace(embedPath)
	parts := strings.Fields(normalized)

	var words []string
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		words = append(words, string(runes))
	}

	constName := strings.Join(words, "")
	// Ensure first character is letter
	if len(constName) == 0 {
		constName = "File"
	} else if !unicode.IsLetter([]rune(constName)[0]) {
		constName = "File" + constName
	}
	return constName
}
