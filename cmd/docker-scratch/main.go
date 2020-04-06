package main

import (
	docker "github.com/fsouza/go-dockerclient"
	"github.com/maxmcd/tester"
)

func main() {
	client, err := docker.NewClientFromEnv()
	tester.Print(client, err)

	tester.Print(client.FilteredListNetworks(docker.NetworkFilterOpts{
		"name": map[string]bool{"nomad-dashboard-dashboard": true},
	}))
}
