package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/containerd/containerd/pkg/seed"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/containerd/sys"
	"github.com/docker/go-connections/sockets"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	registryremotecache "github.com/moby/buildkit/cache/remotecache/registry"
	"github.com/moby/buildkit/control"
	"github.com/moby/buildkit/frontend"
	dockerfile "github.com/moby/buildkit/frontend/dockerfile/builder"
	"github.com/moby/buildkit/frontend/gateway"
	"github.com/moby/buildkit/frontend/gateway/forwarder"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/solver/boltdbcachestorage"
	"github.com/moby/buildkit/util/apicaps"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/moby/buildkit/util/appdefaults"
	"github.com/moby/buildkit/util/profiler"
	"github.com/moby/buildkit/version"
	"github.com/moby/buildkit/worker"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/opencontainers/runc/libcontainer/system"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func init() {
	apicaps.ExportedProduct = "buildkit"
	seed.WithTimeAndRand()
}

type workerInitializerOpt struct {
	sessionManager *session.Manager
	root           string
}

type workerInitializer struct {
	fn func(c *cli.Context, common workerInitializerOpt) ([]worker.Worker, error)
	// less priority number, more preferred
	priority int
}

var (
	appFlags           []cli.Flag
	workerInitializers []workerInitializer
)

func registerWorkerInitializer(wi workerInitializer, flags ...cli.Flag) {
	workerInitializers = append(workerInitializers, wi)
	sort.Slice(workerInitializers,
		func(i, j int) bool {
			return workerInitializers[i].priority < workerInitializers[j].priority
		})
	appFlags = append(appFlags, flags...)
}

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println(c.App.Name, version.Package, c.App.Version, version.Revision)
	}
	app := cli.NewApp()
	app.Name = "buildkitd"
	app.Usage = "build daemon"
	defaultRoot := appdefaults.Root
	defaultAddress := appdefaults.Address
	rootlessUsage := "set all the default options to be compatible with rootless containers"
	if system.RunningInUserNS() {
		app.Flags = append(app.Flags, cli.BoolTFlag{
			Name:  "rootless",
			Usage: rootlessUsage + " (default: true)",
		})
		// if buildkitd is being executed as the mapped-root (not only EUID==0 but also $USER==root)
		// in a user namespace, we need to enable the rootless mode but
		// we don't want to honor $HOME for setting up default paths.
		if u := os.Getenv("USER"); u != "" && u != "root" {
			defaultRoot = appdefaults.UserRoot()
			defaultAddress = appdefaults.UserAddress()
			appdefaults.EnsureUserAddressDir()
		}
	} else {
		app.Flags = append(app.Flags, cli.BoolFlag{
			Name:  "rootless",
			Usage: rootlessUsage,
		})
	}
	app.Flags = append(app.Flags,
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug output in logs",
		},
		cli.StringFlag{
			Name:  "root",
			Usage: "path to state directory",
			Value: defaultRoot,
		},
		cli.StringSliceFlag{
			Name:  "addr",
			Usage: "listening address (socket or tcp)",
			Value: &cli.StringSlice{defaultAddress},
		},
		cli.StringFlag{
			Name:  "group",
			Usage: "group (name or gid) which will own all Unix socket listening addresses",
			Value: "",
		},
		cli.StringFlag{
			Name:  "debugaddr",
			Usage: "debugging address (eg. 0.0.0.0:6060)",
			Value: "",
		},
		cli.StringFlag{
			Name:  "tlscert",
			Usage: "certificate file to use",
		},
		cli.StringFlag{
			Name:  "tlskey",
			Usage: "key file to use",
		},
		cli.StringFlag{
			Name:  "tlscacert",
			Usage: "ca certificate to verify clients",
		},
	)
	app.Flags = append(app.Flags, appFlags...)

	app.Action = func(c *cli.Context) error {
		if os.Geteuid() != 0 {
			return errors.New("rootless mode requires to be executed as the mapped root in a user namespace; you may use RootlessKit for setting up the namespace")
		}
		ctx, cancel := context.WithCancel(appcontext.Context())
		defer cancel()

		if debugAddr := c.GlobalString("debugaddr"); debugAddr != "" {
			if err := setupDebugHandlers(debugAddr); err != nil {
				return err
			}
		}
		opts := []grpc.ServerOption{unaryInterceptor(ctx), grpc.StreamInterceptor(otgrpc.OpenTracingStreamServerInterceptor(tracer))}
		creds, err := serverCredentials(c)
		if err != nil {
			return err
		}
		if creds != nil {
			opts = append(opts, creds)
		}
		server := grpc.NewServer(opts...)

		// relative path does not work with nightlyone/lockfile
		root, err := filepath.Abs(c.GlobalString("root"))
		if err != nil {
			return err
		}

		if err := os.MkdirAll(root, 0700); err != nil {
			return errors.Wrapf(err, "failed to create %s", root)
		}

		controller, err := newController(c, root)
		if err != nil {
			return err
		}

		controller.Register(server)

		errCh := make(chan error, 1)
		addrs := c.GlobalStringSlice("addr")
		if len(addrs) > 1 {
			addrs = addrs[1:] // https://github.com/urfave/cli/issues/160
		}
		if err := serveGRPC(c, server, addrs, errCh); err != nil {
			return err
		}

		select {
		case serverErr := <-errCh:
			err = serverErr
			cancel()
		case <-ctx.Done():
			err = ctx.Err()
		}

		logrus.Infof("stopping server")
		server.GracefulStop()

		return err
	}
	app.Before = func(context *cli.Context) error {
		if context.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}

	app.After = func(context *cli.Context) error {
		if closeTracer != nil {
			return closeTracer.Close()
		}
		return nil
	}

	profiler.Attach(app)

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "buildkitd: %s\n", err)
		os.Exit(1)
	}
}

func serveGRPC(c *cli.Context, server *grpc.Server, addrs []string, errCh chan error) error {
	if len(addrs) == 0 {
		return errors.New("--addr cannot be empty")
	}
	eg, _ := errgroup.WithContext(context.Background())
	listeners := make([]net.Listener, 0, len(addrs))
	for _, addr := range addrs {
		l, err := getListener(c, addr)
		if err != nil {
			for _, l := range listeners {
				l.Close()
			}
			return err
		}
		listeners = append(listeners, l)
	}
	for _, l := range listeners {
		func(l net.Listener) {
			eg.Go(func() error {
				defer l.Close()
				logrus.Infof("running server on %s", l.Addr())
				return server.Serve(l)
			})
		}(l)
	}
	go func() {
		errCh <- eg.Wait()
	}()
	return nil
}

// Convert a string containing either a group name or a stringified gid into a numeric id)
func groupToGid(group string) (int, error) {
	if group == "" {
		return os.Getgid(), nil
	}

	var (
		err error
		id  int
	)

	// Try and parse as a number, if the error is ErrSyntax
	// (i.e. its not a number) then we carry on and try it as a
	// name.
	if id, err = strconv.Atoi(group); err == nil {
		return id, nil
	} else if err.(*strconv.NumError).Err != strconv.ErrSyntax {
		return 0, err
	}

	ginfo, err := user.LookupGroup(group)
	if err != nil {
		return 0, err
	}
	group = ginfo.Gid

	if id, err = strconv.Atoi(group); err != nil {
		return 0, err
	}

	return id, nil
}

func getListener(c *cli.Context, addr string) (net.Listener, error) {
	addrSlice := strings.SplitN(addr, "://", 2)
	if len(addrSlice) < 2 {
		return nil, errors.Errorf("address %s does not contain proto, you meant unix://%s ?",
			addr, addr)
	}
	proto := addrSlice[0]
	listenAddr := addrSlice[1]
	switch proto {
	case "unix", "npipe":
		uid := os.Getuid()
		gid, err := groupToGid(c.GlobalString("group"))
		if err != nil {
			return nil, err
		}
		return sys.GetLocalListener(listenAddr, uid, gid)
	case "tcp":
		return sockets.NewTCPSocket(listenAddr, nil)
	default:
		return nil, errors.Errorf("addr %s not supported", addr)
	}
}

func unaryInterceptor(globalCtx context.Context) grpc.ServerOption {
	withTrace := otgrpc.OpenTracingServerInterceptor(tracer, otgrpc.LogPayloads())

	return grpc.UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		go func() {
			select {
			case <-ctx.Done():
			case <-globalCtx.Done():
				cancel()
			}
		}()

		resp, err = withTrace(ctx, req, info, handler)
		if err != nil {
			logrus.Errorf("%s returned error: %+v", info.FullMethod, err)
		}
		return
	})
}

func serverCredentials(c *cli.Context) (grpc.ServerOption, error) {
	certFile := c.GlobalString("tlscert")
	keyFile := c.GlobalString("tlskey")
	caFile := c.GlobalString("tlscacert")
	if certFile == "" && keyFile == "" {
		return nil, nil
	}
	err := errors.New("you must specify key and cert file if one is specified")
	if certFile == "" {
		return nil, err
	}
	if keyFile == "" {
		return nil, err
	}
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not load server key pair")
	}
	tlsConf := &tls.Config{
		Certificates: []tls.Certificate{certificate},
	}
	if caFile != "" {
		certPool := x509.NewCertPool()
		ca, err := ioutil.ReadFile(caFile)
		if err != nil {
			return nil, errors.Wrap(err, "could not read ca certificate")
		}
		// Append the client certificates from the CA
		if ok := certPool.AppendCertsFromPEM(ca); !ok {
			return nil, errors.New("failed to append ca cert")
		}
		tlsConf.ClientAuth = tls.RequireAndVerifyClientCert
		tlsConf.ClientCAs = certPool
	}
	creds := grpc.Creds(credentials.NewTLS(tlsConf))
	return creds, nil
}

func newController(c *cli.Context, root string) (*control.Controller, error) {
	sessionManager, err := session.NewManager()
	if err != nil {
		return nil, err
	}
	wc, err := newWorkerController(c, workerInitializerOpt{
		sessionManager: sessionManager,
		root:           root,
	})
	if err != nil {
		return nil, err
	}
	frontends := map[string]frontend.Frontend{}
	frontends["dockerfile.v0"] = forwarder.NewGatewayForwarder(wc, dockerfile.Build)
	frontends["gateway.v0"] = gateway.NewGatewayFrontend(wc)

	cacheStorage, err := boltdbcachestorage.NewStore(filepath.Join(root, "cache.db"))
	if err != nil {
		return nil, err
	}

	return control.NewController(control.Opt{
		SessionManager:   sessionManager,
		WorkerController: wc,
		Frontends:        frontends,
		// TODO: support non-registry remote cache
		ResolveCacheExporterFunc: registryremotecache.ResolveCacheExporterFunc(sessionManager),
		ResolveCacheImporterFunc: registryremotecache.ResolveCacheImporterFunc(sessionManager),
		CacheKeyStorage:          cacheStorage,
	})
}

func newWorkerController(c *cli.Context, wiOpt workerInitializerOpt) (*worker.Controller, error) {
	wc := &worker.Controller{}
	nWorkers := 0
	for _, wi := range workerInitializers {
		ws, err := wi.fn(c, wiOpt)
		if err != nil {
			return nil, err
		}
		for _, w := range ws {
			logrus.Infof("found worker %q, labels=%v, platforms=%v", w.ID(), w.Labels(), formatPlatforms(w.Platforms()))
			if err = wc.Add(w); err != nil {
				return nil, err
			}
			nWorkers++
		}
	}
	if nWorkers == 0 {
		return nil, errors.New("no worker found, rebuild the buildkit daemon?")
	}
	defaultWorker, err := wc.GetDefault()
	if err != nil {
		return nil, err
	}
	logrus.Infof("found %d workers, default=%q", nWorkers, defaultWorker.ID())
	logrus.Warn("currently, only the default worker can be used.")
	return wc, nil
}

func attrMap(sl []string) (map[string]string, error) {
	m := map[string]string{}
	for _, v := range sl {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return nil, errors.Errorf("invalid value %s", v)
		}
		m[parts[0]] = parts[1]
	}
	return m, nil
}

func formatPlatforms(p []specs.Platform) []string {
	str := make([]string, 0, len(p))
	for _, pp := range p {
		str = append(str, platforms.Format(platforms.Normalize(pp)))
	}
	return str
}

func parsePlatforms(platformsStr []string) ([]specs.Platform, error) {
	out := make([]specs.Platform, 0, len(platformsStr))
	for _, s := range platformsStr {
		p, err := platforms.Parse(s)
		if err != nil {
			return nil, err
		}
		out = append(out, platforms.Normalize(p))
	}
	return out, nil
}
