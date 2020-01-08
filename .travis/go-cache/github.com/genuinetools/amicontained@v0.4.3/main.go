package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/genuinetools/amicontained/container"
	"github.com/genuinetools/amicontained/version"
	"github.com/genuinetools/pkg/cli"
	"github.com/sirupsen/logrus"
)

var (
	debug bool
)

func main() {
	// Create a new cli program.
	p := cli.NewProgram()
	p.Name = "amicontained"
	p.Description = "A container introspection tool"

	// Set the GitCommit and Version.
	p.GitCommit = version.GITCOMMIT
	p.Version = version.VERSION

	// Setup the global flags.
	p.FlagSet = flag.NewFlagSet("ship", flag.ExitOnError)
	p.FlagSet.BoolVar(&debug, "d", false, "enable debug logging")

	// Set the before function.
	p.Before = func(ctx context.Context) error {
		// Set the log level.
		if debug {
			logrus.SetLevel(logrus.DebugLevel)
		}

		return nil
	}

	// Set the main program action.
	p.Action = func(ctx context.Context) error {
		// Container Runtime
		runtime, err := container.DetectRuntime()
		if err != nil && err != container.ErrContainerRuntimeNotFound {
			return err
		}
		fmt.Printf("Container Runtime: %s\n", runtime)

		// Namespaces
		namespaces := []string{"pid"}
		fmt.Println("Has Namespaces:")
		for _, namespace := range namespaces {
			ns, err := container.HasNamespace(namespace)
			if err != nil {
				fmt.Printf("\t%s: error -> %v\n", namespace, err)
				continue
			}
			fmt.Printf("\t%s: %t\n", namespace, ns)
		}

		// User Namespaces
		userNS, userMappings := container.UserNamespace()
		fmt.Printf("\tuser: %t\n", userNS)
		if len(userMappings) > 0 {
			fmt.Println("User Namespace Mappings:")
			for _, userMapping := range userMappings {
				fmt.Printf("\tContainer -> %d\tHost -> %d\tRange -> %d\n", userMapping.ContainerID, userMapping.HostID, userMapping.Range)
			}
		}

		// AppArmor Profile
		aaprof := container.AppArmorProfile()
		fmt.Printf("AppArmor Profile: %s\n", aaprof)

		// Capabilities
		caps, err := container.Capabilities()
		if err != nil {
			logrus.Warnf("getting capabilities failed: %v", err)
		}
		if len(caps) > 0 {
			fmt.Println("Capabilities:")
			for k, v := range caps {
				if len(v) > 0 {
					fmt.Printf("\t%s -> %s\n", k, strings.Join(v, " "))
				}
			}
		}

		// Chroot
		chroot, err := container.Chroot()
		if err != nil {
			logrus.Debugf("chroot check error: %v", err)
		}
		fmt.Printf("Chroot (not pivot_root): %t\n", chroot)

		// Seccomp
		seccompMode, err := container.SeccompEnforcingMode()
		if err != nil {
			logrus.Debugf("error: %v", err)
		}
		fmt.Printf("Seccomp: %s\n", seccompMode)

		return nil
	}

	// Run our program.
	p.Run()
}
