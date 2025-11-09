type (
	{{.lowerStartCamelObject}}Model interface{
		{{.method}}
	}

	default{{.upperStartCamelObject}}Model struct {
		{{if .withCache}}sqlc.CachedConn{{else}}conn sqlx.SqlConn{{end}}
		gormDB *gorm.DB
		sqlBuilder *sqlstatement.SqlStruct
		table string
	}

	{{.upperStartCamelObject}} struct {
		{{.fields}}
	}
)
