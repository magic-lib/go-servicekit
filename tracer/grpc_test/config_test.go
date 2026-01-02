package example_test

import (
	"github.com/magic-lib/go-servicekit/tracer"
)

var tc = tracer.TraceConfig{
	Namespace:   "my-service",
	ServiceName: "my-grpc-service",
	Endpoint:    "http://202.60.228.31:14268/api/traces",
}
