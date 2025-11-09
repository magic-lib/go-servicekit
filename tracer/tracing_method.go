package tracer

import (
	"context"
	"fmt"
	"github.com/magic-lib/go-plat-utils/cond"
	"github.com/magic-lib/go-plat-utils/conv"
	"github.com/magic-lib/go-plat-utils/utils/httputil"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"strings"
	"sync/atomic"
)

const (
	kindJaeger   = "jaeger"
	kindZipkin   = "zipkin"
	kindOtlpGrpc = "otlpgrpc"
	kindOtlpHttp = "otlphttp"
	kindFile     = "file"
	protocolUdp  = "udp"

	maxSpanNameLength = 60
)

func GetTraceConfig() *TraceConfig {
	if globalTraceConfig == nil {
		return new(TraceConfig)
	}
	globalTraceConfigLock.Lock()
	defer globalTraceConfigLock.Unlock()

	tc := globalTraceConfig.Load().(tracerConfigHolder).tc
	if tc != nil {
		return tc
	}
	return new(TraceConfig)
}
func SetTraceConfig(tc *TraceConfig) {
	if tc == nil {
		return
	}
	err := tc.checkConfig()
	if err != nil {
		return
	}
	globalTraceConfigLock.Lock()
	defer globalTraceConfigLock.Unlock()
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
		if ok && tp != nil {
			return tp, true
		}
	}
	return nil, false
}

func SpanToHeader(ctx context.Context, headers http.Header, createNewSpan func(ctx context.Context) (context.Context, trace.Span)) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}

	span := trace.SpanFromContext(ctx)
	if span == nil || !span.SpanContext().IsValid() {
		if createNewSpan == nil {
			return ctx, span
		}
		ctx, span = createNewSpan(ctx)
	}

	//spanContext := span.SpanContext()

	//if traceId != "" {
	//	traceID, err := trace.TraceIDFromHex(traceId)
	//	if err == nil {
	//		spanContext = spanContext.WithTraceID(traceID)
	//		changed = true
	//	}
	//}
	//if spanId != "" {
	//	spanID, err := trace.SpanIDFromHex(spanId)
	//	if err == nil {
	//		spanContext = spanContext.WithSpanID(spanID)
	//		changed = true
	//	}
	//}
	// Traceparent
	// 00-f6db96bfd2e5b58349c18eb6e720da84-d0c2b79e207757c3-01
	ctx = trace.ContextWithSpan(ctx, span)
	//req.Header 是引用类型，可以直接修改
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(headers))
	return ctx, span
}
func SpanFromRequest(ctx context.Context, req *http.Request) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	newCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(req.Header))
	req = req.WithContext(newCtx)
	return newCtx
}

// TracerFromContext 从上下文中获取 Tracer
func TracerFromContext(ctx context.Context, traceName string) (newCtx context.Context, tracer trace.Tracer) {
	if ctx == nil {
		ctx = context.Background()
	}
	if traceName == "" {
		traceName = GetTraceConfig().getTracerName()
	}
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		//检测span是否是nonTracer
		//trace.nonRecordingSpan
		if span.IsRecording() {

		}
		tracer = span.TracerProvider().Tracer(traceName)
	} else {
		tracer = otel.Tracer(traceName)
	}
	return ctx, tracer
}

func SetErrorTag(span trace.Span, err error) {
	if span == nil || err == nil {
		return
	}
	SetTags(span, map[string]any{
		"error":    true,
		"errormsg": err.Error(),
	})
	span.SetStatus(codes.Error, err.Error())
	span.RecordError(err)
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

func formatSpanName(spanName string) string {
	if len(spanName) > maxSpanNameLength {
		spanName = spanName[:maxSpanNameLength] + "..."
	}
	return spanName
}

// TraceId 获取traceId
func TraceId(ctx context.Context) string {
	return httputil.TraceId(ctx)
}

// StartSpan 创建span
func StartSpan(ctx context.Context, spanMethod string, spanName string) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}

	_, ok := TraceProvider()
	if !ok {
		span := trace.SpanFromContext(ctx)
		return ctx, span
	}

	newSpanName := formatSpanName(spanName)
	if spanMethod != "" {
		newSpanName = fmt.Sprintf("%s %s", strings.ToUpper(spanMethod), newSpanName)
	}

	newCtx, tracer := TracerFromContext(ctx, "github.com/magic-lib/go-servicekit/tracer")
	newCtx, span := tracer.Start(newCtx, newSpanName, trace.WithSpanKind(trace.SpanKindClient))
	SetTags(span, map[string]any{
		"span_name": spanName,
	})
	return newCtx, span
}

// SpanWithError 设置span的错误tag
func SpanWithError(ctx context.Context, err error) trace.Span {
	if ctx == nil {
		ctx = context.Background()
	}
	span := trace.SpanFromContext(ctx)
	_, ok := TraceProvider()
	if !ok {
		return span
	}
	SetErrorTag(span, err)
	return span
}
