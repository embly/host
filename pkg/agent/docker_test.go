package agent

import (
	"fmt"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/maxmcd/tester"
)

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
}
