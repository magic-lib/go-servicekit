func new{{.upperStartCamelObject}}Model(conn sqlx.SqlConn{{if .withCache}}, c cache.CacheConf, opts ...cache.Option{{end}}) *default{{.upperStartCamelObject}}Model {
	if data, ok := {{.lowerStartCamelObject}}ModelCache.Load(conn); ok {
        dataModel, ok := data.(*default{{.upperStartCamelObject}}Model)
        if ok {
            return dataModel
        }
    }
	sqlBuilder := sqlstatement.NewSqlStruct(
        sqlstatement.SetColumnTagName("db"),
        sqlstatement.SetStructData({{.upperStartCamelObject}}{}),
        sqlstatement.SetColumnListBySqlConn(conn, {{.table}}),
        sqlstatement.SetTableName({{.table}}),
    )
    default{{.upperStartCamelObject}} := &default{{.upperStartCamelObject}}Model{
        {{if .withCache}}CachedConn: sqlc.NewConn(conn, c, opts...){{else}}conn:conn{{end}},
        sqlBuilder: sqlBuilder,
        table:      {{.table}},
    }
    {{.lowerStartCamelObject}}ModelCache.Store(conn, default{{.upperStartCamelObject}})
	return default{{.upperStartCamelObject}}
}
