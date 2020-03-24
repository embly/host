package main

import (
	"github.com/embly/host/pkg/agent"
)

func main() {
	agent.NewProxy().Start()
}
