package tracer

import (
	"github.com/gin-gonic/gin"
	"github.com/magic-lib/go-plat-utils/utils/httputil/param"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"net/http"
	"slices"
	"strings"
)

var defaultSpanNameFormatter = func(method string, path string) string {
	method = strings.ToUpper(method)
	if !slices.Contains([]string{
		http.MethodGet, http.MethodHead,
		http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete,
		http.MethodConnect, http.MethodOptions,
		http.MethodTrace,
	}, method) {
		method = "HTTP"
	}

	if path != "" {
		return method + " " + path
	}
	return method
}

func (hc *TraceConfig) commMiddleware(fun func(r *http.Request) (tracName string, spanName string)) func(next http.HandlerFunc) http.HandlerFunc {
	err := hc.checkConfig()
	if err != nil {
		return func(next http.HandlerFunc) http.HandlerFunc {
			return next
		}
	}
	if fun == nil {
		fun = func(r *http.Request) (string, string) {
			return "", ""
		}
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := SpanFromRequest(r.Context(), r)

			traceName, spanName := fun(r)

			if traceName == "" {
				traceName = hc.ServiceName
			}
			if spanName == "" {
				spanName = defaultSpanNameFormatter(r.Method, r.URL.Path)
			}

			ctx, span := hc.StartSpan(ctx, spanName, traceName)
			defer span.End()

			SetTags(span, map[string]any{
				"service":                 hc.ServiceName,
				"http.request.method":     r.Method,
				"http.response.body.size": r.ContentLength,
				"http.url":                r.URL,
				"client.address":          param.ClientIP(r),
				"url.scheme":              r.URL.Scheme,
				"url.path":                r.URL.Path,
				"url.host":                r.URL.Host,
			})
			r = r.WithContext(ctx)
			next(w, r)
		}
	}
}
func (hc *TraceConfig) HttpMiddleware(fun func(r *http.Request) (tracName string, spanName string), next http.Handler) http.Handler {
	middleware := hc.commMiddleware(fun)
	return middleware(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func (hc *TraceConfig) GinMiddleware(opts ...otelgin.Option) gin.HandlerFunc {
	err := hc.checkConfig()
	if err != nil {
		return nil
	}
	optList := make([]otelgin.Option, 0)
	if len(opts) > 0 {
		optList = append(optList, opts...)
	}

	optList = append(optList, otelgin.WithTracerProvider(otel.GetTracerProvider()))
	optList = append(optList, otelgin.WithPropagators(otel.GetTextMapPropagator()))
	optList = append(optList, otelgin.WithSpanNameFormatter(func(c *gin.Context) string {
		return defaultSpanNameFormatter(c.Request.Method, c.FullPath())
	}))

	hf := otelgin.Middleware(hc.getTracerName(), optList...)
	return hf
}
