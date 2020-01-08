package main

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/moby/buildkit/util/appcontext"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	jaeger "github.com/uber/jaeger-client-go"
	"github.com/urfave/cli"
)

func getTracer() (opentracing.Tracer, io.Closer) {
	if traceAddr := os.Getenv("JAEGER_TRACE"); traceAddr != "" {
		tr, err := jaeger.NewUDPTransport(traceAddr, 0)
		if err != nil {
			panic(err)
		}

		// metricsFactory := prometheus.New()
		return jaeger.NewTracer(
			"buildctl",
			jaeger.NewConstSampler(true),
			jaeger.NewRemoteReporter(tr),
		)
	}

	return opentracing.NoopTracer{}, &nopCloser{}
}

func attachAppContext(app *cli.App) {
	ctx := appcontext.Context()

	tracer, closer := getTracer()

	var span opentracing.Span

	for i, cmd := range app.Commands {
		func(before cli.BeforeFunc) {
			name := cmd.Name
			app.Commands[i].Before = func(clicontext *cli.Context) error {
				if before != nil {
					if err := before(clicontext); err != nil {
						return err
					}
				}

				span = tracer.StartSpan(name)
				span.LogFields(log.String("command", strings.Join(os.Args, " ")))

				ctx = opentracing.ContextWithSpan(ctx, span)

				clicontext.App.Metadata["context"] = ctx
				return nil
			}
		}(cmd.Before)
	}

	app.ExitErrHandler = func(clicontext *cli.Context, err error) {
		if span != nil {
			ext.Error.Set(span, true)
		}
		cli.HandleExitCoder(err)
	}

	after := app.After
	app.After = func(clicontext *cli.Context) error {
		if after != nil {
			if err := after(clicontext); err != nil {
				return err
			}
		}
		if span != nil {
			span.Finish()
		}
		return closer.Close()
	}

}

func commandContext(c *cli.Context) context.Context {
	return c.App.Metadata["context"].(context.Context)
}

type nopCloser struct {
}

func (*nopCloser) Close() error {
	return nil
}
