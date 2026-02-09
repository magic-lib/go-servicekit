func (m *default{{.upperStartCamelObject}}Model) insert(ctx context.Context, data *{{.upperStartCamelObject}}) (sql.Result,error) {
	{{if .withCache}}{{.keys}}
    ret, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("insert into %s (%s) values ({{.expression}})", m.table, {{.lowerStartCamelObject}}RowsExpectAutoSet)
        ctx, span := tracer.StartSpan(ctx, "SQL", query)
	    defer span.End()
		return conn.ExecCtx(ctx, query, {{.expressionValues}})
	}, {{.keyValues}}){{else}}query := fmt.Sprintf("insert into %s (%s) values ({{.expression}})", m.table, {{.lowerStartCamelObject}}RowsExpectAutoSet)
    ctx, span := tracer.StartSpan(ctx, "SQL", query)
    defer span.End()
	return m.conn.ExecCtx(ctx, query, {{.expressionValues}}){{end}}
}

func (m *default{{.upperStartCamelObject}}Model) Insert(ctx context.Context, data *{{.upperStartCamelObject}}, session ...sqlx.Session) (sql.Result,error) {
	query := fmt.Sprintf("insert into %s (%s) values ({{.expression}})", m.table, {{.lowerStartCamelObject}}RowsExpectAutoSet)
    ctx, span := tracer.StartSpan(ctx, "SQL", query)
    defer span.End()

    insertSql, insertData, err := m.sqlBuilder.InsertSql(data)
    if err != nil {
        return nil, err
    }
    var oneSession sqlx.Session
    if len(session) > 0 && session[0] != nil {
        oneSession = session[0]
    } else {
        oneSession = m.conn
    }
    return oneSession.ExecCtx(ctx, insertSql, insertData...)
}

func (m *default{{.upperStartCamelObject}}Model) InsertList(ctx context.Context, dataList []*{{.upperStartCamelObject}}, session ...sqlx.Session) (sql.Result,error) {
	query := fmt.Sprintf("insert into %s (%s) values ({{.expression}})", m.table, {{.lowerStartCamelObject}}RowsExpectAutoSet)
    ctx, span := tracer.StartSpan(ctx, "SQL", query)
    defer span.End()

    insertSql, insertData, err := m.sqlBuilder.InsertSql(dataList)
    if err != nil {
        return nil, err
    }
    var oneSession sqlx.Session
    if len(session) > 0 && session[0] != nil {
        oneSession = session[0]
    } else {
        oneSession = m.conn
    }
    return oneSession.ExecCtx(ctx, insertSql, insertData...)
}
