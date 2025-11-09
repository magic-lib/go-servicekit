package tracer

import (
	"context"
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

func (hc *TraceConfig) GormMiddleware(ctx context.Context, db *gorm.DB) *gorm.DB {
	err := hc.checkConfig()
	if err != nil {
		return db
	}
	err = db.Use(tracing.NewPlugin())
	if err != nil {
		return db
	}
	db = db.WithContext(ctx)
	return db
}
