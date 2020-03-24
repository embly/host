package agent

import (
	"fmt"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/maxmcd/tester"
)

func TestDockerLabelSearch(te *testing.T) {
	t := tester.New(te)

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
		fmt.Println(x.Action, x.Type, x.Actor.Attributes)
		t.Print(x)
	}
}
