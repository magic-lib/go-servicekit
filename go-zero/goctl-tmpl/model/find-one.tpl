func (m *default{{.upperStartCamelObject}}Model) FindOne(ctx context.Context, {{.lowerStartCamelPrimaryKey}} {{.dataType}}, session ...sqlx.Session) (*{{.upperStartCamelObject}}, error) {
	{{if .withCache}}{{.cacheKey}}
	var resp {{.upperStartCamelObject}}
	err := m.QueryRowCtx(ctx, &resp, {{.cacheKeyVariable}}, func(ctx context.Context, conn sqlx.SqlConn, v any) error {
		query :=  fmt.Sprintf("select %s from %s where {{.originalPrimaryKey}} = {{if .postgreSql}}$1{{else}}?{{end}} limit 1", {{.lowerStartCamelObject}}Rows, m.table)
		ctx, span := tracer.StartSpan(ctx, "SQL", query)
        defer span.End()
		return conn.QueryRowCtx(ctx, v, query, {{.lowerStartCamelPrimaryKey}})
	})
	switch err {
	case nil:
		return &resp, nil
	case sqlc.ErrNotFound:
    	return nil, nil
	default:
		return nil, err
	}{{else}}query := fmt.Sprintf("select %s from %s where {{.originalPrimaryKey}} = {{if .postgreSql}}$1{{else}}?{{end}} limit 1", {{.lowerStartCamelObject}}Rows, m.table)
	var resp {{.upperStartCamelObject}}
	ctx, span := tracer.StartSpan(ctx, "SQL", query)
    defer span.End()
    var oneSession sqlx.Session
    if len(session) > 0 && session[0] != nil {
        oneSession = session[0]
    }else{
        oneSession = m.conn
    }
	err := oneSession.QueryRowCtx(ctx, &resp, query, {{.lowerStartCamelPrimaryKey}})
	switch err {
	case nil:
		return &resp, nil
	case sqlx.ErrNotFound:
    	return nil, nil
	default:
		return nil, err
	}{{end}}
}

func (m *default{{.upperStartCamelObject}}Model) Find{{.upperStartCamelObject}}(ctx context.Context, whereCond sqlstatement.LogicCondition, session ...sqlx.Session) (*{{.upperStartCamelObject}}, error) {
	query, data, err := m.sqlBuilder.SelectSql("*", whereCond, 0, 1)
	if err != nil {
        return nil, err
    }
	resp := new({{.upperStartCamelObject}})
	ctx, span := tracer.StartSpan(ctx, "SQL", query)
    defer span.End()
    var oneSession sqlx.Session
    if len(session) > 0 && session[0] != nil {
        oneSession = session[0]
    }else{
        oneSession = m.conn
    }
	err = oneSession.QueryRowCtx(ctx, resp, query, data...)
	switch err {
	case nil:
		return resp, nil
	case sqlx.ErrNotFound:
    	return nil, nil
	default:
		return nil, err
	}
}

func (m *default{{.upperStartCamelObject}}Model) List{{.upperStartCamelObject}}(ctx context.Context, whereCond sqlstatement.LogicCondition, session ...sqlx.Session) ([]*{{.upperStartCamelObject}}, error) {
	list, _, err := m.List{{.upperStartCamelObject}}ByPage(ctx, whereCond, nil, 0, "", session...)
	return list, err
}


func (m *default{{.upperStartCamelObject}}Model) List{{.upperStartCamelObject}}ByPage(ctx context.Context, whereCond sqlstatement.LogicCondition, pageModel *httputil.PageModel, maxLimit int, orderBy string, session ...sqlx.Session) ([]*{{.upperStartCamelObject}}, int64, error) {
	if pageModel == nil {
        pageModel = new(httputil.PageModel)
    }else{
        if pageModel.PageSize > 0 { // 每页数量
            if maxLimit == 0 {
                maxLimit = 100
            }
            pageModel = pageModel.GetPage(maxLimit)
        }
    }

    var query  string
    var data []any
    var err error
    var useCount bool

    if orderBy == "" {
        query, data, err = m.sqlBuilder.SelectSql("*", whereCond, pageModel.PageOffset, pageModel.PageSize)
        if err != nil {
            return nil, 0, err
        }
    }else{
        query, data, err = m.sqlBuilder.SelectSql("*", whereCond, 0, 0)
        if err != nil {
            return nil, 0, err
        }
        if orderBy != "" {
            query = fmt.Sprintf("%s %s", query, orderBy)
        }
        if pageModel.PageNow > 0 && pageModel.PageSize > 0 {
            query = fmt.Sprintf("%s LIMIT %d,%d", query, pageModel.PageOffset, pageModel.PageSize)
            useCount = true
        }
    }

	list := make([]*{{.upperStartCamelObject}}, 0)
	ctx, span := tracer.StartSpan(ctx, "SQL", query)
    defer span.End()

    var oneSession sqlx.Session
    if len(session) > 0 && session[0] != nil {
        oneSession = session[0]
    }else{
        oneSession = m.conn
    }
	err = oneSession.QueryRowsCtx(ctx, &list, query, data...)
	listLen := int64(len(list))
	if err != nil {
		return list, listLen, err
	}
	if !useCount {
		return list, listLen, nil
	}

	total, err := m.Count{{.upperStartCamelObject}}(ctx, whereCond, session...)
	if err != nil {
		return list, listLen, err
	}

	return list, total, nil
}


func (m *default{{.upperStartCamelObject}}Model) Count{{.upperStartCamelObject}}(ctx context.Context, whereCond sqlstatement.LogicCondition, session ...sqlx.Session) (int64, error) {
	countSql, countData, err := m.sqlBuilder.SelectSql("COUNT(*)", whereCond, 0, 0)
    if err != nil {
        return 0, err
    }
    var total int64

    ctx, span := tracer.StartSpan(ctx, "SQL", countSql)
    defer span.End()
    var oneSession sqlx.Session
    if len(session) > 0 && session[0] != nil {
        oneSession = session[0]
    }else{
        oneSession = m.conn
    }
    err = oneSession.QueryRowCtx(ctx, &total, countSql, countData...)
    if err != nil {
        return 0, err
    }
    return total, nil
}