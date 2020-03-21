package proxy

import (
	"net"
	"strconv"
)

type defaultService struct {
	name string
	// TODO: alive?
}

type LoadBalancer interface {
	NextTask() (XTask, error)
}

var _ LoadBalancer = &randomLoadBalancer{}

type randomLoadBalancer struct {
	tasks []Task
	alive bool
}

func (lb *randomLoadBalancer) NextTask() (task XTask, err error) {
	return
}

type XTask interface {
	Address() string
}

var _ XTask = &defaultTask{}

type defaultTask struct {
	ip   net.IP
	port int
}

func (t *defaultTask) Address() string {
	return net.JoinHostPort(t.ip.String(), strconv.Itoa(t.port))
}
