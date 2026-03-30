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

func (m *default{{.upperStartCamelObject}}Model) InsertOrUpdate(ctx context.Context, data *{{.upperStartCamelObject}}, uniqFunc func(data *{{.upperStartCamelObject}}) sqlstatement.LogicCondition, updateFunc func(data *{{.upperStartCamelObject}}) error,session ...sqlx.Session) (*{{.upperStartCamelObject}}, error) {
	if uniqFunc == nil {
        return nil, fmt.Errorf("param uniqFunc empty")
    }

    whereCond := uniqFunc(data)
    newData, err := m.Find{{.upperStartCamelObject}}(ctx, whereCond, session...)
    if err != nil {
        return nil, err
    }
    if newData == nil {
        ret, err := m.Insert(ctx, data, session...)
        if err != nil {
            return nil, err
        }
        if ret == nil {
            return nil, fmt.Errorf("insert fail")
        }
        lastId, err := ret.LastInsertId()
        if err != nil {
            return nil, err
        }
        data.{{.upperStartCamelPrimaryKey}}, _ = conv.Convert[{{.dataType}}](lastId)
        return data, nil
    }
    data.{{.upperStartCamelPrimaryKey}} = newData.{{.upperStartCamelPrimaryKey}}
    if updateFunc == nil {
        return data, m.Update(ctx, data, session...)
    }

    err = m.UpdatePartialByFunc(ctx, newData.{{.upperStartCamelPrimaryKey}}, updateFunc, session...)
    if err == nil {
        newId := newData.{{.upperStartCamelPrimaryKey}}
        _ = updateFunc(newData)
        newData.{{.upperStartCamelPrimaryKey}} = newId //防止被方法里覆盖了
        return newData, nil
        // 提高效率，避免重新查询

        // 如果已经更新成功了，则需要重新获取最终更新的数据返回
        //newUpdateData, getErr := m.FindOne(ctx, newData.{{.upperStartCamelPrimaryKey}}, session...)
        //if getErr != nil {
        //    return data, nil
        //}
        //return newUpdateData, nil
    }
    return data, err
}
