package agent

import docker "github.com/fsouza/go-dockerclient"

type Docker interface {
	AddEventListener(chan<- *docker.APIEvents) error
	InspectContainerWithOptions(docker.InspectContainerOptions) (*docker.Container, error)
}

func NewDocker() (Docker, error) {
	return docker.NewClientFromEnv()
}

// Container tracks a runnning docker container
type Container struct {
	IPAddress   string
	ContainerID string
	TaskID      TaskID
}
