func (m *default{{.upperStartCamelObject}}Model) update(ctx context.Context, {{if .containsIndexCache}}newData{{else}}data{{end}} *{{.upperStartCamelObject}}) error {
	{{if .withCache}}{{if .containsIndexCache}}data, err:=m.FindOne(ctx, newData.{{.upperStartCamelPrimaryKey}})
	if err!=nil{
		return err
	}

{{end}}	{{.keys}}
    _, {{if .containsIndexCache}}err{{else}}err:{{end}}= m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("update %s set %s where {{.originalPrimaryKey}} = {{if .postgreSql}}$1{{else}}?{{end}}", m.table, {{.lowerStartCamelObject}}RowsWithPlaceHolder)
		ctx, span := tracer.StartSpan(ctx, "SQL", query)
        defer span.End()
		return conn.ExecCtx(ctx, query, {{.expressionValues}})
	}, {{.keyValues}}){{else}}query := fmt.Sprintf("update %s set %s where {{.originalPrimaryKey}} = {{if .postgreSql}}$1{{else}}?{{end}}", m.table, {{.lowerStartCamelObject}}RowsWithPlaceHolder)
    ctx, span := tracer.StartSpan(ctx, "SQL", query)
    defer span.End()
    _,err:=m.conn.ExecCtx(ctx, query, {{.expressionValues}}){{end}}
	return err
}

func (m *default{{.upperStartCamelObject}}Model) Update(ctx context.Context, data *{{.upperStartCamelObject}}, session ...sqlx.Session) error {
    var oneSession sqlx.Session
    if len(session) > 0 && session[0] != nil {
        oneSession = session[0]
    } else {
        oneSession = m.conn
    }
    return m.updatePartialBySession(ctx, oneSession, data, []string{}, sqlstatement.LogicCondition{
        Conditions: []any{
            sqlstatement.Condition{
                Field:    "{{.originalPrimaryKey}}",
                Operator: sqlstatement.OperatorEqual,
                Value:    data.{{.upperStartCamelPrimaryKey}},
            },
        },
    })
}

func (m *default{{.upperStartCamelObject}}Model) UpdatePartial(ctx context.Context, data *{{.upperStartCamelObject}}, columns []string, whereCondition sqlstatement.LogicCondition, session ...sqlx.Session) error {
    var oneSession sqlx.Session
    if len(session) > 0 && session[0] != nil {
        oneSession = session[0]
    } else {
        oneSession = m.conn
    }
    return m.updatePartialBySession(ctx, oneSession, data, columns, whereCondition)
}

func (m *default{{.upperStartCamelObject}}Model) UpdatePartialByFunc(ctx context.Context, {{.lowerStartCamelPrimaryKey}} {{.dataType}}, updateFunc func(data *{{.upperStartCamelObject}}) error, session ...sqlx.Session) error {
	one, err := m.FindOne(ctx, {{.lowerStartCamelPrimaryKey}}, session...)
	if err != nil {
		return err
	}
	if one == nil {
	    return fmt.Errorf("{{.upperStartCamelObject}} : %v not found", {{.lowerStartCamelPrimaryKey}})
	}
	err = updateFunc(one)
	if err != nil {
		return err
	}
	one.{{.upperStartCamelPrimaryKey}} = {{.lowerStartCamelPrimaryKey}}
	return m.Update(ctx, one, session...)
}

func (m *default{{.upperStartCamelObject}}Model) updatePartialBySession(ctx context.Context, session sqlx.Session, data *{{.upperStartCamelObject}}, columns []string, whereCondition sqlstatement.LogicCondition) error {
	if data == nil {
		return nil
	}
	sql, list := new(sqlstatement.Statement).GenerateWhereClause(whereCondition)
    if len(list) == 0 || sql == "" {
        // 更新的条件不能为空，避免全表误更新了
        return fmt.Errorf("param update where empty")
    }
    query, updateData, err := m.sqlBuilder.UpdateSql(data, columns, whereCondition)
    if err != nil {
        return err
    }
    ctx, span := tracer.StartSpan(ctx, "SQL", query)
    defer span.End()

    if session == nil {
        session = m.conn
    }
    _, err = session.ExecCtx(ctx, query, updateData...)
    return err
}
