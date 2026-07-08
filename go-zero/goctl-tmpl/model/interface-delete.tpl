Delete(ctx context.Context, {{.lowerStartCamelPrimaryKey}} {{.dataType}}, session ...sqlx.Session) error
DeleteList(ctx context.Context, {{.lowerStartCamelPrimaryKey}}List []{{.dataType}}, session ...sqlx.Session) error
DeleteByWhere(ctx context.Context, whereCondition sqlstatement.LogicCondition, session ...sqlx.Session) error

ExecCtx(ctx context.Context, query string, args ...any) (sql.Result, error)
QueryRowCtx(ctx context.Context, v any, query string, args ...any) error
TransactCtx(ctx context.Context, fn func(context.Context, sqlx.Session) error) error
