package agent

import docker "github.com/fsouza/go-dockerclient"

type Docker interface {
	AddEventListener(chan<- *docker.APIEvents) error
	InspectContainerWithOptions(docker.InspectContainerOptions) (*docker.Container, error)
}

func NewDocker() (Docker, error) {
	return docker.NewClientFromEnv()
}

type container struct {
	ip string
	id string
}
