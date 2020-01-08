package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/util/apicaps"
	"github.com/moby/buildkit/util/appdefaults"
	"github.com/moby/buildkit/util/profiler"
	"github.com/moby/buildkit/version"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func init() {
	apicaps.ExportedProduct = "buildkit"
}

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println(c.App.Name, version.Package, c.App.Version, version.Revision)
	}
	app := cli.NewApp()
	app.Name = "buildctl"
	app.Usage = "build utility"

	defaultAddress := os.Getenv("BUILDKIT_HOST")
	if defaultAddress == "" {
		defaultAddress = appdefaults.Address
	}

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug output in logs",
		},
		cli.StringFlag{
			Name:  "addr",
			Usage: "buildkitd address",
			Value: defaultAddress,
		},
		cli.StringFlag{
			Name:  "tlsservername",
			Usage: "buildkitd server name for certificate validation",
			Value: "",
		},
		cli.StringFlag{
			Name:  "tlscacert",
			Usage: "CA certificate for validation",
			Value: "",
		},
		cli.StringFlag{
			Name:  "tlscert",
			Usage: "client certificate",
			Value: "",
		},
		cli.StringFlag{
			Name:  "tlskey",
			Usage: "client key",
			Value: "",
		},
		cli.IntFlag{
			Name:  "timeout",
			Usage: "timeout backend connection after value seconds",
			Value: 5,
		},
	}

	app.Commands = []cli.Command{
		diskUsageCommand,
		pruneCommand,
		buildCommand,
		debugCommand,
	}

	var debugEnabled bool

	app.Before = func(context *cli.Context) error {
		debugEnabled = context.GlobalBool("debug")
		if debugEnabled {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}

	attachAppContext(app)

	profiler.Attach(app)

	if err := app.Run(os.Args); err != nil {
		if debugEnabled {
			fmt.Fprintf(os.Stderr, "error: %+v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		os.Exit(1)
	}
}

func resolveClient(c *cli.Context) (*client.Client, error) {
	serverName := c.GlobalString("tlsservername")
	if serverName == "" {
		// guess servername as hostname of target address
		uri, err := url.Parse(c.GlobalString("addr"))
		if err != nil {
			return nil, err
		}
		serverName = uri.Hostname()
	}
	caCert := c.GlobalString("tlscacert")
	cert := c.GlobalString("tlscert")
	key := c.GlobalString("tlskey")

	opts := []client.ClientOpt{client.WithBlock()}

	ctx := commandContext(c)

	if span := opentracing.SpanFromContext(ctx); span != nil {
		opts = append(opts, client.WithTracer(span.Tracer()))
	}

	if caCert != "" || cert != "" || key != "" {
		opts = append(opts, client.WithCredentials(serverName, caCert, cert, key))
	}

	timeout := time.Duration(c.GlobalInt("timeout"))
	ctx, cancel := context.WithTimeout(ctx, timeout*time.Second)
	defer cancel()

	return client.New(ctx, c.GlobalString("addr"), opts...)
}
