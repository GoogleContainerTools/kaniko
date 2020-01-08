package common

import (
	"crypto/tls"
	"net"
	"strings"
	"time"

	"github.com/docker/swarmkit/api"
	"github.com/docker/swarmkit/xnet"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Dial establishes a connection and creates a client.
// It infers connection parameters from CLI options.
func Dial(cmd *cobra.Command) (api.ControlClient, error) {
	conn, err := DialConn(cmd)
	if err != nil {
		return nil, err
	}

	return api.NewControlClient(conn), nil
}

// DialConn establishes a connection to SwarmKit.
func DialConn(cmd *cobra.Command) (*grpc.ClientConn, error) {
	addr, err := cmd.Flags().GetString("socket")
	if err != nil {
		return nil, err
	}

	opts := []grpc.DialOption{}
	insecureCreds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	opts = append(opts, grpc.WithTransportCredentials(insecureCreds))
	opts = append(opts, grpc.WithDialer(
		func(addr string, timeout time.Duration) (net.Conn, error) {
			return xnet.DialTimeoutLocal(addr, timeout)
		}))
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// Context returns a request context based on CLI arguments.
func Context(cmd *cobra.Command) context.Context {
	// TODO(aluzzardi): Actually create a context.
	return context.TODO()
}

// ParseLogDriverFlags parses a silly string format for log driver and options.
// Fully baked log driver config should be returned.
//
// If no log driver is available, nil, nil will be returned.
func ParseLogDriverFlags(flags *pflag.FlagSet) (*api.Driver, error) {
	if !flags.Changed("log-driver") {
		return nil, nil
	}

	name, err := flags.GetString("log-driver")
	if err != nil {
		return nil, err
	}

	var opts map[string]string
	if flags.Changed("log-opt") {
		rawOpts, err := flags.GetStringSlice("log-opt")
		if err != nil {
			return nil, err
		}

		opts = make(map[string]string, len(rawOpts))
		for _, rawOpt := range rawOpts {
			parts := strings.SplitN(rawOpt, "=", 2)
			if len(parts) == 1 {
				opts[parts[0]] = ""
				continue
			}

			opts[parts[0]] = parts[1]
		}
	}

	return &api.Driver{
		Name:    name,
		Options: opts,
	}, nil
}
