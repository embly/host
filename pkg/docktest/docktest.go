package docktest

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/embly/host/pkg/exec"
	docker "github.com/fsouza/go-dockerclient"
)

type Client struct {
	*docker.Client
}

var client *Client

func NewClient() (*Client, error) {
	c, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	client = &Client{Client: c}
	return client, nil
}

type Container struct {
	*docker.Container
}

func (cont *Container) Delete() (err error) {
	_ = client.StopContainer(cont.ID, 0) // might already be stopped
	return client.RemoveContainer(docker.RemoveContainerOptions{ID: cont.ID})
}

func (cont *Container) Exec(cmd []string) (stdout []byte, stderr []byte, err error) {
	dockerExec, err := client.CreateExec(docker.CreateExecOptions{
		Container:    cont.ID,
		Cmd:          cmd,
		AttachStderr: true,
		AttachStdout: true,
	})
	if err != nil {
		return
	}
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	if err = client.StartExec(dockerExec.ID, docker.StartExecOptions{
		OutputStream: &stdoutBuf,
		ErrorStream:  &stderrBuf,
	}); err != nil {
		return
	}
	execInspect, err := client.InspectExec(dockerExec.ID)
	if err != nil {
		return
	}
	if execInspect.ExitCode != 0 {
		err = exec.CodeExitError{
			Code: execInspect.ExitCode,
		}
		return
	}

	return stdoutBuf.Bytes(), stderrBuf.Bytes(), nil
}

type ContainerCreateOptions struct {
	Name        string
	Image       string
	Cmd         []string
	Ports       []string
	CapAdd      []string
	NetworkMode string
}

func (client *Client) ContainerCreateAndStart(opt ContainerCreateOptions) (cont *Container, err error) {

	expPorts := map[docker.Port]struct{}{}
	bindings := map[docker.Port][]docker.PortBinding{}
	for _, port := range opt.Ports {
		formatted := docker.Port(fmt.Sprintf("%s/tcp", port))
		expPorts[formatted] = struct{}{}
		bindings[formatted] = []docker.PortBinding{{HostIP: "127.0.0.1", HostPort: port}}
	}
	opts := docker.CreateContainerOptions{
		Name: opt.Name,
		Config: &docker.Config{
			Cmd:          opt.Cmd,
			Image:        opt.Image,
			ExposedPorts: expPorts,
		},
		HostConfig: &docker.HostConfig{
			CapAdd:          opt.CapAdd,
			PublishAllPorts: true,
			PortBindings:    bindings,
			NetworkMode:     opt.NetworkMode,
		},
	}
	var dcont *docker.Container
	dcont, err = client.CreateContainer(opts)
	if err == docker.ErrNoSuchImage {
		parts := strings.Split(opt.Image, ":")
		var rep string
		tag := "latest"
		rep = parts[0]
		if len(parts) == 2 {
			tag = parts[1]
		}
		fmt.Println("pulling image with repo", rep, "and tag", tag)
		if err = client.PullImage(docker.PullImageOptions{
			Repository: rep,
			Tag:        tag,
		}, docker.AuthConfiguration{}); err != nil {
			return
		}
		if dcont, err = client.CreateContainer(opts); err != nil {
			return
		}
	}
	if err == docker.ErrContainerAlreadyExists {
		if err = client.ContainerDeleteByName(opt.Name); err != nil {
			return
		}
		if dcont, err = client.CreateContainer(opts); err != nil {
			return
		}
	}
	if err != nil {
		return
	}
	if err = client.StartContainer(dcont.ID, nil); err != nil {
		return
	}
	cont = &Container{Container: dcont}
	return
}

func (client *Client) ContainerDeleteByName(name string) (err error) {
	cs, err := client.ListContainers(docker.ListContainersOptions{
		All: true,
		Filters: map[string][]string{
			"name": []string{name},
		},
	})

	if err != nil {
		return
	}
	if len(cs) == 0 {
		// container already deleted?
		return nil
	}
	if cs[0].State == "running" {
		if err = client.StopContainer(cs[0].ID, 0); err != nil {
			return
		}
	}
	return client.RemoveContainer(docker.RemoveContainerOptions{
		ID: cs[0].ID,
	})

}
