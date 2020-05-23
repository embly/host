package agent

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/embly/host/pkg/pb"
	nomad "github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/helper"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var DefaultGRPCPort = 5477

func (a *Agent) StartDeployServer() (err error) {
	portNumber := DefaultGRPCPort
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", portNumber))
	if err != nil {
		return
	}
	a.grpcServer = grpc.NewServer()
	pb.RegisterDeployServiceServer(a.grpcServer, a)
	go func() {
		panic(a.grpcServer.Serve(lis))
	}()
	return nil
}

var _ pb.DeployServiceServer = &Agent{}

func ServiceToJob(service pb.Service) (job *nomad.Job) {
	job = nomad.NewServiceJob(service.Name, service.Name, "dc1", 1)
	job.AddDatacenter("dc1")

	count := service.Count
	if count == 0 {
		count = 1
	}
	taskGroup := nomad.NewTaskGroup(service.Name, int(count))

	driver := "docker"
	for _, container := range service.Containers {
		task := nomad.NewTask(container.Name, driver)

		dynamicPorts := []nomad.Port{}
		portMap := map[string]int{}
		services := []*nomad.Service{}
		for _, port := range container.Ports {
			dynamicPorts = append(dynamicPorts, nomad.Port{
				Label: fmt.Sprint(port.Number),
			})
			portMap[fmt.Sprint(port.Number)] = int(port.Number)
			// Technically we could overload all the proxy
			// information into one service. If we ever has reason
			// to think the overhead of spawning a consul service for every port
			// is a bad idea
			services = append(services, &nomad.Service{
				Name:      port.ConsulName(service.Name, container.Name),
				PortLabel: fmt.Sprint(port.Number),
				Tags: []string{
					fmt.Sprintf("dns_name=%s.%s:%d", container.Name, service.Name, port.Number),
					fmt.Sprintf("protocol=%s", port.Protocol()),
				},
			})
		}
		if len(container.ConnectTo) > 0 {
			task.Meta = map[string]string{
				"connect_to": strings.Join(container.ConnectTo, ","),
			}
		}
		task.Resources = &nomad.Resources{
			CPU:      helper.IntToPtr(int(container.Cpu)),
			MemoryMB: helper.IntToPtr(int(container.Memory)),
			Networks: []*nomad.NetworkResource{
				{
					MBits:        helper.IntToPtr(20),
					DynamicPorts: dynamicPorts,
				},
			},
		}
		task.Env = container.Environment

		task.SetConfig("image", container.Image)
		task.SetConfig("runtime", "runsc") // gvisor
		task.SetConfig("port_map", []map[string]int{portMap})
		task.SetConfig("dns_servers", []string{"172.17.0.1"})
		taskGroup.AddTask(task)
		task.Services = services
	}
	job.AddTaskGroup(taskGroup)

	return
}

func DeployIsh(job *nomad.Job, srv pb.DeployService_DeployServer) (err error) {
	print := func(v ...interface{}) {
		_ = srv.Send(&pb.DeployMeta{
			Stdout: []byte(fmt.Sprintln(v...)),
		})
	}

	client, err := nomad.NewClient(&nomad.Config{
		Address: "http://localhost:4646",
		Region:  "dc1-1",
	})
	if err != nil {
		return
	}
	jobs := client.Jobs()

	regResp, _, err := jobs.Register(job, nil)
	if err != nil {
		return errors.WithStack(err)
	}
	print("eval id", regResp.EvalID)

	eval, _, err := client.Evaluations().Info(regResp.EvalID, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	print("deployment id", eval.DeploymentID)

	allocs, _, err := client.Deployments().Allocations(eval.DeploymentID, nil)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, alloc := range allocs {
		for task, state := range alloc.TaskStates {
			print(task)
			for _, event := range state.Events {
				print(event.Details)
			}
			print()
		}
	}
	return nil
}

func (a *Agent) Deploy(req *pb.Service, srv pb.DeployService_DeployServer) error {
	logrus.Infof("new DeployService.Deploy(%v)", req)
	job := ServiceToJob(*req)
	err := DeployIsh(job, srv)
	if err != nil {
		_ = srv.Send(&pb.DeployMeta{
			Stderr: []byte(fmt.Sprintf("%+v\n", err)),
		})
	}
	return err
}

func (a *Agent) Health(ctx context.Context, req *pb.HealthRequest) (resp *pb.HealthResponse, err error) {
	logrus.Infof("new DeployService.Health(%v)", req)
	return &pb.HealthResponse{
		Code: 0,
		Msg:  "ok",
	}, nil
}
