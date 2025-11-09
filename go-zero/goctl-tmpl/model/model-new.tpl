func new{{.upperStartCamelObject}}Model(conn sqlx.SqlConn{{if .withCache}}, c cache.CacheConf, opts ...cache.Option{{end}}) *default{{.upperStartCamelObject}}Model {
	sqlDb, _ := conn.RawDB()
	sqlBuilder := sqlstatement.NewSqlStruct(
        sqlstatement.SetColumnTagName("db"),
        sqlstatement.SetStructData({{.upperStartCamelObject}}{}),
        sqlstatement.SetColumnList(sqlDb, {{.table}}),
        sqlstatement.SetTableName({{.table}}),
    )
	return &default{{.upperStartCamelObject}}Model{
		{{if .withCache}}CachedConn: sqlc.NewConn(conn, c, opts...){{else}}conn:conn{{end}},
		sqlBuilder: sqlBuilder,
		table:      {{.table}},
	}
}
