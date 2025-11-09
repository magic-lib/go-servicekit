package tracer

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

func (hc *TraceConfig) GrpcMiddleware() (grpc.ServerOption, grpc.DialOption) {
	return grpc.StatsHandler(otelgrpc.NewServerHandler()), grpc.WithStatsHandler(otelgrpc.NewClientHandler())
}
