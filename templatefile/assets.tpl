package {{.Package}}

import (
	"bytes"
    "embed"
    "fmt"
    "io/fs"
    "github.com/xuri/excelize/v2"
)

{{range .Files}}//go:embed {{.EmbedPath}}
var {{.EmbedFS}} embed.FS
{{end}}

const (
{{range .Files}}	{{.ConstName}} = "{{.EmbedPath}}"
{{end}})

type AssetFile struct {
    Key      string // 文件唯一标识
	FileName string // 文件名（含后缀）
	FullName string // 文件全路径
	BaseName string // 不含后缀的文件名
	Ext      string // 文件后缀
	Dir      string // 目录路径
	Size     int64  // 文件大小
	Content  []byte // 文件内容
	EmbedFS  embed.FS
	ModTime  string // 修改时间
}

// AssetDir represents an embedded directory, allowing fs.Sub to read files within it.
type AssetDir struct {
	Key     string   // 目录唯一标识
	DirName string   // 目录名
	EmbedFS embed.FS // 整个目录的 embed.FS
}

// fileCacheData 全局使用的静态文件缓存
var fileCacheData = map[string]*AssetFile{}

// dirCacheData 目录文件系统缓存
var dirCacheData = map[string]fs.FS{}

var allFiles = map[string]*AssetFile{
{{range .Files}}{{if not .IsDirWild}}	{{.ConstName}}: {
        Key:      "{{.ConstName}}",
        FileName: "{{.Name}}",
        FullName: {{.ConstName}},
        BaseName: "{{.BaseName}}",
        Ext:      "{{.Ext}}",
        Dir:      "{{.Dir}}",
        Size:     {{.Size}},
        Content:  nil,
        EmbedFS:  {{.EmbedFS}},
        ModTime:  "{{.ModTime}}",
    },
{{end}}{{end}}}

var allDirs = map[string]*AssetDir{
{{range .Files}}{{if .IsDirWild}}	{{.ConstName}}: {
		Key:     "{{.ConstName}}",
		DirName: "{{.Dir}}",
		EmbedFS: {{.EmbedFS}},
	},
{{end}}{{end}}}

func LoadAssetFile(fileName string) (*AssetFile, error) {
    if data, ok := fileCacheData[fileName]; ok {
		return data, nil
	}
	if f, ok := allFiles[fileName]; ok {
		data, err := f.EmbedFS.ReadFile(fileName)
		if err != nil {
			return nil, err
		}
		f.Content = data
		fileCacheData[fileName] = f
		return f, nil
	}
	return nil, fmt.Errorf("file %s not found", fileName)
}

func LoadAllAssetFiles() []*AssetFile {
	return lo.MapToSlice(allFiles, func(fileName string, file *AssetFile) *AssetFile {
		return file
	})
}

// LoadAssetDir returns an fs.FS for the given directory, allowing file access within it.
// The dirName should match the Dir field of an AssetDir entry.
func LoadAssetDir(dirName string) (fs.FS, error) {
	if dfs, ok := dirCacheData[dirName]; ok {
		return dfs, nil
	}
	if d, ok := allDirs[dirName]; ok {
		subFS, err := fs.Sub(d.EmbedFS, d.DirName)
		if err != nil {
			return nil, fmt.Errorf("failed to create sub filesystem for dir %s: %w", dirName, err)
		}
		dirCacheData[dirName] = subFS
		return subFS, nil
	}
	return nil, fmt.Errorf("dir %s not found", dirName)
}

type ExcelTmplData struct {
	NewSheetName  string // 新建文件sheet名称
	TmplSheetName string // 模版sheet名称
	Data          any    // 需要填充的数据
}

// CreateExcelByTmpl 通过excel模版创建excel文件
func (file *AssetFile) CreateExcelByTmpl(sheetNameList []*ExcelTmplData, setNewSheetValueCallback func(newFile *excelize.File, sheetNameMap []*ExcelTmplData) error) (*bytes.Buffer, error) {
	if setNewSheetValueCallback == nil {
		return nil, fmt.Errorf("setNewSheetValueCallback is nil")
	}

	if file.Ext != "xlsx" && file.Ext != "xls" {
		return nil, fmt.Errorf("file %s is not an excel file", file.FileName)
	}

	reader := bytes.NewReader(file.Content)
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	newFile := excelize.NewFile()
	defer func() {
		_ = newFile.Close()
	}()
	defaultSheet := newFile.GetSheetName(0)
	isDeletedDefaultSheet := false
	for _, tmplData := range sheetNameList {
		if tmplData.TmplSheetName == "" {
			// 查看模版是不是只有一个sheet，如果有多个，则报错，否则直接默认用第一个
			tmplSheetNameList := f.GetSheetList()
			if len(tmplSheetNameList) > 1 {
				return nil, fmt.Errorf("模版[%s]有多个sheet，请指定sheet名称", file.FileName)
			}
			tmplData.TmplSheetName = tmplSheetNameList[0]
		}
		if tmplData.NewSheetName == "" {
			if len(sheetNameList) > 1 { // 如果只有一个表数据，则默认用模版名称
				return nil, fmt.Errorf("数据[%s]有多个sheet，请指定新建的sheet名称", file.FileName)
			}
			tmplData.NewSheetName = tmplData.TmplSheetName
		}

		index, err := newFile.NewSheet(tmplData.NewSheetName)
		if err != nil {
			return nil, fmt.Errorf("创建sheet[%s]失败: %w", tmplData.NewSheetName, err)
		}
		newFile.SetActiveSheet(index)
		if !isDeletedDefaultSheet && defaultSheet != "" {
			err = newFile.DeleteSheet(defaultSheet)
			if err != nil {
				return nil, err
			}
			isDeletedDefaultSheet = true
		}

		tmplRows, err := f.GetRows(tmplData.TmplSheetName)
		if err != nil {
			return nil, fmt.Errorf("获取模板sheet[%s]数据失败: %w", tmplData.TmplSheetName, err)
		}
		for rowIdx, row := range tmplRows {
			rowNum := rowIdx + 1
			cell, _ := excelize.CoordinatesToCellName(1, rowNum)
			err = newFile.SetSheetRow(tmplData.NewSheetName, cell, &row)
			if err != nil {
				return nil, fmt.Errorf("创建sheet[%s]行数据失败: %w", tmplData.NewSheetName, err)
			}
		}
	}

	err = setNewSheetValueCallback(newFile, sheetNameList)
	if err != nil {
		return nil, err
	}

	buffer, err := newFile.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buffer, nil
}