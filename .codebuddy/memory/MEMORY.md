# 项目记忆

## assets.go 生成器
- 工具：`templatefile/generate_assets.go`
- 模板：`templatefile/assets.tmpl`（嵌入到可执行文件中）
- 默认包名：如果目录名无效或未指定，使用 "static"
- 常量命名规则：文件名去除扩展名，替换下划线和连字符为空格，单词首字母大写，拼接成驼峰格式
- 生成的文件：在目标目录中创建 `assets.go`，包含所有文件的嵌入指令和常量定义
- 跳过文件：自动跳过 `assets.go` 文件本身

## 使用方式
```bash
go run generate_assets.go <目录> [包名]
go build -o genassets generate_assets.go
./genassets <目录> [包名]
```

## 项目约定
- 静态资源文件通常放在 `templatefile/` 目录下
- 生成的 `assets.go` 用于嵌入 Excel 模板等静态文件
- 包名优先使用目录名，但需符合 Go 标识符规范

---
*最后更新：2026-06-08*