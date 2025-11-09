Delete(ctx context.Context, {{.lowerStartCamelPrimaryKey}} {{.dataType}}, session ...sqlx.Session) error
DeleteList(ctx context.Context, {{.lowerStartCamelPrimaryKey}}List []{{.dataType}}, session ...sqlx.Session) error
DeleteByWhere(ctx context.Context, whereCondition sqlstatement.LogicCondition, session ...sqlx.Session) error
