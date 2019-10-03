package cmd

import (
	"fmt"

	"github.com/GoogleContainerTools/kaniko/pkg/version"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of kaniko",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Kaniko version : ", version.Version())
	},
}
