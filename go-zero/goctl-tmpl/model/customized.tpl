func (m *default{{.upperStartCamelObject}}Model) GormDB(ctx context.Context) (*gorm.DB, error) {
	if m.gormDB != nil {
		gormDB := tracer.GetTraceConfig().GormMiddleware(ctx, m.gormDB)
		return gormDB, nil
	}

	sqlDb, err := m.conn.RawDB()
	if err != nil {
		return nil, err
	}
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn: sqlDb,
	}), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	m.gormDB = gormDB
	gormDB = tracer.GetTraceConfig().GormMiddleware(ctx, gormDB)
	return gormDB, nil
}


func (m *default{{.upperStartCamelObject}}Model) ExecCtx(ctx context.Context, query string, args ...any) (sql.Result,error) {
    ctx, span := tracer.StartSpan(ctx, "SQL", query)
    defer span.End()
	return m.conn.ExecCtx(ctx, query, args...)
}

func (m *default{{.upperStartCamelObject}}Model) PrepareCtx(ctx context.Context, query string) (sqlx.StmtSession,error) {
    ctx, span := tracer.StartSpan(ctx, "SQL", query)
    defer span.End()
	return m.conn.PrepareCtx(ctx, query)
}

func (m *default{{.upperStartCamelObject}}Model) QueryRowCtx(ctx context.Context, v any, query string, args ...any) error {
    ctx, span := tracer.StartSpan(ctx, "SQL", query)
    defer span.End()
	return m.conn.QueryRowCtx(ctx, v, query, args...)
}

func (m *default{{.upperStartCamelObject}}Model) QueryRowPartialCtx(ctx context.Context, v any, query string, args ...any) error {
    ctx, span := tracer.StartSpan(ctx, "SQL", query)
    defer span.End()
	return m.conn.QueryRowPartialCtx(ctx, v, query, args...)
}

func (m *default{{.upperStartCamelObject}}Model) QueryRowsCtx(ctx context.Context, v any, query string, args ...any) error {
    ctx, span := tracer.StartSpan(ctx, "SQL", query)
    defer span.End()
	return m.conn.QueryRowsCtx(ctx, v, query, args...)
}

func (m *default{{.upperStartCamelObject}}Model) QueryRowsPartialCtx(ctx context.Context, v any, query string, args ...any) error {
    ctx, span := tracer.StartSpan(ctx, "SQL", query)
    defer span.End()
	return m.conn.QueryRowsPartialCtx(ctx, v, query, args...)
}

func (m *default{{.upperStartCamelObject}}Model) TransactCtx(ctx context.Context, fn func(context.Context, sqlx.Session) error) error {
    ctx, span := tracer.StartSpan(ctx, "TRANSACT", "TransactCtx")
    defer span.End()
	return m.conn.TransactCtx(ctx, fn)
}