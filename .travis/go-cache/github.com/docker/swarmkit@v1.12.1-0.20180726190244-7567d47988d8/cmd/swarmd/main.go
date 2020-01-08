package main

import (
	_ "expvar"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"

	engineapi "github.com/docker/docker/client"
	"github.com/docker/swarmkit/agent/exec/dockerapi"
	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/api/genericresource"
	"github.com/docker/swarmkit/cli"
	"github.com/docker/swarmkit/cmd/swarmd/defaults"
	"github.com/docker/swarmkit/log"
	"github.com/docker/swarmkit/manager/encryption"
	"github.com/docker/swarmkit/node"
	"github.com/docker/swarmkit/version"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

var externalCAOpt cli.ExternalCAOpt

func main() {
	if err := mainCmd.Execute(); err != nil {
		log.L.Fatal(err)
	}
}

var (
	mainCmd = &cobra.Command{
		Use:          os.Args[0],
		Short:        "Run a swarm control process",
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			logrus.SetOutput(os.Stderr)
			flag, err := cmd.Flags().GetString("log-level")
			if err != nil {
				log.L.Fatal(err)
			}
			level, err := logrus.ParseLevel(flag)
			if err != nil {
				log.L.Fatal(err)
			}
			logrus.SetLevel(level)

			v, err := cmd.Flags().GetBool("version")
			if err != nil {
				log.L.Fatal(err)
			}
			if v {
				version.PrintVersion()
				os.Exit(0)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			hostname, err := cmd.Flags().GetString("hostname")
			if err != nil {
				return err
			}

			advertiseAddr, err := cmd.Flags().GetString("advertise-remote-api")
			if err != nil {
				return err
			}

			addr, err := cmd.Flags().GetString("listen-remote-api")
			if err != nil {
				return err
			}
			addrHost, _, err := net.SplitHostPort(addr)
			if err == nil {
				ip := net.ParseIP(addrHost)
				if ip != nil && (ip.IsUnspecified() || ip.IsLoopback()) {
					fmt.Println("Warning: Specifying a valid address with --listen-remote-api may be necessary for other managers to reach this one.")
				}
			}

			unix, err := cmd.Flags().GetString("listen-control-api")
			if err != nil {
				return err
			}

			metricsAddr, err := cmd.Flags().GetString("listen-metrics")
			if err != nil {
				return err
			}

			debugAddr, err := cmd.Flags().GetString("listen-debug")
			if err != nil {
				return err
			}

			managerAddr, err := cmd.Flags().GetString("join-addr")
			if err != nil {
				return err
			}

			forceNewCluster, err := cmd.Flags().GetBool("force-new-cluster")
			if err != nil {
				return err
			}

			hb, err := cmd.Flags().GetUint32("heartbeat-tick")
			if err != nil {
				return err
			}

			election, err := cmd.Flags().GetUint32("election-tick")
			if err != nil {
				return err
			}

			stateDir, err := cmd.Flags().GetString("state-dir")
			if err != nil {
				return err
			}

			joinToken, err := cmd.Flags().GetString("join-token")
			if err != nil {
				return err
			}

			engineAddr, err := cmd.Flags().GetString("engine-addr")
			if err != nil {
				return err
			}

			autolockManagers, err := cmd.Flags().GetBool("autolock")
			if err != nil {
				return err
			}

			var unlockKey []byte
			if cmd.Flags().Changed("unlock-key") {
				unlockKeyString, err := cmd.Flags().GetString("unlock-key")
				if err != nil {
					return err
				}
				unlockKey, err = encryption.ParseHumanReadableKey(unlockKeyString)
				if err != nil {
					return err
				}
			}

			var resources []*api.GenericResource
			if cmd.Flags().Changed("generic-node-resources") {
				genericResources, err := cmd.Flags().GetString("generic-node-resources")
				if err != nil {
					return err
				}
				resources, err = genericresource.ParseCmd(genericResources)
				if err != nil {
					return err
				}
			}

			// Create a cancellable context for our GRPC call
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			if err := os.MkdirAll(stateDir, 0700); err != nil {
				return err
			}

			client, err := engineapi.NewClient(engineAddr, "", nil, nil)
			if err != nil {
				return err
			}

			executor := dockerapi.NewExecutor(client, resources)

			if debugAddr != "" {
				go func() {
					// setup listening to give access to pprof, expvar, etc.
					if err := http.ListenAndServe(debugAddr, nil); err != nil {
						panic(err)
					}
				}()
			}

			if metricsAddr != "" {
				// This allows to measure latency distribution.
				grpc_prometheus.EnableHandlingTimeHistogram()

				l, err := net.Listen("tcp", metricsAddr)
				if err != nil {
					panic(err)
				}
				mux := http.NewServeMux()
				mux.Handle("/metrics", prometheus.Handler())

				go func() {
					if err := http.Serve(l, mux); err != nil {
						logrus.Errorf("serve metrics api: %s", err)
					}
				}()
			}

			n, err := node.New(&node.Config{
				Hostname:           hostname,
				ForceNewCluster:    forceNewCluster,
				ListenControlAPI:   unix,
				ListenRemoteAPI:    addr,
				AdvertiseRemoteAPI: advertiseAddr,
				JoinAddr:           managerAddr,
				StateDir:           stateDir,
				JoinToken:          joinToken,
				ExternalCAs:        externalCAOpt.Value(),
				Executor:           executor,
				HeartbeatTick:      hb,
				ElectionTick:       election,
				AutoLockManagers:   autolockManagers,
				UnlockKey:          unlockKey,
			})
			if err != nil {
				return err
			}

			if err := n.Start(ctx); err != nil {
				return err
			}

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			go func() {
				<-c
				n.Stop(ctx)
			}()

			go func() {
				select {
				case <-n.Ready():
				case <-ctx.Done():
				}
				if ctx.Err() == nil {
					logrus.Info("node is ready")
				}
			}()

			return n.Err(ctx)
		},
	}
)

func init() {
	mainCmd.Flags().BoolP("version", "v", false, "Display the version and exit")
	mainCmd.Flags().StringP("log-level", "l", "info", "Log level (options \"debug\", \"info\", \"warn\", \"error\", \"fatal\", \"panic\")")
	mainCmd.Flags().StringP("state-dir", "d", defaults.StateDir, "State directory")
	mainCmd.Flags().StringP("join-token", "", "", "Specifies the secret token required to join the cluster")
	mainCmd.Flags().String("engine-addr", "unix:///var/run/docker.sock", "Address of engine instance of agent.")
	mainCmd.Flags().String("hostname", "", "Override reported agent hostname")
	mainCmd.Flags().String("advertise-remote-api", "", "Advertise address for remote API")
	mainCmd.Flags().String("listen-remote-api", "0.0.0.0:4242", "Listen address for remote API")
	mainCmd.Flags().String("listen-control-api", defaults.ControlAPISocket, "Listen socket for control API")
	mainCmd.Flags().String("listen-debug", "", "Bind the Go debug server on the provided address")
	mainCmd.Flags().String("listen-metrics", "", "Listen address for metrics")
	mainCmd.Flags().String("join-addr", "", "Join cluster with a node at this address")
	mainCmd.Flags().String("generic-node-resources", "", "user defined resources (e.g. fpga=2,gpu=UUID1,gpu=UUID2,gpu=UUID3)")
	mainCmd.Flags().Bool("force-new-cluster", false, "Force the creation of a new cluster from data directory")
	mainCmd.Flags().Uint32("heartbeat-tick", 1, "Defines the heartbeat interval (in seconds) for raft member health-check")
	mainCmd.Flags().Uint32("election-tick", 10, "Defines the amount of ticks (in seconds) needed without a Leader to trigger a new election")
	mainCmd.Flags().Var(&externalCAOpt, "external-ca", "Specifications of one or more certificate signing endpoints")
	mainCmd.Flags().Bool("autolock", false, "Require an unlock key in order to start a manager once it's been stopped")
	mainCmd.Flags().String("unlock-key", "", "Unlock this manager using this key")
}
