package proxy

import (
	consul "github.com/hashicorp/consul/api"
)

type Consul interface {
	Start(config *consul.Config) (err error)
	ListenForChanges() error
	Services() ([]Service, error)
	ServiceDeletedCallback(func(Service) error)
}

var _ Consul = &defaultConsul{}

type defaultConsul struct {
	client *consul.Client
}

func NewConsul() Consul {
	return &defaultConsul{}
}

func (c *defaultConsul) Start(config *consul.Config) (err error) {
	c.client, err = consul.NewClient(config)
	if err != nil {
		return
	}
	services, _, err := c.client.Catalog().Services(nil)
	if err != nil {
		return
	}
	_ = services
	return
}
func (c *defaultConsul) Services() (services []Service, err error) {
	return
}
func (c *defaultConsul) ServiceDeletedCallback(cb func(Service) error) {

}

func (c *defaultConsul) ListenForChanges() (err error) {
	// n := 1
	// if n <= 0 {
	// 	n = 1
	// }

	// sem := make(chan int, n)
	// cfgs := make(chan []string, len(m))
	// for name, passing := range m {
	// 	name, passing := name, passing
	// 	go func() {
	// 		sem <- 1
	// 		cfgs <- w.serviceConfig(name, passing)
	// 		<-sem
	// 	}()
	// }

	// var config []string
	// for i := 0; i < len(m); i++ {
	// 	cfg := <-cfgs
	// 	config = append(config, cfg...)
	// }
	// return
}
