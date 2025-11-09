func (m *default{{.upperStartCamelObject}}Model) delete(ctx context.Context, {{.lowerStartCamelPrimaryKey}} {{.dataType}}) error {
	{{if .withCache}}{{if .containsIndexCache}}data, err:=m.FindOne(ctx, {{.lowerStartCamelPrimaryKey}})
	if err!=nil{
		return err
	}

{{end}}	{{.keys}}
    _, err {{if .containsIndexCache}}={{else}}:={{end}} m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("delete from %s where {{.originalPrimaryKey}} = {{if .postgreSql}}$1{{else}}?{{end}}", m.table)
		ctx, span := tracer.StartSpan(ctx, "SQL", query)
        defer span.End()
		return conn.ExecCtx(ctx, query, {{.lowerStartCamelPrimaryKey}})
	}, {{.keyValues}}){{else}}query := fmt.Sprintf("delete from %s where {{.originalPrimaryKey}} = {{if .postgreSql}}$1{{else}}?{{end}}", m.table)
		ctx, span := tracer.StartSpan(ctx, "SQL", query)
        defer span.End()
		_,err:=m.conn.ExecCtx(ctx, query, {{.lowerStartCamelPrimaryKey}}){{end}}
	return err
}

func (m *default{{.upperStartCamelObject}}Model) Delete(ctx context.Context, {{.lowerStartCamelPrimaryKey}} {{.dataType}}, session ...sqlx.Session) error {
    whereCondition := sqlstatement.LogicCondition{
        Conditions: []any{
            sqlstatement.Condition{
                Field:    "{{.originalPrimaryKey}}",
                Operator: sqlstatement.OperatorEqual,
                Value:    {{.lowerStartCamelPrimaryKey}},
            },
        },
    }
    return m.DeleteByWhere(ctx, whereCondition, session...)
}

func (m *default{{.upperStartCamelObject}}Model) DeleteList(ctx context.Context, {{.lowerStartCamelPrimaryKey}}List []{{.dataType}}, session ...sqlx.Session) error {
	if len({{.lowerStartCamelPrimaryKey}}List) == 0 {
    	return fmt.Errorf("param {{.lowerStartCamelPrimaryKey}}List empty")
    }
    whereCondition := sqlstatement.LogicCondition{
        Conditions: []any{
            sqlstatement.Condition{
                Field:    "{{.originalPrimaryKey}}",
                Operator: sqlstatement.OperatorIn,
                Value:    {{.lowerStartCamelPrimaryKey}}List,
            },
        },
    }
    return m.DeleteByWhere(ctx, whereCondition, session...)
}

func (m *default{{.upperStartCamelObject}}Model) DeleteByWhere(ctx context.Context, whereCondition sqlstatement.LogicCondition, session ...sqlx.Session) error {
	sql, list := new(sqlstatement.Statement).GenerateWhereClause(whereCondition)
    if len(list) == 0 || sql == "" {
        // 删除的条件不能为空，避免全表误删除了
        return fmt.Errorf("param delete where empty")
    }
	query, deleteData, err := m.sqlBuilder.DeleteSql(whereCondition)
	if err != nil {
		return err
	}
	ctx, span := tracer.StartSpan(ctx, "SQL", query)
	defer span.End()

	if len(session) > 0 && session[0] != nil {
        _, err = session[0].ExecCtx(ctx, query, deleteData...)
        return err
    }
    _, err = m.conn.ExecCtx(ctx, query, deleteData...)
    return err
}