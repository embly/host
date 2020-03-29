package agent

import (
	"fmt"
	"testing"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/maxmcd/tester"
)

type fakeDocker struct {
	listener   chan *docker.APIEvents
	containers map[string]docker.Container
}

func (fd *fakeDocker) AddEventListener(listener chan<- *docker.APIEvents) error {
	go func() {
		for {
			thing := <-fd.listener
			listener <- thing
		}
	}()
	return nil
}
func (fd *fakeDocker) InspectContainerWithOptions(opts docker.InspectContainerOptions) (*docker.Container, error) {
	if cont, ok := fd.containers[opts.ID]; ok {
		return &cont, nil
	}
	return nil, nil
}

func newFakeDocker() (Docker, error) {
	return &fakeDocker{containers: map[string]docker.Container{}, listener: make(chan *docker.APIEvents)}, nil
}
func TestDockerLabelSearch(te *testing.T) {
	t := tester.New(te)
	t.Skip()
	allocID := "4243abe5-1b52-1791-4af9-8383a649265b"
	c, err := docker.NewClientFromEnv()
	t.PanicOnErr(err)
	containers, err := c.ListContainers(docker.ListContainersOptions{
		Filters: map[string][]string{
			"label": []string{fmt.Sprintf("com.hashicorp.nomad.alloc_id=%s", allocID)},
		},
	})
	t.PanicOnErr(err)
	t.Print(containers)

	listener := make(chan *docker.APIEvents)
	err = c.AddEventListener(listener)
	t.PanicOnErr(err)

	for x := range listener {
		if x.Action == "start" &&
			x.Type == "container" &&
			len(x.Actor.Attributes["com.hashicorp.nomad.alloc_id"]) > 0 {
			fmt.Println(x.Action, x.Type, x.Actor.Attributes, time.Now())
			cont, err := c.InspectContainerWithOptions(docker.InspectContainerOptions{ID: x.Actor.ID})
			if err != nil {
				panic(err)
			}
			// cont.NetworkSettings.IPAddress
			t.Print(cont.NetworkSettings.IPAddress)
		}
	}
}
