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

	taskGroup := nomad.NewTaskGroup(service.Name, 1)

	driver := "docker"
	for _, container := range service.Containers {
		task := nomad.NewTask(container.Name, driver)

		dynamicPorts := []nomad.Port{}
		portMap := map[string]int{}
		for _, port := range container.ports {
			dynamicPorts = append(dynamicPorts, nomad.Port{
				Label: fmt.Sprint(port.number),
			})
			portMap[fmt.Sprint(port.number)] = port.number
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
		task.SetConfig("image", container.Image)
		task.SetConfig("port_map", []map[string]int{portMap})
		taskGroup.AddTask(task)
	}
	job.AddTaskGroup(taskGroup)

	return
}
