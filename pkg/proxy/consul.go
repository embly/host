package proxy

import (
	"strconv"
	"strings"

	consul "github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
)

type Consul interface {
	Start(config *consul.Config) (err error)
	ListenForChanges() error
	Services() ([]Service, error)
	ServiceDeletedCallback(func(Service) error)
}

var _ Consul = &defaultConsul{}

type defaultConsul struct {
	client          *consul.Client
	serviceParallel int
}

func NewConsul() Consul {
	return &defaultConsul{
		serviceParallel: 3,
	}
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

type TmpService struct {
	address  string
	port     int
	protocol string
	dnsName  string
	dnsPort  int
}

func tagsToDnsData(tags []string) (name string, port int) {
	for _, tag := range tags {
		if strings.Contains(tag, "dns-name") {
			parts := strings.Split(tag, "=")
			if len(parts) <= 1 {
				continue
			}
			hostParts := strings.Split(parts[1], ":")
			if len(hostParts) <= 1 {
				continue
			}
			port, _ = strconv.Atoi(hostParts[1])
			name = hostParts[0]
			return
		}
	}
	logrus.Error("dns-name not found")
	return
}

func (c *defaultConsul) getServiceInstaces(name string) (instances []TmpService) {
	services, _, err := c.client.Catalog().Service(name, "", nil)
	if err != nil {
		logrus.Error(err)
		return
	}
	for _, service := range services {
		// TODO: why parse these here if they're the same on every service
		dnsName, dnsPort := tagsToDnsData(service.ServiceTags)
		if dnsName == "" || dnsPort == 0 {
			logrus.Error("error parsing tags on service", service.Node, service.ServiceID)
		}
		instances = append(instances, TmpService{
			address: service.Address,
			port:    service.ServicePort,
			dnsName: dnsName,
			dnsPort: dnsPort,
		})
		// service.Address
	}

	return
}

func filterServices(serviceNames map[string][]string) map[string][]string {
	out := map[string][]string{}
	for name, tags := range serviceNames {
		if len(tags) == 0 {
			continue
		}
		// TODO: either heavily document this, or find a better way to consistently identfy
		// routable services
		for _, tag := range tags {
			if strings.Contains(tag, "dns-name") {
				out[name] = tags
				break
			}
		}
	}
	return out
}
func (c *defaultConsul) getServices() (services []TmpService) {
	serviceNames, _, err := c.client.Catalog().Services(nil)
	if err != nil {
		logrus.Error(err)
		return
	}
	serviceNames = filterServices(serviceNames)

	n := c.serviceParallel
	if n <= 0 {
		n = 1
	}

	sem := make(chan int, n)
	cfgs := make(chan []TmpService, len(serviceNames))
	for name := range serviceNames {
		go func(name string) {
			sem <- 1
			cfgs <- c.getServiceInstaces(name)
			<-sem
		}(name)
	}

	for i := 0; i < len(serviceNames); i++ {
		cfg := <-cfgs
		services = append(services, cfg...)
	}
	return

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
	return
}
