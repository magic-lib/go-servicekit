package tracer

import (
	"github.com/magic-lib/go-plat-utils/utils/httputil/param"
	"github.com/samber/lo"
	"github.com/zeromicro/go-zero/core/service"
	ztrace "github.com/zeromicro/go-zero/core/trace"
	"github.com/zeromicro/go-zero/rest"
	"net/http"
)

// InitGoZeroTracing 初始化gozeroTracing,避免后续更新会被 rest.MustNewServer 抢占的问题
func (hc *TraceConfig) InitGoZeroTracing(srvConfig *service.ServiceConf) error {
	_, err := hc.InitTrace()
	if err != nil {
		return err
	}
	tc := GetTraceConfig()
	if tc.Endpoint == "" {
		return nil
	}
	srvConfig.Telemetry.Name = tc.getTracerName()
	srvConfig.Telemetry.Endpoint = tc.Endpoint
	srvConfig.Telemetry.Sampler = tc.getSampler()
	srvConfig.Telemetry.Batcher = tc.Batcher
	srvConfig.Telemetry.OtlpHeaders = tc.OtlpHeaders
	srvConfig.Telemetry.OtlpHttpPath = tc.OtlpHttpPath
	srvConfig.Telemetry.OtlpHttpSecure = tc.OtlpHttpSecure
	srvConfig.Telemetry.Disabled = false
	ztrace.StartAgent(srvConfig.Telemetry)
	return nil
}

func (hc *TraceConfig) GoZeroMiddleware(serv *rest.Server, pc ...*param.PathConfig) func(next http.HandlerFunc) http.HandlerFunc {
	err := hc.checkConfig()
	if err == nil {
		ztrace.StartAgent(ztrace.Config{
			Name:     hc.getTracerName(),
			Endpoint: hc.Endpoint,
			Batcher:  hc.Batcher,
			Sampler:  hc.getSampler(),
		})
		hc.useGoZero = true
	}
	return hc.commMiddleware(func(r *http.Request) (string, string) {
		routesList := serv.Routes()
		currentRouteList := make([]*param.CurrentRoute, 0)
		lo.ForEach(routesList, func(route rest.Route, index int) {
			currentRouteList = append(currentRouteList, &param.CurrentRoute{
				Method: route.Method,
				Path:   route.Path,
			})
		})
		oneRoute, ok := param.FindMatchingRoute(r, currentRouteList, pc...)
		if ok {
			return "", defaultSpanNameFormatter(oneRoute.Method, oneRoute.Path)
		}
		return "", ""
	})
}
