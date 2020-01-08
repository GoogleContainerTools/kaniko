package main

import (
	_ "crypto/sha256"

	"github.com/containerd/continuity/commands"
)

func main() {
	commands.MainCmd.Execute()
}
