package agent

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	consul "github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
)

type ConsulInventory struct {
	services        map[string]Service
	connectRequests map[string]ConnectTo
}

// Service defines a collection of tasks that can be routed to. Service will load balance traffic between these tasks
type Service struct {
	inventory map[string]Task
	lock      *sync.RWMutex
	hostname  string
	port      int
	alive     bool
	protocol  string
}

type ConnectTo struct {
	desiredServices []string
	node            string
	allocID         string
	serviceID       string
}

func (ct *ConnectTo) Name() string {
	return fmt.Sprintf("%s.%s", ct.node, ct.allocID)
}

func (s *Service) Name() string {
	return fmt.Sprintf("%s:%d", s.hostname, s.port)
}

func (s *Service) processUpdate(inventory map[string]Task) (deleted map[string]Task, added map[string]Task) {
	// check for new tasks we don't have and add them
	deleted = map[string]Task{}
	added = map[string]Task{}

	s.lock.Lock()
	for key, task := range inventory {
		if _, ok := s.inventory[key]; !ok {
			s.inventory[key] = task
			added[key] = task
		}
	}

	// check for tasks that no longer exist and remove them
	for key := range s.inventory {
		if task, ok := inventory[key]; !ok {
			deleted[key] = task
			delete(s.inventory, key)
		}
	}
	s.lock.Unlock()
	return
}

func (s *Service) NextTask() (task Task, err error) {
	// random task
	for _, task := range s.inventory {
		return task, nil
	}
	return
}

type Task struct {
	address   string
	port      int
	allocID   string
	node      string
	serviceID string
}

func (t *Task) Name() string {
	return fmt.Sprintf("%s.%s", t.node, t.allocID)
}

func (t *Task) Address() string {
	return net.JoinHostPort(t.address, strconv.Itoa(t.port))
}

type ConsulData interface {
	Updates(chan ConsulInventory)
}

var _ ConsulData = &defaultConsulData{}

type defaultConsulData struct {
	cc              ConsulClient
	serviceParallel int
}

func NewConsulData() (cd ConsulData, err error) {
	cc, err := NewConsulClient(consul.DefaultConfig())
	if err != nil {
		return
	}
	cd = &defaultConsulData{
		cc:              cc,
		serviceParallel: 3,
	}
	return
}

func (c *defaultConsulData) Updates(ch chan ConsulInventory) {
	var lastIndex uint64
	var q *consul.QueryOptions
	for {
		logrus.WithFields(logrus.Fields{"index": lastIndex}).Info("consul wait")
		q = &consul.QueryOptions{RequireConsistent: true, WaitIndex: lastIndex}
		serviceTags, meta, err := c.cc.Services(q)
		if err != nil {
			// TODO: what if we can never reconnect?
			logrus.Error(err)
			time.Sleep(time.Second)
			continue
		}
		serviceTags = filterServices(serviceTags)
		lastIndex = meta.LastIndex
		ch <- c.getInventory(serviceTags)
	}
}

func tagsToData(tags []string) (name, protocol string, port int) {
	for _, tag := range tags {
		if strings.Contains(tag, "dns_name") {
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
		}
		if strings.Contains(tag, "protocol") {
			parts := strings.Split(tag, "=")
			if len(parts) <= 1 {
				continue
			}
			protocol = parts[1]
		}
	}
	return
}

var allocIDRegex = regexp.MustCompile(`_nomad-task-([a-f0-9-]{36})`)

func parseAllocIDFromServiceID(in string) (out string) {
	matches := allocIDRegex.FindAllStringSubmatch(in, 1)
	if len(matches) == 0 || len(matches[0]) <= 1 {
		logrus.Error("could not parse alloc id from: ", in)
		return
	}
	return matches[0][1]
}

func (c *defaultConsulData) getService(name string, tags []string) (service Service) {
	service.alive = true
	service.lock = &sync.RWMutex{}

	dnsName, protocol, dnsPort := tagsToData(tags)
	if dnsName == "" || dnsPort == 0 {
		logrus.Error("error parsing tags on service")
		return
	}
	service.port = dnsPort
	service.protocol = protocol
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
			allocID:   parseAllocIDFromServiceID(s.ServiceID),
		}
		service.inventory[task.Name()] = task
	}
	return
}

func parseConnectToTag(tags []string) []string {
	for _, tag := range tags {
		if strings.Contains(tag, "connect_to") {
			parts := strings.Split(tag, "=")
			if len(parts) < 2 {
				continue
			}
			return strings.Split(parts[1], ",")
		}
	}
	return nil
}

func (c *defaultConsulData) getConnectTo(name string, tags []string) (ct []ConnectTo) {
	services, err := c.cc.Service(name, "")
	if err != nil {
		logrus.Error(err)
		return
	}
	for _, s := range services {
		hostnames := parseConnectToTag(tags)
		if len(hostnames) == 0 {
			continue
		}
		ct = append(ct, ConnectTo{
			serviceID:       s.ServiceID,
			node:            s.Node,
			allocID:         parseAllocIDFromServiceID(s.ServiceID),
			desiredServices: hostnames,
		})
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
			if strings.Contains(tag, "dns_name") || strings.Contains(tag, "connect_to") {
				out[name] = tags
				break
			}
		}
	}
	return out
}

func isService(tags []string) bool {
	for _, tag := range tags {
		if strings.Contains(tag, "dns_name") {
			return true
		}
	}
	return false
}

func (c *defaultConsulData) getInventory(serviceTags map[string][]string) (inventory ConsulInventory) {
	inventory = ConsulInventory{
		services:        map[string]Service{},
		connectRequests: map[string]ConnectTo{},
	}

	n := c.serviceParallel
	if n <= 0 {
		n = 3
	}

	sem := make(chan int, n)

	cfgs := make(chan Service, len(serviceTags))
	connects := make(chan []ConnectTo, len(serviceTags))
	for name, tags := range serviceTags {
		tags, name := tags, name
		go func() {
			sem <- 1
			if isService(tags) {
				cfgs <- c.getService(name, tags)
			} else {
				connects <- c.getConnectTo(name, tags)
			}
			<-sem
		}()
	}

	for i := 0; i < len(serviceTags); i++ {
		select {
		case svc := <-cfgs:
			if svc.hostname != "" {
				inventory.services[svc.Name()] = svc
			}
		case cts := <-connects:
			for _, ct := range cts {
				inventory.connectRequests[ct.Name()] = ct
			}
		}
	}
	return
}
