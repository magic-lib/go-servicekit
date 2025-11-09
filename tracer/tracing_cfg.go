package tracer

import (
	"context"
	"fmt"
	"github.com/magic-lib/go-plat-utils/goroutines"
	ztrace "github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"log"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
)

type TraceConfig struct {
	Namespace      string            `json:"namespace"`   //整个项目命名
	ServiceName    string            `json:"serviceName"` //各个微服务命名
	Endpoint       string            `json:"endpoint"`    //接入点
	Batcher        string            `json:",default=jaeger,options=jaeger|zipkin|otlpgrpc|otlphttp|file"`
	OtlpHeaders    map[string]string `json:",optional"`
	OtlpHttpPath   string            `json:",optional"`
	OtlpHttpSecure bool              `json:",optional"`
	SamplerPercent int               `json:",samplerPercent"` //采样比例
	lastTracer     trace.Tracer      //上次使用过的tracer
	useGoZero      bool
}

var (
	traceProviderCache    = make(map[string]*sdktrace.TracerProvider)
	singleton             = goroutines.NewSingleFlight()
	globalTraceConfigLock sync.Mutex
	globalTraceConfig     *atomic.Value
)

type tracerConfigHolder struct {
	tc *TraceConfig
}

func (hc *TraceConfig) getTracerName() string {
	err := hc.checkConfig()
	if err != nil {
		return hc.ServiceName
	}
	return fmt.Sprintf("%s.%s", hc.Namespace, hc.ServiceName)
}
func (hc *TraceConfig) getSampler() float64 {
	if hc.SamplerPercent == 0 { //默认为50%概率
		hc.SamplerPercent = 50
	}
	if hc.SamplerPercent > 100 {
		hc.SamplerPercent = 100
	}
	if hc.SamplerPercent < 0 {
		hc.SamplerPercent = 0
	}
	return float64(hc.SamplerPercent) / 100
}

func (hc *TraceConfig) checkConfig() error {
	if hc.Batcher == "" {
		hc.Batcher = kindJaeger
	}
	if hc.Namespace == "" {
		return fmt.Errorf("trace Namespace is required")
	}
	_ = hc.getSampler()

	if hc.ServiceName == "" {
		log.Println("TraceConfig serviceName is required")
		return fmt.Errorf("serviceName is required")
	}
	if hc.Endpoint == "" {
		log.Println("TraceConfig endpoint is required")
		return fmt.Errorf("endpoint is required")
	}
	return nil
}
func (hc *TraceConfig) startTraceSpan(ctx context.Context, traceName, spanName string) (context.Context, trace.Span) {
	newTraceCtx, tracer := TracerFromContext(ctx, traceName)
	span := trace.SpanFromContext(ctx)
	options := make([]trace.SpanStartOption, 0)
	if span == nil || !span.SpanContext().IsValid() {
		options = []trace.SpanStartOption{
			trace.WithNewRoot(),
		}
	}
	newCtx, span := tracer.Start(newTraceCtx, spanName, options...)
	newCtx = trace.ContextWithSpan(newCtx, span)
	hc.lastTracer = tracer
	return newCtx, span
}

// 没有tranceName的情况下，使用上次使用过的trance
func (hc *TraceConfig) startNewSpan(ctx context.Context, spanName string) (context.Context, trace.Span) {
	if hc.lastTracer == nil {
		return hc.startTraceSpan(ctx, hc.ServiceName, spanName)
	}
	parentSpan := trace.SpanFromContext(ctx)
	if parentSpan != nil && parentSpan.SpanContext().IsValid() {
		ctx = trace.ContextWithSpan(ctx, parentSpan)
	}
	return hc.lastTracer.Start(ctx, spanName)
}

func (hc *TraceConfig) initTraceProvider() (*sdktrace.TracerProvider, error) {
	if tpTemp, ok := TraceProvider(); ok {
		return tpTemp, nil
	}

	err := hc.checkConfig()
	if err != nil {
		return nil, err
	}

	backGroundCtx := context.Background()

	// 定义服务资源，用于在 Jaeger UI 中识别服务
	res, err := resource.New(backGroundCtx,
		resource.WithAttributes(semconv.ServiceNameKey.String(hc.getTracerName())),
	)
	if err != nil {
		return nil, err
	}

	exporter, err := hc.createExporter()
	if err != nil {
		return nil, err
	}

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(hc.getSampler())),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	}

	// 创建 TracerProvider，配置批量处理器和资源
	newTp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(newTp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.Printf("[otel] error: %v", err)
	}))

	return newTp, nil
}

func (hc *TraceConfig) createExporter() (sdktrace.SpanExporter, error) {
	switch hc.Batcher {
	case kindJaeger:
		u, err := url.Parse(hc.Endpoint)
		if err == nil && u.Scheme == protocolUdp {
			return jaeger.New(jaeger.WithAgentEndpoint(jaeger.WithAgentHost(u.Hostname()),
				jaeger.WithAgentPort(u.Port())))
		}
		return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(hc.Endpoint)))
	case kindZipkin:
		return zipkin.New(hc.Endpoint)
	case kindOtlpGrpc:
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(hc.Endpoint),
		}
		if len(hc.OtlpHeaders) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(hc.OtlpHeaders))
		}
		return otlptracegrpc.New(context.Background(), opts...)
	case kindOtlpHttp:
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(hc.Endpoint),
		}

		if !hc.OtlpHttpSecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if len(hc.OtlpHeaders) > 0 {
			opts = append(opts, otlptracehttp.WithHeaders(hc.OtlpHeaders))
		}
		if len(hc.OtlpHttpPath) > 0 {
			opts = append(opts, otlptracehttp.WithURLPath(hc.OtlpHttpPath))
		}
		return otlptracehttp.New(context.Background(), opts...)
	case kindFile:
		f, err := os.OpenFile(hc.Endpoint, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("file exporter endpoint error: %s", err.Error())
		}
		return stdouttrace.New(stdouttrace.WithWriter(f))
	default:
		return nil, fmt.Errorf("unknown exporter: %s", hc.Batcher)
	}
}

// InitTrace 初始化，只用执行一次
func (hc *TraceConfig) InitTrace() (*sdktrace.TracerProvider, error) {
	if tp := traceProviderCache[hc.Endpoint]; tp != nil {
		return tp, nil
	}
	if globalTraceConfig == nil {
		SetTraceConfig(hc)
	}
	//一个名称只需要执行一次, 避免重复执行
	_, err := singleton.Once(hc.Endpoint, func() (any, error) {
		return hc.initTraceProvider()
	})
	if err != nil {
		return nil, err
	}
	if tpTemp, ok := TraceProvider(); ok {
		traceProviderCache[hc.Endpoint] = tpTemp
		return tpTemp, nil
	}
	return nil, nil
}

func (hc *TraceConfig) Stop() {
	tp, ok := TraceProvider()
	if ok {
		err := tp.Shutdown(context.Background())
		if err != nil {
			fmt.Printf("Failed to shutdown tracer provider: %v", err)
			return
		}
		delete(traceProviderCache, hc.Endpoint)
	}
	if hc.useGoZero {
		ztrace.StopAgent()
		hc.useGoZero = false
	}
}

func (hc *TraceConfig) StartSpan(ctx context.Context, spanName string, traceName ...string) (context.Context, trace.Span) {
	if len(traceName) == 0 || traceName[0] == "" {
		return hc.startNewSpan(ctx, spanName)
	}
	return hc.startTraceSpan(ctx, traceName[0], spanName)
}
