package main

import (
	"errors"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

var (
	mainCmd = &cobra.Command{
		Use:   os.Args[0],
		Short: "Benchmark swarm",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			count, err := cmd.Flags().GetUint64("count")
			if err != nil {
				return err
			}
			if count == 0 {
				return errors.New("--count is mandatory")
			}
			manager, err := cmd.Flags().GetString("manager")
			if err != nil {
				return err
			}
			port, err := cmd.Flags().GetInt("port")
			if err != nil {
				return err
			}
			ip, err := cmd.Flags().GetString("ip")
			if err != nil {
				return err
			}

			b := NewBenchmark(&Config{
				Count:   count,
				Manager: manager,
				IP:      ip,
				Port:    port,
				Unit:    time.Second,
			})
			return b.Run(ctx)
		},
	}
)

func init() {
	mainCmd.Flags().Int64P("count", "c", 0, "Number of tasks to start for the benchmarking session")
	mainCmd.Flags().StringP("manager", "m", "localhost:4242", "Specify the manager address")
	mainCmd.Flags().IntP("port", "p", 2222, "Port used by the benchmark for listening")
	mainCmd.Flags().StringP("ip", "i", "127.0.0.1", "IP of the benchmarking tool. Tasks will phone home to this address")
}

func main() {
	if err := mainCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
