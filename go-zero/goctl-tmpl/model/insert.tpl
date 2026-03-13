func (m *default{{.upperStartCamelObject}}Model) insert(ctx context.Context, data *{{.upperStartCamelObject}}) (sql.Result,error) {
	{{if .withCache}}{{.keys}}
    ret, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("insert into %s (%s) values ({{.expression}})", m.table, {{.lowerStartCamelObject}}RowsExpectAutoSet)
        //ctx, span := tracer.StartSpan(ctx, "SQL", query)
	    //defer span.End()
		return conn.ExecCtx(ctx, query, {{.expressionValues}})
	}, {{.keyValues}}){{else}}query := fmt.Sprintf("insert into %s (%s) values ({{.expression}})", m.table, {{.lowerStartCamelObject}}RowsExpectAutoSet)
    //ctx, span := tracer.StartSpan(ctx, "SQL", query)
    //defer span.End()
	return m.conn.ExecCtx(ctx, query, {{.expressionValues}}){{end}}
}

func (m *default{{.upperStartCamelObject}}Model) Insert(ctx context.Context, data *{{.upperStartCamelObject}}, session ...sqlx.Session) (sql.Result,error) {
	//query := fmt.Sprintf("insert into %s (%s) values ({{.expression}})", m.table, {{.lowerStartCamelObject}}RowsExpectAutoSet)
    //ctx, span := tracer.StartSpan(ctx, "SQL", query)
    //defer span.End()

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
	//query := fmt.Sprintf("insert into %s (%s) values ({{.expression}})", m.table, {{.lowerStartCamelObject}}RowsExpectAutoSet)
    //ctx, span := tracer.StartSpan(ctx, "SQL", query)
    //defer span.End()

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

func (m *default{{.upperStartCamelObject}}Model) InsertOrUpdate(ctx context.Context, data *{{.upperStartCamelObject}}, uniqFunc func(data *{{.upperStartCamelObject}}) sqlstatement.LogicCondition, updateFunc func(data *{{.upperStartCamelObject}}) error,session ...sqlx.Session) error {
	if uniqFunc == nil {
        return fmt.Errorf("param uniqFunc empty")
    }

    whereCond := uniqFunc(data)
    newData, err := m.Find{{.upperStartCamelObject}}(ctx, whereCond, session...)
    if err != nil {
        return err
    }
    if newData == nil {
        ret, err := m.Insert(ctx, data, session...)
        if err != nil {
            return err
        }
        if ret == nil {
            return fmt.Errorf("insert fail")
        }
        data.{{.upperStartCamelPrimaryKey}}, _ = ret.LastInsertId()
        return nil
    }
    data.{{.upperStartCamelPrimaryKey}} = newData.{{.upperStartCamelPrimaryKey}}
    if updateFunc == nil {
        return m.Update(ctx, data, session...)
    }
    return m.UpdatePartialByFunc(ctx, newData.{{.upperStartCamelPrimaryKey}}, updateFunc, session...)
}
