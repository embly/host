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
	"google.golang.org/grpc"
)

type Agent struct {
	ip     net.IP
	cd     *ConsulData
	nd     *NomadData
	docker Docker

	grpcServer *grpc.Server

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

func DefaultNewAgent(ip net.IP) (*Agent, error) {
	return NewAgent(ip, NewProxySocket, NewConsulData, NewNomadData, NewIPTables(exec.New()), NewDocker)
}

func NewAgent(ip net.IP,
	newProxySocket func(string, net.IP, int) (ProxySocket, error),
	newConsulData func() (*ConsulData, error),
	newNomadData func() (*NomadData, error),
	newIPTables func() (IPTables, error),
	newDocker func() (Docker, error)) (a *Agent, err error) {
	for _, sleepFor := range EndpointDialTimeouts {
		a, err = newAgent(ip, newProxySocket, newConsulData, newNomadData, newIPTables, newDocker)
		if _, ok := err.(transientError); ok {
			logrus.Info("error starting proxy, sleeping and retrying: ", err)
			time.Sleep(sleepFor * 5)
			continue
		}
		return a, err
	}
	return
}

type transientError error

func newAgent(ip net.IP,
	newProxySocket func(string, net.IP, int) (ProxySocket, error),
	newConsulData func() (*ConsulData, error),
	newNomadData func() (*NomadData, error),
	newIPTables func() (IPTables, error),
	newDocker func() (Docker, error)) (
	a *Agent, err error) {
	a = &Agent{}
	a.ip = ip
	a.initFields()

	a.newProxySocket = newProxySocket

	if a.cd, err = newConsulData(); err != nil {
		err = transientError(err)
		return
	}

	if a.nd, err = newNomadData(); err != nil {
		err = transientError(err)
		return
	}
	if err = a.nd.SetNodeID(); err != nil {
		err = transientError(err)
		return
	}
	if a.docker, err = newDocker(); err != nil {
		err = transientError(err)
		return
	}
	if err = a.docker.AddEventListener(a.dockerListener); err != nil {
		err = transientError(err)
		return
	}
	if a.ipt, err = newIPTables(); err != nil {
		return
	}
	_ = a.ipt.CleanUpPreroutingRules()

	if err = a.StartDeployServer(); err != nil {
		return
	}
	go a.cd.Updates(a.consulUpdates)
	go a.nd.Updates(a.nomadUpdates)
	return
}

func (a *Agent) initFields() {
	a.dockerListener = make(chan *docker.APIEvents, 2)
	a.consulUpdates = make(chan map[string]Service, 2)
	a.nomadUpdates = make(chan []*nomad.Allocation, 2)

	a.services = map[string]*Service{}
	a.connectRequests = map[TaskID]ConnectRequest{}
	a.proxies = map[string]ProxySocket{}
	a.containers = map[TaskID]Container{}
	a.allocations = map[string]Allocation{}
	a.rules = map[ProxyKey]ProxyRule{}
}

func (a *Agent) Start() {
	logrus.Info("agent started")
	go func() {
		log.Fatal(StartDNS())
	}()
	for {
		a.Tick()
	}
}

const (
	TickDocker = iota
	TickNomad
	TickConsul
)

func (a *Agent) Tick() int {
	select {
	case event := <-a.dockerListener:
		_ = a.processDockerUpdate(event)
		return TickDocker
	case allocations := <-a.nomadUpdates:
		a.processNomadUpdate(allocations)
		return TickNomad
	case services := <-a.consulUpdates:
		a.processConsulUpdate(services)
		return TickConsul
	}
}

var NomadAllocKey = "com.hashicora.nomad.alloc_id"

func (a *Agent) setUpProxyRule(ct ConnectRequest) (err error) {
	cont, haveContainer := a.containers[ct.taskID]
	for _, address := range ct.desiredServices {
		_, haveProxyRule := a.rules[ct.taskID.toProxyKey(address)]
		proxySocket, haveProxySocket := a.proxies[address]
		if haveContainer && !haveProxyRule && haveProxySocket {
			pr := ProxyRule{
				proxyIP:       a.ip.String(),
				containerIP:   cont.IPAddress,
				proxyPort:     proxySocket.ListenPort(),
				containerPort: a.services[address].port,
			}
			if err = a.ipt.AddProxyRuleToPrerouting(pr); err != nil {
				return
			}
			a.rules[ct.taskID.toProxyKey(address)] = pr
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

// when a new container starts we check our existing task to see if we
// have a proxy and set one up if we don't
func (a *Agent) processDockerUpdate(event *docker.APIEvents) (err error) {
	taskID := taskIDFromString(event.Actor.Attributes["name"])
	if event.Type != "container" || taskID.allocID == "" {
		return
	}
	switch event.Action {
	case "start":
		if err = a.processDockerStart(taskID, event); err != nil {
			return
		}
	case "kill":
		fallthrough
	case "stop":
		// if a container is stopped then we delete the proxy rule

		delete(a.containers, taskID)
		for _, ct := range a.connectRequests {
			if ct.taskID == taskID {
				for _, address := range ct.desiredServices {
					pr, havePR := a.rules[taskID.toProxyKey(address)]
					if !havePR {
						continue
					}
					if err := a.ipt.DeleteProxyRule(pr); err != nil {
						logrus.Error(errors.Wrap(err, "error deleting proxy rule"))
					}
					delete(a.rules, taskID.toProxyKey(address))
				}
			}
		}
	}
	return
}

func (a *Agent) processDockerStart(taskID TaskID, event *docker.APIEvents) (err error) {
	cont, err := a.docker.InspectContainerWithOptions(docker.InspectContainerOptions{ID: event.Actor.ID})
	if err != nil {
		return err
	}
	a.containers[taskID] = Container{
		IPAddress:   cont.NetworkSettings.IPAddress,
		ContainerID: cont.ID,
		TaskID:      taskID,
	}
	for _, ct := range a.connectRequests {
		// TODO: if alloc ids can be the same on different nodes this could create a strange highly unlikely incorrect rule creation
		// filter by node probably for connectTo requests
		if ct.taskID == taskID {
			if err = a.setUpProxyRule(ct); err != nil {
				logrus.Error(errors.Wrap(err, "couldn't set up proxy rule in docker listener"))
			}
		}
	}
	return
}

func (a *Agent) allocToAllocation(alloc *nomad.Allocation) Allocation {
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

func (a *Agent) processNomadUpdate(allocations []*nomad.Allocation) {
	keys := map[string]struct{}{}
	connectKeys := map[TaskID]struct{}{}

	for _, alloc := range allocations {
		keys[alloc.ID] = struct{}{}

		a.allocations[alloc.ID] = a.allocToAllocation(alloc)

		for _, taskGroup := range alloc.Job.TaskGroups {
			for _, task := range taskGroup.Tasks {
				if task.Meta["connect_to"] == "" {
					continue
				}
				key := TaskID{allocID: alloc.ID, name: task.Name}
				connectRequest := ConnectRequest{
					taskID:          key,
					desiredServices: strings.Split(task.Meta["connect_to"], ","),
				}
				if _, ok := a.connectRequests[key]; !ok {
					a.connectRequests[key] = connectRequest
				}
				if err := a.setUpProxyRule(connectRequest); err != nil {
					logrus.Error(errors.Wrap(err, "couldn't set up proxy rule in nomad listener"))
				}
				connectKeys[key] = struct{}{}
			}
		}
	}

	for key := range a.connectRequests {
		if _, ok := connectKeys[key]; !ok {
			delete(a.connectRequests, key)
		}
	}

	// delete allocs that no longer exists
	for allocID := range a.allocations {
		if _, ok := keys[allocID]; !ok {
			delete(a.allocations, allocID)
		}
	}
}

func (a *Agent) processConsulUpdate(services map[string]Service) {
	// newTasks := map[string]map[string]Task{}

	for key, service := range services {
		// check for services we don't have and add them
		_, ok := a.services[key]
		if !ok {
			logrus.Info("adding proxy for ", key)
			svc := service
			// add an entirely new service
			proxySocket, err := a.newProxySocket(svc.protocol, a.ip, 0)
			if err != nil {
				// this will error if the input data (protocol, addr, etc..) is invalid
				// likely means we just ignore the service
				logrus.Error(err)
				continue
			}
			go proxySocket.ProxyLoop(&svc)
			a.services[key] = &svc
			a.proxies[key] = proxySocket

			// TODO: less dumb than this?
			for _, ct := range a.connectRequests {
				for _, addr := range ct.desiredServices {
					if addr == key {
						if err = a.setUpProxyRule(ct); err != nil {
							logrus.Error(errors.Wrap(err, "couldn't set up proxy rule in consul listener"))
						}
					}
				}
			}
		} else {
			// update the task list on an existing service
			added, deleted := a.services[key].processUpdate(service.inventory)
			_, _ = added, deleted
		}
	}

	// check for services we no longer have and shut them down
	for key := range a.services {
		if _, ok := services[key]; !ok {
			// marking it as dead will stop new requests
			// drop the proxy from the map and it will eventually exit
			a.services[key].alive = false
			a.proxies[key].Close()
			delete(a.services, key)
			delete(a.proxies, key)
		}
	}
}
