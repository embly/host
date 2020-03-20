package proxy

import (
	"net"
	"strconv"
)

type Service interface {
	IsAlive() bool
	Name() string
}

type defaultService struct {
	name string
	// TODO: alive?
}

type LoadBalancer interface {
	NextTask() (Task, error)
}

var _ LoadBalancer = &randomLoadBalancer{}

type randomLoadBalancer struct {
	tasks []Task
	alive bool
}

func (lb *randomLoadBalancer) NextTask() (task Task, err error) {
	return
}

type Task interface {
	Address() string
}

var _ Task = &defaultTask{}

type defaultTask struct {
	ip   net.IP
	port int
}

func (t *defaultTask) Address() string {
	return net.JoinHostPort(t.ip.String(), strconv.Itoa(t.port))
}
