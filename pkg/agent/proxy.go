package agent

import (
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/embly/host/pkg/exec"
	docker "github.com/fsouza/go-dockerclient"
	nomad "github.com/hashicorp/nomad/api"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Proxy struct {
	ip     net.IP
	cd     *ConsulData
	nd     *NomadData
	docker Docker

	dockerListener chan *docker.APIEvents
	consulUpdates  chan map[string]Service
	nomadUpdates   chan []*nomad.Allocation

	ipt            IPTables
	newProxySocket func(string, net.IP, int) (ProxySocket, error)

	// services stores services, keyed by the hostname the service is addressed by
	services map[string]*Service

	// connectRequests stores all global requests to connect to another service
	connectRequests map[TaskID]ConnectRequest

	// proxies stores active proxies. key is the service hostname+port
	proxies map[string]ProxySocket

	// containers tracks containers on this host. container contains the container ip addr and id
	containers map[TaskID]Container

	// allocations are nomad allocations, keyed by alloc_id
	allocations map[string]Allocation

	// proxy rules for a specific container
	rules map[ProxyKey]ProxyRule

	// temporary rules for just sibling containers
	siblingRules map[LinkKey]ProxyRule
}

type ProxyKey struct {
	taskID TaskID

	// like counter.dashboard:3000
	address string
}

type TaskID struct {
	allocID string
	name    string
}

func (tid TaskID) toProxyKey(address string) ProxyKey { return ProxyKey{taskID: tid, address: address} }

type LinkKey struct {
	allocID  string
	name     string
	linkName string
	port     ResourcePort
}

func DefaultNewProxy(ip net.IP) (*Proxy, error) {
	return NewProxy(ip, NewProxySocket, NewConsulData, NewNomadData, NewIPTables(exec.New()), NewDocker)
}

func NewProxy(ip net.IP,
	newProxySocket func(string, net.IP, int) (ProxySocket, error),
	newConsulData func() (*ConsulData, error),
	newNomadData func() (*NomadData, error),
	newIPTables func() (IPTables, error),
	newDocker func() (Docker, error)) (p *Proxy, err error) {
	for _, sleepFor := range EndpointDialTimeouts {
		p, err = newProxy(ip, newProxySocket, newConsulData, newNomadData, newIPTables, newDocker)
		if _, ok := err.(transientError); ok {
			logrus.Info("error starting proxy, sleeping and retrying: ", err)
			time.Sleep(sleepFor * 5)
			continue
		}
		return p, err
	}
	return
}

type transientError error

func newProxy(ip net.IP,
	newProxySocket func(string, net.IP, int) (ProxySocket, error),
	newConsulData func() (*ConsulData, error),
	newNomadData func() (*NomadData, error),
	newIPTables func() (IPTables, error),
	newDocker func() (Docker, error)) (
	p *Proxy, err error) {
	p = &Proxy{}
	p.ip = ip
	p.initFields()

	p.newProxySocket = newProxySocket

	if p.cd, err = newConsulData(); err != nil {
		err = transientError(err)
		return
	}

	if p.nd, err = newNomadData(); err != nil {
		err = transientError(err)
		return
	}
	if err = p.nd.SetNodeID(); err != nil {
		err = transientError(err)
		return
	}
	if p.docker, err = newDocker(); err != nil {
		err = transientError(err)
		return
	}
	if err = p.docker.AddEventListener(p.dockerListener); err != nil {
		err = transientError(err)
		return
	}
	if p.ipt, err = newIPTables(); err != nil {
		return
	}
	_ = p.ipt.CleanUpPreroutingRules()

	go p.cd.Updates(p.consulUpdates)
	go p.nd.Updates(p.nomadUpdates)
	return
}

func (p *Proxy) initFields() {
	p.dockerListener = make(chan *docker.APIEvents, 2)
	p.consulUpdates = make(chan map[string]Service, 2)
	p.nomadUpdates = make(chan []*nomad.Allocation, 2)

	p.services = map[string]*Service{}
	p.connectRequests = map[TaskID]ConnectRequest{}
	p.proxies = map[string]ProxySocket{}
	p.containers = map[TaskID]Container{}
	p.allocations = map[string]Allocation{}
	p.rules = map[ProxyKey]ProxyRule{}
}

func (p *Proxy) Start() {
	go func() {
		log.Fatal(StartDNS())
	}()
	for {
		p.Tick()
	}
}

const (
	TickDocker = iota
	TickNomad
	TickConsul
)

func (p *Proxy) Tick() int {
	select {
	case event := <-p.dockerListener:
		_ = p.processDockerUpdate(event)
		p.checkForTaskResourceLinks()
		return TickDocker
	case allocations := <-p.nomadUpdates:
		p.processNomadUpdate(allocations)
		p.checkForTaskResourceLinks()
		return TickNomad
	case services := <-p.consulUpdates:
		p.processConsulUpdate(services)
		return TickConsul
	}
}

var NomadAllocKey = "com.hashicorp.nomad.alloc_id"

func (p *Proxy) setUpProxyRule(ct ConnectRequest) (err error) {
	cont, haveContainer := p.containers[ct.taskID]
	for _, address := range ct.desiredServices {
		_, haveProxyRule := p.rules[ct.taskID.toProxyKey(address)]
		proxySocket, haveProxySocket := p.proxies[address]
		if haveContainer && !haveProxyRule && haveProxySocket {
			pr := ProxyRule{
				proxyIP:       p.ip.String(),
				containerIP:   cont.IPAddress,
				proxyPort:     proxySocket.ListenPort(),
				containerPort: p.services[address].port,
			}
			if err = p.ipt.AddProxyRuleToPrerouting(pr); err != nil {
				return
			}
			p.rules[ct.taskID.toProxyKey(address)] = pr
		}
	}
	return
}

func taskIDFromString(id string) (taskID TaskID) {
	parts := strings.SplitN(id, "-", 2)
	if len(parts) < 2 {
		return
	}
	return TaskID{
		name:    parts[0],
		allocID: parts[1],
	}
}

func (p *Proxy) checkForTaskResourceLinks() {
	for _, alloc := range p.allocations {
		alloc.allTaskResourcePairs(func(a, b TaskResource) {
			key := LinkKey{allocID: alloc.ID, name: a.Name, linkName: b.Name}
			if _, ok := p.siblingRules[key]; ok {
				// we have the necessary proxy rules for this one
				return
			}
			aCont, aOk := p.containers[TaskID{
				allocID: alloc.ID,
				name:    a.Name,
			}]
			if !aOk {
				return
			}
			bCont, bOk := p.containers[TaskID{
				allocID: alloc.ID,
				name:    b.Name,
			}]
			if !bOk {
				return
			}
			for _, resourcePort := range b.Ports {
				key.port = resourcePort
				if _, ok := p.siblingRules[key]; ok {
					continue
				}
				pr := ProxyRule{
					containerIP:   aCont.IPAddress,
					containerPort: resourcePort.Listening,
					proxyIP:       bCont.IPAddress,
					proxyPort:     resourcePort.Value,
				}
				if err := p.ipt.AddProxyRuleToPrerouting(pr); err != nil {
					logrus.Error(err)
				}
				p.siblingRules[key] = pr
			}
		})
	}

	// todo: delete stale?
}

// when a new container starts we check our existing task to see if we
// have a proxy and set one up if we don't
func (p *Proxy) processDockerUpdate(event *docker.APIEvents) (err error) {
	taskID := taskIDFromString(event.Actor.Attributes["name"])
	if event.Type != "container" || taskID.allocID == "" {
		return
	}
	switch event.Action {
	case "start":
		if err = p.processDockerStart(taskID, event); err != nil {
			return
		}
	case "stop":
		fallthrough
	case "kill":
		// if a container is stopped then we delete the proxy rule

		delete(p.containers, taskID)
		for _, ct := range p.connectRequests {
			if ct.taskID == taskID {
				for _, address := range ct.desiredServices {
					pr, havePR := p.rules[taskID.toProxyKey(address)]
					if !havePR {
						continue
					}
					if err := p.ipt.DeleteProxyRule(pr); err != nil {
						logrus.Error(errors.Wrap(err, "error deleting proxy rule"))
					}
					delete(p.rules, taskID.toProxyKey(address))
				}
			}
		}
	}
	return
}

func (p *Proxy) processDockerStart(taskID TaskID, event *docker.APIEvents) (err error) {
	cont, err := p.docker.InspectContainerWithOptions(docker.InspectContainerOptions{ID: event.Actor.ID})
	if err != nil {
		return err
	}
	p.containers[taskID] = Container{
		IPAddress:   cont.NetworkSettings.IPAddress,
		ContainerID: cont.ID,
		TaskID:      taskID,
	}
	for _, ct := range p.connectRequests {
		// TODO: if alloc ids can be the same on different nodes this could create a strange highly unlikely incorrect rule creation
		// filter by node probably for connectTo requests
		if ct.taskID == taskID {
			if err = p.setUpProxyRule(ct); err != nil {
				logrus.Error(errors.Wrap(err, "couldn't set up proxy rule in docker listener"))
			}
		}
	}

	if alloc, ok := p.allocations[taskID.allocID]; ok {
		// we have an allocation locally for this container
		for _, tr := range alloc.TaskResources {
			if tr.Name == taskID.name {
				continue
			}
		}
	}
	return
}

func (p *Proxy) allocToAllocation(alloc *nomad.Allocation) Allocation {
	allocation := Allocation{
		ID:            alloc.ID,
		TaskResources: map[string]TaskResource{},
	}
	for name, taskResource := range alloc.TaskResources {
		_ = taskResource
		if len(taskResource.Networks) == 0 {
			continue
		}
		var ports []ResourcePort
		for _, port := range taskResource.Networks[0].DynamicPorts {
			listening, err := strconv.Atoi(port.Label)
			if err != nil {
				continue
			}
			ports = append(ports, ResourcePort{
				Listening: listening,
				Value:     port.Value,
			})
		}
		allocation.TaskResources[name] = TaskResource{
			Name:      name,
			IPAddress: taskResource.Networks[0].IP,
			Ports:     ports,
		}
	}
	return allocation
}

func (p *Proxy) processNomadUpdate(allocations []*nomad.Allocation) {
	keys := map[string]struct{}{}
	connectKeys := map[TaskID]struct{}{}

	for _, alloc := range allocations {
		keys[alloc.ID] = struct{}{}

		p.allocations[alloc.ID] = p.allocToAllocation(alloc)

		for _, taskGroup := range alloc.Job.TaskGroups {
			for _, task := range taskGroup.Tasks {
				key := TaskID{allocID: alloc.ID, name: task.Name}
				connectRequest := ConnectRequest{
					taskID:          key,
					desiredServices: strings.Split(task.Meta["connect_to"], ","),
				}
				if _, ok := p.connectRequests[key]; !ok {
					p.connectRequests[key] = connectRequest
				}
				if err := p.setUpProxyRule(connectRequest); err != nil {
					logrus.Error(errors.Wrap(err, "couldn't set up proxy rule in nomad listener"))
				}
				connectKeys[key] = struct{}{}
			}
		}
	}

	for key := range p.connectRequests {
		if _, ok := connectKeys[key]; !ok {
			delete(p.connectRequests, key)
		}
	}

	// delete allocs that no longer exists
	for allocID := range p.allocations {
		if _, ok := keys[allocID]; !ok {
			delete(p.allocations, allocID)
		}
	}
}

func (p *Proxy) processConsulUpdate(services map[string]Service) {
	// newTasks := map[string]map[string]Task{}

	for key, service := range services {
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

			// TODO: less dumb than this?
			for _, ct := range p.connectRequests {
				for _, addr := range ct.desiredServices {
					if addr == key {
						if err = p.setUpProxyRule(ct); err != nil {
							logrus.Error(errors.Wrap(err, "couldn't set up proxy rule in consul listener"))
						}
					}
				}
			}
		} else {
			// update the task list on an existing service
			added, deleted := p.services[key].processUpdate(service.inventory)
			_, _ = added, deleted
		}
	}

	// check for services we no longer have and shut them down
	for key := range p.services {
		if _, ok := services[key]; !ok {
			// marking it as dead will stop new requests
			// drop the proxy from the map and it will eventually exit
			p.services[key].alive = false
			p.proxies[key].Close()
			delete(p.services, key)
			delete(p.proxies, key)
		}
	}
}
