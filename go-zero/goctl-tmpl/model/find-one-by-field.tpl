func (m *default{{.upperStartCamelObject}}Model) Find{{.upperStartCamelObject}}By{{.upperField}}(ctx context.Context, {{.in}}) (*{{.upperStartCamelObject}}, error) {
	{{if .withCache}}{{.cacheKey}}
	var resp {{.upperStartCamelObject}}
	err := m.QueryRowIndexCtx(ctx, &resp, {{.cacheKeyVariable}}, m.formatPrimary, func(ctx context.Context, conn sqlx.SqlConn, v any) (i any, e error) {
		query := fmt.Sprintf("select %s from %s where {{.originalField}} limit 1", {{.lowerStartCamelObject}}Rows, m.table)
		ctx, span := tracer.StartSpan(ctx, "SQL", query)
        defer span.End()
		if err := conn.QueryRowCtx(ctx, &resp, query, {{.lowerStartCamelField}}); err != nil {
			return nil, err
		}
		return resp.{{.upperStartCamelPrimaryKey}}, nil
	}, m.queryPrimary)
	switch err {
	case nil:
		return &resp, nil
	case sqlc.ErrNotFound:
    	return nil, nil
	default:
		return nil, err
	}
}{{else}}var resp {{.upperStartCamelObject}}
	query := fmt.Sprintf("select %s from %s where {{.originalField}} limit 1", {{.lowerStartCamelObject}}Rows, m.table )
	ctx, span := tracer.StartSpan(ctx, "SQL", query)
    defer span.End()
	err := m.conn.QueryRowCtx(ctx, &resp, query, {{.lowerStartCamelField}})
	switch err {
	case nil:
		return &resp, nil
	case sqlx.ErrNotFound:
		return nil, nil
	default:
		return nil, err
	}
}{{end}}