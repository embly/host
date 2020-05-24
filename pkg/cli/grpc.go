package cli

import (
	"context"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/embly/host/pkg/agent"
	"github.com/embly/host/pkg/pb"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type APIClient struct {
	grpcClient   pb.DeployServiceClient
	dockerClient *docker.Client
}

func NewAPIClient() (c *APIClient, err error) {
	conn, err := grpc.Dial(
		"127.0.0.1:"+strconv.Itoa(agent.DefaultGRPCPort),
		grpc.WithInsecure(),
		grpc.WithConnectParams(grpc.ConnectParams{
			MinConnectTimeout: time.Millisecond * 300,
		}))
	if err != nil {
		return nil, err
	}
	c = &APIClient{}
	c.grpcClient = pb.NewDeployServiceClient(conn)
	return c, nil
}

func (ac *APIClient) connectDocker() (err error) {
	ac.dockerClient, err = docker.NewClientFromEnv()
	return
}

func (ac *APIClient) Healthy() (ok bool, err error) {
	resp, err := ac.grpcClient.Health(context.Background(), nil)
	if err != nil {
		return false, errors.Wrap(err, "error fetching embly client health")
	}
	if resp.Code != 0 {
		return false, errors.New(resp.Msg)
	}
	return true, nil
}

func (ac *APIClient) DeployService(service Service) (err error) {
	pbService := pb.Service{
		Name:       service.Name,
		Count:      int32(service.Count),
		Containers: map[string]*pb.Container{},
	}
	for name, cont := range service.Containers {
		ports := []*pb.Port{}
		for _, port := range cont.ports {
			ports = append(ports, &pb.Port{
				IsUDP:  port.isUDP,
				Number: int32(port.number),
			})
		}
		pbService.Containers[name] = &pb.Container{
			Name:        cont.Name,
			Image:       cont.Image,
			Cpu:         int32(cont.CPU),
			Memory:      int32(cont.Memory),
			Ports:       ports,
			ConnectTo:   cont.ConnectTo,
			Environment: cont.Environment,
		}
	}

	deployClient, err := ac.grpcClient.Deploy(context.Background(), &pb.DeployRequest{
		Services: []*pb.Service{&pbService}})
	if err != nil {
		return
	}
	for {
		var deployResponse *pb.DeployResponse
		deployResponse, err = deployClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return
		}
		if deployResponse.Meta != nil {
			deployMeta := deployResponse.Meta
			if len(deployMeta.Stderr) > 0 {
				_, _ = os.Stderr.Write(deployMeta.Stderr)
			}
			if len(deployMeta.Stdout) > 0 {
				_, _ = os.Stdout.Write(deployMeta.Stdout)
			}
		}
	}
	return nil
}
