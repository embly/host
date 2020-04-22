package main

import (
	"net"

	"github.com/embly/host/pkg/agent"
	"github.com/sirupsen/logrus"
)

func main() {
	eth0 := net.IPv4(192, 168, 86, 30)
	loopback := net.IPv4(127, 0, 0, 1)
	docker0 := net.IPv4(172, 17, 0, 1)
	_, _, _ = eth0, loopback, docker0
	a, err := agent.DefaultNewAgent(docker0)
	if err != nil {
		logrus.Fatal("couldn't start proxy agent", err)
		return
	}
	logrus.Info("hi")
	a.Start()
}
