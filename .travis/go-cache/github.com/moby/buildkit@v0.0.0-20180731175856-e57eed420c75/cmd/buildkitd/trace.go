package main

import (
	"io"
	"os"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
)

var tracer opentracing.Tracer
var closeTracer io.Closer

func init() {

	tracer = opentracing.NoopTracer{}

	if traceAddr := os.Getenv("JAEGER_TRACE"); traceAddr != "" {
		tr, err := jaeger.NewUDPTransport(traceAddr, 0)
		if err != nil {
			panic(err)
		}

		tracer, closeTracer = jaeger.NewTracer(
			"buildkitd",
			jaeger.NewConstSampler(true),
			jaeger.NewRemoteReporter(tr),
		)
	}

}
