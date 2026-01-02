package tracer

import (
	"context"
	"github.com/magic-lib/go-plat-utils/cond"
	"github.com/magic-lib/go-plat-utils/conv"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"sync/atomic"
)

const (
	kindJaeger   = "jaeger"
	kindZipkin   = "zipkin"
	kindOtlpGrpc = "otlpgrpc"
	kindOtlpHttp = "otlphttp"
	kindFile     = "file"
	protocolUdp  = "udp"
)

func GetTraceConfig() *TraceConfig {
	if globalTraceConfig == nil {
		return nil
	}
	return globalTraceConfig.Load().(tracerConfigHolder).tc
}
func SetTraceConfig(tc *TraceConfig) {
	if tc == nil {
		return
	}
	if globalTraceConfig == nil {
		globalTraceConfig = &atomic.Value{}
	}
	globalTraceConfig.Store(tracerConfigHolder{
		tc: tc,
	})
}
func TraceProvider() (*sdktrace.TracerProvider, bool) {
	provider := otel.GetTracerProvider()
	if !cond.IsNil(provider) {
		tp, ok := provider.(*sdktrace.TracerProvider)
		if ok {
			return tp, true
		}
	}
	return nil, false
}

func SpanToRequest(ctx context.Context, req *http.Request, traceId, spanId string) bool {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return false
	}

	spanContext := span.SpanContext()
	changed := false
	if traceId != "" {
		traceID, err := trace.TraceIDFromHex(traceId)
		if err == nil {
			spanContext = spanContext.WithTraceID(traceID)
			changed = true
		}
	}
	if spanId != "" {
		spanID, err := trace.SpanIDFromHex(spanId)
		if err == nil {
			spanContext = spanContext.WithSpanID(spanID)
			changed = true
		}
	}

	if changed {
		ctx = trace.ContextWithSpanContext(ctx, spanContext)
	}
	//req.Header 是引用类型，可以直接修改
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
	return true
}
func SpanFromRequest(ctx context.Context, req *http.Request) context.Context {
	newCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(req.Header))
	req = req.WithContext(newCtx)
	return newCtx
}

// TracerFromContext 从上下文中获取 Tracer
func TracerFromContext(ctx context.Context, traceName string) (tracer trace.Tracer) {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		tracer = span.TracerProvider().Tracer(traceName)
	} else {
		tracer = otel.Tracer(traceName)
	}
	return tracer
}

func SetErrorTag(span trace.Span, err error) {
	if span == nil || err == nil {
		return
	}
	SetTags(span, map[string]any{
		"error":    true,
		"errormsg": err.Error(),
	})
}

// SetWarnTag 设置告警tag
func SetWarnTag(span trace.Span, err error) {
	if span == nil || err == nil {
		return
	}
	SetTags(span, map[string]any{
		"warn":    true,
		"warnmsg": err.Error(),
	})
}

// SetTags 设置通用tag属性
func SetTags(span trace.Span, attributeMap map[string]any) {
	if span == nil {
		return
	}
	attributeList := make([]attribute.KeyValue, 0)
	for key, value := range attributeMap {
		switch value.(type) {
		case int, uint, int8, uint8, int16, uint16, int32, uint32, int64, uint64:
			vValue, err := conv.Convert[int64](value)
			if err == nil {
				attributeList = append(attributeList, attribute.Int64(key, vValue))
				continue
			}
		case float32, float64:
			vValue, err := conv.Convert[float64](value)
			if err == nil {
				attributeList = append(attributeList, attribute.Float64(key, vValue))
				continue
			}
		case bool:
			vValue, err := conv.Convert[bool](value)
			if err == nil {
				attributeList = append(attributeList, attribute.Bool(key, vValue))
				continue
			}
		}
		vValue := conv.String(value)
		attributeList = append(attributeList, attribute.String(key, vValue))
	}
	if len(attributeList) > 0 {
		span.SetAttributes(attributeList...)
	}
}
