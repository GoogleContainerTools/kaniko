package testutil

import (
	"flag"
)

var rootEnabled bool

func init() {
	flag.BoolVar(&rootEnabled, "test.root", false, "enable tests that require root")
}
