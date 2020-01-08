package cmd

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// Version can be set via:
// -ldflags="-X 'github.com/google/go-containerregistry/pkg/crane.Version=$TAG'"
var Version string

func init() { Root.AddCommand(NewCmdVersion()) }

// NewCmdVersion creates a new cobra.Command for the version subcommand.
func NewCmdVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			if Version == "" {
				// If Version is unset, use the current commit.
				hash, err := exec.Command("git", "rev-parse", "HEAD").Output()
				if err != nil {
					log.Fatalf("error parsing git commit: %v", err)
				}
				Version = strings.TrimSpace(string(hash))
			}
			fmt.Println(Version)
		},
	}
}
