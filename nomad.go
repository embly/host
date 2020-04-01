package host

import (
	"fmt"

	nomad "github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/helper"
	"github.com/pkg/errors"
)

func DeployIsh(job *nomad.Job) (err error) {
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
		return
	}

	fmt.Println(regResp.EvalID)
	eval, _, err := client.Evaluations().Info(regResp.EvalID, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	fmt.Println(eval.DeploymentID)

	allocs, _, err := client.Deployments().Allocations(eval.DeploymentID, nil)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, alloc := range allocs {
		for task, state := range alloc.TaskStates {
			fmt.Print(task)
			for _, event := range state.Events {
				fmt.Print(event.Details)
			}
			fmt.Println()
		}
	}
	return nil
}

func ServiceToJob(service Service) (job *nomad.Job) {
	job = nomad.NewServiceJob(service.Name, service.Name, "dc1", 1)
	job.AddDatacenter("dc1")

	count := service.Count
	if count == 0 {
		count = 1
	}
	taskGroup := nomad.NewTaskGroup(service.Name, count)
	// taskGroup.Networks = []*nomad.NetworkResource{{
	// 	Mode: "bridge",
	// }}

	// TODO: maybe no custom docker driver
	// driver := "docker-embly"
	driver := "docker"
	for _, container := range service.Containers {
		task := nomad.NewTask(container.Name, driver)

		dynamicPorts := []nomad.Port{}
		portMap := map[string]int{}
		services := []*nomad.Service{}
		for _, port := range container.ports {
			dynamicPorts = append(dynamicPorts, nomad.Port{
				Label: fmt.Sprint(port.number),
			})
			portMap[fmt.Sprint(port.number)] = port.number
			// Technically we could overload all the proxy
			// information into one service. If we ever has reason
			// to think the overhead of spawning a consul service for every port
			// is a bad idea
			services = append(services, &nomad.Service{
				Name:      port.consulName(service.Name, container.Name),
				PortLabel: fmt.Sprint(port.number),
				Tags: []string{
					fmt.Sprintf("dns_name=%s.%s:%d", container.Name, service.Name, port.number),
					fmt.Sprintf("protocol=%s", port.protocol()),
				},
			})
		}
		for _, address := range container.ConnectTo {
			services = append(services, &nomad.Service{
				Name: container.connectToConsulName(service.Name, container.Name, address),
				Tags: []string{
					fmt.Sprintf("connect_to=%s", address),
				},
			})
		}
		task.Resources = &nomad.Resources{
			CPU:      helper.IntToPtr(container.CPU),
			MemoryMB: helper.IntToPtr(container.Memory),
			Networks: []*nomad.NetworkResource{
				{
					MBits:        helper.IntToPtr(20),
					DynamicPorts: dynamicPorts,
				},
			},
		}
		task.Env = container.Environment

		// TODO: maybe no custom docker driver
		// task.SetConfig("network_mode", "create_shared")
		// task.SetConfig("network_aliases", []string{container.Name})
		task.SetConfig("image", container.Image)
		task.SetConfig("port_map", []map[string]int{portMap})
		taskGroup.AddTask(task)
		task.Services = services
	}
	job.AddTaskGroup(taskGroup)

	return
}
