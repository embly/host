package cli

import (
	"context"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/embly/host/pkg/agent"
	"github.com/embly/host/pkg/pb"
	"google.golang.org/grpc"
)

type APIClient struct {
	// grpc client
	client pb.DeployServiceClient
}

func NewAPIClient() (*APIClient, error) {
	conn, err := grpc.Dial(
		"127.0.0.1:"+strconv.Itoa(agent.DefaultGRPCPort),
		grpc.WithInsecure(),
		grpc.WithConnectParams(grpc.ConnectParams{
			MinConnectTimeout: time.Millisecond * 300,
		}))
	if err != nil {
		return nil, err
	}
	c := APIClient{}
	c.client = pb.NewDeployServiceClient(conn)
	return &c, nil
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

	deployClient, err := ac.client.Deploy(context.Background(), &pbService)
	if err != nil {
		return
	}
	for {
		var deployMeta *pb.DeployMeta
		deployMeta, err = deployClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return
		}
		if len(deployMeta.Stderr) > 0 {
			_, _ = os.Stderr.Write(deployMeta.Stderr)
		}
		if len(deployMeta.Stdout) > 0 {
			_, _ = os.Stdout.Write(deployMeta.Stdout)
		}
	}
	return nil
}
