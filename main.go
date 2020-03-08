package main

import (
	"fmt"

	nomad "github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/helper"
	"go.starlark.net/starlark"
)

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

type Service struct {
	name   string
	image  string
	ports  map[int]string
	cpu    int
	memory int
}

func (s *Service) Service(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (v starlark.Value, err error) {
	name := string(args.Index(0).(starlark.String))
	s.name = name
	for _, kwarg := range kwargs {
		key := string(kwarg.Index(0).(starlark.String))
		switch key {
		case "image":
			s.image = string(kwarg.Index(1).(starlark.String))
		case "cpu":
			i, _ := kwarg.Index(1).(starlark.Int).Int64()
			s.cpu = int(i)
		case "memory":
			i, _ := kwarg.Index(1).(starlark.Int).Int64()
			s.memory = int(i)
		}
		fmt.Println(kwarg)
	}
	return starlark.None, nil
}

func main() {

	service := &Service{}

	// Execute Starlark program in a file.
	thread := &starlark.Thread{Name: ""}
	globals, err := starlark.ExecFile(thread, "project.py", nil, starlark.StringDict{
		"service": starlark.NewBuiltin("service", service.Service),
	})
	_ = globals
	panicOnErr(err)
	startService(service)

}

func startService(service *Service) {
	client, err := nomad.NewClient(&nomad.Config{
		Address: "http://localhost:4646",
		Region:  "dc1-1",
	})
	panicOnErr(err)

	jobs := client.Jobs()
	task := nomad.NewTask(service.name, "docker")
	task.Resources = &nomad.Resources{
		CPU:      helper.IntToPtr(service.cpu),
		MemoryMB: helper.IntToPtr(service.memory),
		Networks: []*nomad.NetworkResource{
			{
				MBits: helper.IntToPtr(50),
				// TODO
				DynamicPorts: []nomad.Port{{Label: "http"}},
			},
		},
	}
	task.SetConfig("image", service.image)

	// TODO
	task.SetConfig("port_map", []map[string]int{{
		"http": 8000,
	}})

	taskGroup := nomad.NewTaskGroup(service.image, 1)
	taskGroup.AddTask(task)

	job := nomad.NewServiceJob(service.image, service.image, "dc1", 1)
	job.AddDatacenter("dc1")
	job.AddTaskGroup(taskGroup)

	regResp, _, err := jobs.Register(job, nil)
	panicOnErr(err)

	fmt.Println(regResp.EvalID)
	eval, _, err := client.Evaluations().Info(regResp.EvalID, nil)
	panicOnErr(err)

	fmt.Println(eval.DeploymentID)

	allocs, _, err := client.Deployments().Allocations(eval.DeploymentID, nil)
	panicOnErr(err)
	for _, alloc := range allocs {
		for task, state := range alloc.TaskStates {
			fmt.Print(task)
			for _, event := range state.Events {
				fmt.Print(event.Details)
			}
			fmt.Println()
		}
	}

}
