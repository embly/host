package agent

import (
	"net"
	"time"

	"github.com/embly/host/pkg/exec"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Proxy struct {
	ip     net.IP
	cd     ConsulData
	docker Docker

	listener      chan *docker.APIEvents
	consulUpdates chan ConsulInventory

	ipt            IPTables
	newProxySocket func(string, net.IP, int) (ProxySocket, error)

	// services stores services, keyed by the hostname the service is addressed by
	services map[string]*Service

	// connectRequests stores all global requests to connect to another service
	connectRequests map[string]ConnectTo

	// allocInventoryReference stores a map of allocID to "{service_hostname:port}.{task_id}"
	allocInventoryReference map[string]addressAndTaskID

	// proxies stores active proxies. key is the service hostname+port
	proxies map[string]ProxySocket

	// containerAllocs tracks containers on this host. container contains the container ip addr and id
	// keyed by alloc_id
	containerAllocs map[string]container

	// proxy rules for a specific container, keyed by alloc_id
	rules map[string]ProxyRule
}

func DefaultNewProxy(ip net.IP) (*Proxy, error) {
	return NewProxy(ip, NewProxySocket, NewConsulData, NewIPTables(exec.New()), NewDocker)
}

func NewProxy(ip net.IP,
	newProxySocket func(string, net.IP, int) (ProxySocket, error),
	newConsulData func() (ConsulData, error),
	newIPTables func() (IPTables, error),
	newDocker func() (Docker, error)) (p *Proxy, err error) {
	for _, sleepFor := range EndpointDialTimeouts {
		p, err = newProxy(ip, newProxySocket, newConsulData, newIPTables, newDocker)
		if _, ok := err.(transientError); ok {
			logrus.Info("error starting proxy, sleeping and retrying: ", err)
			time.Sleep(sleepFor)
			continue
		}
		return p, err
	}
	return
}

type transientError error

func newProxy(ip net.IP,
	newProxySocket func(string, net.IP, int) (ProxySocket, error),
	newConsulData func() (ConsulData, error),
	newIPTables func() (IPTables, error),
	newDocker func() (Docker, error)) (
	p *Proxy, err error) {
	p = &Proxy{}
	p.ip = ip
	p.newProxySocket = newProxySocket

	p.services = map[string]*Service{}
	p.proxies = map[string]ProxySocket{}
	p.rules = map[string]ProxyRule{}
	p.containerAllocs = map[string]container{}
	p.allocInventoryReference = map[string]addressAndTaskID{}

	p.consulUpdates = make(chan ConsulInventory, 2)
	p.listener = make(chan *docker.APIEvents, 2)

	if p.cd, err = newConsulData(); err != nil {
		err = transientError(err)
		return
	}
	if p.docker, err = newDocker(); err != nil {
		err = transientError(err)
		return
	}
	if err = p.docker.AddEventListener(p.listener); err != nil {
		err = transientError(err)
		return
	}
	if p.ipt, err = newIPTables(); err != nil {
		return
	}
	if err = p.ipt.DeleteChains(); err != nil {
		return
	}
	if err = p.ipt.CreateChains(); err != nil {
		return
	}

	go p.cd.Updates(p.consulUpdates)
	return
}

func (p *Proxy) Start() {
	for {
		_ = p.Tick()
	}
}

func (p *Proxy) Tick() (err error) {
	select {
	case event := <-p.listener:
		return p.processDockerUpdate(event)
	case inventory := <-p.consulUpdates:
		p.processConsulUpdate(inventory)
		return nil
	}
}

var NomadAllocKey = "com.hashicorp.nomad.alloc_id"

func (p *Proxy) setUpProxyRule(allocID, containerIP string, containerPort int, ps ProxySocket) (err error) {
	pr := ProxyRule{
		proxyIP:       p.ip.String(),
		containerIP:   containerIP,
		proxyPort:     ps.ListenPort(),
		containerPort: containerPort,
	}
	if err = p.ipt.AddProxyRule(pr); err != nil {
		return
	}
	p.rules[allocID] = pr
	return
}
func (p *Proxy) processDockerUpdate(event *docker.APIEvents) (err error) {
	// when a new container start somes in we check our existing task to see if we
	// have a proxy and set one up if we don't
	allocID := event.Actor.Attributes[NomadAllocKey]
	if event.Type != "container" && allocID != "" {
		return
	}
	if event.Action == "start" {
		cont, err := p.docker.InspectContainerWithOptions(docker.InspectContainerOptions{ID: event.Actor.ID})
		if err != nil {
			return err
		}
		// if we have the alloc already (from a consul event)
		// then we have everything we need, create the alloc
		if aandtID, ok := p.allocInventoryReference[allocID]; ok {
			proxySocket := p.proxies[aandtID.address]
			task := p.services[aandtID.address].inventory[aandtID.taskID]
			if err = p.setUpProxyRule(allocID, cont.NetworkSettings.IPAddress, task.port, proxySocket); err != nil {
				logrus.Error(errors.Wrap(err, "couldn't set up proxy rule in docker listener"))
			}
		}

		// this must be added after the proxy rule to prevent a moment when it's unclear
		// who should create the rule
		p.containerAllocs[allocID] = container{
			ip: cont.NetworkSettings.IPAddress,
			id: cont.ID,
		}
	}

	// if a container is stopped then we delete the proxy rule
	if event.Action == "stop" {
		delete(p.containerAllocs, allocID)
		pr := p.rules[allocID]
		if err := p.ipt.DeleteProxyRule(pr); err != nil {
			logrus.Error(errors.Wrap(err, "error deleting proxy rule"))
		}
		delete(p.rules, allocID)
	}
	return
}

type addressAndTaskID struct {
	address string
	taskID  string
}

func (p *Proxy) processConsulUpdate(inventory ConsulInventory) {
	// newTasks := map[string]map[string]Task{}

	for key, service := range inventory.services {
		// check for services we don't have and add them
		_, ok := p.services[key]
		if !ok {
			logrus.Info("adding proxy for ", key)
			svc := service
			// add an entirely new service
			proxySocket, err := p.newProxySocket(svc.protocol, p.ip, 0)
			if err != nil {
				// this will error if the input data (protocol, addr, etc..) is invalid
				// likely means we just ignore the service
				logrus.Error(err)
				continue
			}
			go proxySocket.ProxyLoop(&svc)
			p.services[key] = &svc
			p.proxies[key] = proxySocket
		} else {
			// update the task list on an existing service
			added, deleted := p.services[key].processUpdate(service.inventory)
			_, _ = added, deleted
		}
	}

	// check for services we no longer have and shut them down
	for key := range p.services {
		if _, ok := inventory.services[key]; !ok {
			// marking it as dead will stop new requests
			// drop the proxy from the map and it will eventually exit
			p.services[key].alive = false
			p.proxies[key].Close()
			delete(p.services, key)
			delete(p.proxies, key)
		}
	}

	for key, connectTo := range p.connectRequests {
		if _, ok := p.connectRequests[key]; !ok {
			p.connectRequests[key] = connectTo
			cont, haveAlloc := p.containerAllocs[connectTo.allocID]
			_, haveProxyRule := p.rules[connectTo.allocID]
			if haveAlloc && !haveProxyRule {
				_ = cont
				// TODO: set it up
				// proxySocket := p.proxies[key]
				// if err := p.setUpProxyRule(connectTo.allocID, cont.ip, task.port, proxySocket); err != nil {
				// 	logrus.Error(errors.Wrap(err, "couldn't set up proxy rule in consul listener"))
				// }
			}

		}
	}

	for key := range p.connectRequests {
		if _, ok := inventory.connectRequests[key]; !ok {
			delete(p.connectRequests, key)
		}
	}

	return
}
