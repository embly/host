package proxy

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	consul "github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
)

type Service struct {
	inventory map[string]Task
	hostname  string
	port      int
	alive     bool
	protocol  string
}

func (s *Service) Name() string {
	return fmt.Sprintf("%s:%d", s.hostname, s.port)
}

func (s *Service) processUpdate(inventory map[string]Task) {
	// check for new tasks we don't have and add them
	for key, task := range inventory {
		if _, ok := s.inventory[key]; !ok {
			s.inventory[key] = task
		}
	}

	// check for tasks that no longer exist and remove them
	for key := range s.inventory {
		if _, ok := inventory[key]; !ok {
			delete(s.inventory, key)
		}
	}
}

func (s *Service) NextTask() (task Task, err error) {
	return
}

type Task struct {
	address   string
	port      int
	node      string
	serviceID string
}

func (t *Task) Name() string {
	return fmt.Sprintf("%s.%s", t.node, t.serviceID)
}

func (t *Task) Address() string {
	return ""
}

type ConsulData interface {
	Updates(chan map[string]Service)
}

var _ ConsulData = &defaultConsulData{}

type defaultConsulData struct {
	cc              ConsulClient
	serviceParallel int
}

func NewConsulData() (cd ConsulData, err error) {
	cc, err := NewConsulClient(api.DefaultConfig())
	if err != nil {
		return
	}
	cd = &defaultConsulData{
		cc:              cc,
		serviceParallel: 3,
	}
	return
}

func (c *defaultConsulData) Updates(ch chan map[string]Service) {
	var lastIndex uint64
	var q *consul.QueryOptions
	for {
		q = &consul.QueryOptions{RequireConsistent: true, WaitIndex: lastIndex}
		serviceTags, meta, err := c.cc.Services(q)
		if err != nil {
			logrus.Error(err)
			return
		}
		serviceTags = filterServices(serviceTags)
		ch <- c.getInventory(serviceTags)
		lastIndex = meta.LastIndex
	}
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

func (c *defaultConsulData) getService(name string, tags []string) (service Service) {
	dnsName, dnsPort := tagsToDnsData(tags)
	if dnsName == "" || dnsPort == 0 {
		logrus.Error("error parsing tags on service")
		return
	}
	service.port = dnsPort
	service.hostname = dnsName
	service.inventory = map[string]Task{}
	services, err := c.cc.Service(name, "")
	if err != nil {
		logrus.Error(err)
		return
	}
	for _, s := range services {
		task := Task{
			address:   s.Address,
			port:      s.ServicePort,
			node:      s.Node,
			serviceID: s.ServiceID,
		}
		service.inventory[task.Name()] = task
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
func (c *defaultConsulData) getInventory(serviceTags map[string][]string) (inventory map[string]Service) {

	inventory = map[string]Service{}

	n := c.serviceParallel
	if n <= 0 {
		n = 3
	}

	sem := make(chan int, n)

	cfgs := make(chan Service, len(serviceTags))
	for name, tags := range serviceTags {
		name, tags := name, tags
		go func() {
			sem <- 1
			cfgs <- c.getService(name, tags)
			<-sem
		}()
	}

	for i := 0; i < len(serviceTags); i++ {
		svc := <-cfgs
		if svc.hostname != "" {
			inventory[svc.Name()] = svc
		}
	}
	return

}
