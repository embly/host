package main

import (
	"net"

	"github.com/embly/host/pkg/agent"
	"github.com/sirupsen/logrus"
)

func main() {
	p, err := agent.DefaultNewProxy(net.IPv4(127, 0, 0, 1))
	if err != nil {
		logrus.Fatal("couldn't start proxy agent", err)
		return
	}
	p.Start()
}
