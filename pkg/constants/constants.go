package constants

import "github.com/sirupsen/logrus"

const (
	// DefaultLogLevel is the default log level
	DefaultLogLevel = logrus.InfoLevel

	// RootDir is the path to the root directory
	RootDir = "/"

	// WorkDir is the path to the work-dir direcotry
	WorkDir = "/work-dir/"
)

// Whitelist is a list of the directories and files that should be ignored when extracting
// the filesystem and snapshotting
var Whitelist = []string{"/work-dir", "/dockerfile", "/dev", "/sys", "/proc", "/var/run/secrets",
	"/etc/hostname", "/etc/hosts", "/etc/mtab", "/etc/resolv.conf", "/.dockerenv"}
