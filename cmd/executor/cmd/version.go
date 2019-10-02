package cmd

import (
  "fmt"

  "github.com/spf13/cobra"
  "github.com/GoogleContainerTools/kaniko/pkg/version"
)

func init() {
  RootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
  Use:   "version",
  Short: "Print the version number of kaniko",
  Run: func(cmd *cobra.Command, args []string) {
    fmt.Print("Kaniko version : ", version.Version())
  },
}