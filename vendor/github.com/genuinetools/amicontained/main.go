package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/genuinetools/amicontained/container"
	"github.com/genuinetools/amicontained/version"
	"github.com/sirupsen/logrus"
)

const (
	// BANNER is what is printed for help/info output
	BANNER = `                 _                 _        _                _
  __ _ _ __ ___ (_) ___ ___  _ __ | |_ __ _(_)_ __   ___  __| |
 / _` + "`" + ` | '_ ` + "`" + ` _ \| |/ __/ _ \| '_ \| __/ _` + "`" + ` | | '_ \ / _ \/ _` + "`" + ` |
| (_| | | | | | | | (_| (_) | | | | || (_| | | | | |  __/ (_| |
 \__,_|_| |_| |_|_|\___\___/|_| |_|\__\__,_|_|_| |_|\___|\__,_|
 Container introspection tool.
 Version: %s

`
)

var (
	debug bool
	vrsn  bool
)

func init() {
	// parse flags
	flag.BoolVar(&vrsn, "version", false, "print version and exit")
	flag.BoolVar(&vrsn, "v", false, "print version and exit (shorthand)")
	flag.BoolVar(&debug, "d", false, "run in debug mode")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(BANNER, version.VERSION))
		flag.PrintDefaults()
	}

	flag.Parse()

	// set log level
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if vrsn {
		fmt.Printf("amicontained version %s, build %s\n", version.VERSION, version.GITCOMMIT)
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		return
	}

	// parse the arg
	arg := flag.Args()[0]

	if arg == "help" {
		usageAndExit("", 0)
	}

	if arg == "version" {
		fmt.Printf("amicontained version %s, build %s\n", version.VERSION, version.GITCOMMIT)
		os.Exit(0)
	}
}

func main() {
	// Container Runtime
	runtime, err := container.DetectRuntime()
	if err != nil && err != container.ErrContainerRuntimeNotFound {
		log.Fatal(err)
		return
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
}

func usageAndExit(message string, exitCode int) {
	if message != "" {
		fmt.Fprintf(os.Stderr, message)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(exitCode)
}
