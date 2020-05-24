package cli

import (
	"fmt"
	"os"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"
)

var (
	NomadContainerName  = "embly-nomad"
	ConsulContainerName = "embly-consul"
	AgentContainerName  = "embly-agent"
	AllContainerNames   = []string{NomadContainerName, ConsulContainerName, AgentContainerName}

	// TODO: tag these with the version of the binary release

	NomadImageName  = "embly/nomad"
	ConsulImageName = "embly/consul"
	AgentImageName  = "embly/twelve"
	AllImageNames   = []string{NomadImageName, ConsulImageName, AgentImageName}

	DefaultContainerStopTimeout = 30 // seconds
)

func (ac *APIClient) StartLocalServices() (err error) {
	if err = ac.connectDocker(); err != nil {
		return
	}

	healthy, _ := ac.Healthy()
	// TODO: return version number here and shut down and upgrade if
	// we need to
	if healthy {
		fmt.Println("Embly is already running locally")
		return nil
	}

	// If we're not healthy let's kill and cleanup all of our related containers
	for _, name := range AllContainerNames {
		// do we care about nomad or consul cleanup? do we want to try stopping first?
		if err = ac.dockerClient.KillContainer(docker.KillContainerOptions{
			ID: name,
		}); err != nil {
			switch err := errors.Cause(err).(type) {
			case *docker.ContainerNotRunning:
			case *docker.NoSuchContainer:
			default:
				return err
			}
		}
		if err = ac.dockerClient.RemoveContainer(docker.RemoveContainerOptions{
			ID: name,
		}); err != nil {
			switch err := errors.Cause(err).(type) {
			case *docker.ContainerNotRunning:
			case *docker.NoSuchContainer:
			default:
				return err
			}
		}
	}

	for _, name := range AllImageNames {
		_, err := ac.dockerClient.InspectImage(name + ":latest")
		if err != nil && err.Error() == "no such image" {
			if err = ac.dockerClient.PullImage(docker.PullImageOptions{
				Repository:   "docker.io/" + name,
				Tag:          "latest",
				OutputStream: os.Stdout,
			}, docker.AuthConfiguration{}); err != nil {
				return err
			}
		}
	}

	if _, err = ac.dockerClient.CreateContainer(docker.CreateContainerOptions{
		Name: ConsulContainerName,
		Config: &docker.Config{
			Image: ConsulImageName,
			Cmd:   []string{"consul", "agent", "-dev", "-client", "127.0.0.1"},
		},
		HostConfig: &docker.HostConfig{
			NetworkMode: "host",
		},
		NetworkingConfig: &docker.NetworkingConfig{},
	}); err != nil {
		return
	}
	if _, err = ac.dockerClient.CreateContainer(docker.CreateContainerOptions{
		Name: AgentContainerName,
		Config: &docker.Config{
			Image: AgentImageName,
		},
		HostConfig: &docker.HostConfig{
			NetworkMode: "host",
			CapAdd:      []string{"SYS_ADMIN", "NET_ADMIN"},
			Binds: []string{
				"/var/run/docker.sock:/var/run/docker.sock",
				"/tmp/nomad/data:/tmp/nomad/data",
			},
		},
		NetworkingConfig: &docker.NetworkingConfig{},
	}); err != nil {
		return
	}
	if _, err = ac.dockerClient.CreateContainer(docker.CreateContainerOptions{
		Name: NomadContainerName,
		Config: &docker.Config{
			Image: NomadImageName,
		},
		HostConfig: &docker.HostConfig{
			NetworkMode: "host",
			CapAdd:      []string{"NET_ADMIN"},
		},
		NetworkingConfig: &docker.NetworkingConfig{},
	}); err != nil {
		return
	}

	for _, name := range AllContainerNames {
		if err = ac.dockerClient.StartContainer(name, nil); err != nil {
			return err
		}
	}

	fmt.Println("Services launched successfully")
	return
}

func (ac *APIClient) StopLocalServices() (err error) {
	if err = ac.connectDocker(); err != nil {
		return
	}
	go func() {
		<-time.NewTimer(time.Duration(DefaultContainerStopTimeout) * time.Second / 3).C
		fmt.Println("Waiting for containers to stop")
	}()
	errChan := make(chan error)
	waitCh := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(len(AllContainerNames))
	go func() {
		for _, name := range AllContainerNames {
			go func(name string) {
				if err := ac.dockerClient.StopContainer(name,
					uint(DefaultContainerStopTimeout)); err != nil {
					switch err := errors.Cause(err).(type) {
					case *docker.ContainerNotRunning:
					case *docker.NoSuchContainer:
					default:
						errChan <- err
					}
				}
				wg.Done()
			}(name)
		}
		wg.Wait()
		close(waitCh)
	}()
	select {
	case <-waitCh:
	case err := <-errChan:
		return err
	}
	return
}
