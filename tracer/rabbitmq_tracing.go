package tracer

import (
	"context"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// amqpHeadersCarrier 实现了 propagation.TextMapCarrier 接口
type amqpHeadersCarrier map[string]any

func (c amqpHeadersCarrier) Get(key string) string {
	if value, ok := c[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func (c amqpHeadersCarrier) Set(key, value string) {
	c[key] = value
}

func (c amqpHeadersCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for key := range c {
		keys = append(keys, key)
	}
	return keys
}

var _ propagation.TextMapCarrier = amqpHeadersCarrier(nil)

func (hc *TraceConfig) RabbitMQPublishTable(ctx context.Context, headers map[string]any) amqp.Table {
	otel.GetTextMapPropagator().Inject(ctx, amqpHeadersCarrier(headers))
	return headers
}

func (hc *TraceConfig) RabbitMQConsumer(ctx context.Context, headers amqp.Table) context.Context {
	ctx = otel.GetTextMapPropagator().Extract(ctx, amqpHeadersCarrier(headers))
	spanContext := trace.SpanContextFromContext(ctx)
	newCtx := trace.ContextWithSpanContext(ctx, spanContext)
	return newCtx
}
