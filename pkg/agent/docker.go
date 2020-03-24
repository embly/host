package agent

import (
	"fmt"

	docker "github.com/fsouza/go-dockerclient"
)

/*
docker listener will listen for new containers with alloc_ids
if there is a proxy for the alloc id it will create the iptable rule for the proxy
if the container goes away it will remove the iptable rule

if there is a docker container with an alloc_id and a new consul rule
is added we must also add the iptables rule
*/

type ContainerListener interface {
	Initialize() error
	Start()
	Cleanup() error
	NewTask(task Task)
	DeleteTask(task Task)
}

var _ ContainerListener = &defaultContainerListener{}

type defaultContainerListener struct {
	tasks    map[string]Task
	rules    map[string]ProxyRule
	c        *docker.Client
	listener chan *docker.APIEvents
	ipt      IPTables
}

func (dl *defaultContainerListener) Initialize() (err error) {
	dl.tasks = map[string]Task{}
	dl.rules = map[string]ProxyRule{}

	if dl.c, err = docker.NewClientFromEnv(); err != nil {
		return
	}
	listener := make(chan *docker.APIEvents)
	if err = dl.c.AddEventListener(listener); err != nil {
		return
	}
	if err = dl.ipt.CreateChains(); err != nil {
		return
	}
	return nil
}
func (dl *defaultContainerListener) Cleanup() error {
	return dl.ipt.DeleteChains()
}

func (dl *defaultContainerListener) Start() {
	for x := range dl.listener {
		if x.Action == "start" &&
			x.Type == "container" &&
			len(x.Actor.Attributes["com.hashicorp.nomad.alloc_id"]) > 0 {
			fmt.Println(x.Action, x.Type, x.Actor.Attributes)
		}
	}
}
func (dl *defaultContainerListener) NewTask(task Task) {

}
func (dl *defaultContainerListener) DeleteTask(task Task) {

}
