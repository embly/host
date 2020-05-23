package main

import (
	"github.com/embly/host/pkg/cli"
)

var (
	// version added at build time
	version string
)

func main() {
	cli.RunCommand(version)
}
