package commands

import (
	"io/ioutil"
	"log"

	"github.com/containerd/continuity"
	"github.com/spf13/cobra"
)

var ApplyCmd = &cobra.Command{
	Use:   "apply <root> [<manifest>]",
	Short: "Apply the manifest to the provided root",
	Run: func(cmd *cobra.Command, args []string) {
		root, path := args[0], args[1]

		p, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatalf("error reading manifest: %v", err)
		}

		m, err := continuity.Unmarshal(p)
		if err != nil {
			log.Fatalf("error unmarshaling manifest: %v", err)
		}

		ctx, err := continuity.NewContext(root)
		if err != nil {
			log.Fatalf("error getting context: %v", err)
		}

		if err := continuity.ApplyManifest(ctx, m); err != nil {
			log.Fatalf("error applying manifest: %v", err)
		}
	},
}
