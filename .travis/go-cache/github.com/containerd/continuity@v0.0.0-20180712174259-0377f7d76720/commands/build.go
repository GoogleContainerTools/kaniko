package commands

import (
	"log"
	"os"

	"github.com/containerd/continuity"
	"github.com/spf13/cobra"
)

var (
	buildCmdConfig struct {
		format string
	}

	BuildCmd = &cobra.Command{
		Use:   "build <root>",
		Short: "Build a manifest for the provided root",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				log.Fatalln("please specify a root")
			}

			ctx, err := continuity.NewContext(args[0])
			if err != nil {
				log.Fatalf("error creating path context: %v", err)
			}

			m, err := continuity.BuildManifest(ctx)
			if err != nil {
				log.Fatalf("error generating manifest: %v", err)
			}

			p, err := continuity.Marshal(m)
			if err != nil {
				log.Fatalf("error marshaling manifest: %v", err)
			}

			if _, err := os.Stdout.Write(p); err != nil {
				log.Fatalf("error writing to stdout: %v", err)
			}
		},
	}
)

func init() {
	BuildCmd.Flags().StringVar(&buildCmdConfig.format, "format", "pb", "specify the output format of the manifest")
}
