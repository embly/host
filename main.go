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

func NewProject() *Project {
	return &Project{
		Services:      map[string]Service{},
		LoadBalancers: map[string]LoadBalancer{},
	}
}

type Project struct {
	Services      map[string]Service
	LoadBalancers map[string]LoadBalancer
	Ports         map[string]Port
}

type LoadBalancer struct {
	Name   string
	Port   int
	Type   string
	Target LoadBalancerTarget
}

type LoadBalancerTarget struct {
	Service string
	Port    Port
}

type Service struct {
	Name   string
	Image  string
	Count  int
	Ports  map[string]Port
	CPU    int
	Memory int
}

type Port struct {
	Name    string
	Port    int
	Type    string
	Service string
}

func (p *Project) Port(thread *starlark.Thread, fn *starlark.Builtin,
	args starlark.Tuple, kwargs []starlark.Tuple) (v starlark.Value, err error) {

	name := string(args.Index(0).(starlark.String))
	port := Port{}
	port.Name = name

	if args.Len() > 1 {
		portNumber, _ := args.Index(1).(starlark.Int).Int64()
		port.Port = int(portNumber)
	}
	for _, kwarg := range kwargs {
		key := string(kwarg.Index(0).(starlark.String))
		switch key {
		case "type":
			port.Type = string(kwarg.Index(1).(starlark.String))
		case "port":
			i, _ := kwarg.Index(1).(starlark.Int).Int64()
			port.Port = int(i)
		}
	}
	return
}

func (p *Project) Service(thread *starlark.Thread, fn *starlark.Builtin,
	args starlark.Tuple, kwargs []starlark.Tuple) (v starlark.Value, err error) {
	name := string(args.Index(0).(starlark.String))
	service := Service{}
	service.Name = name
	for _, kwarg := range kwargs {
		key := string(kwarg.Index(0).(starlark.String))
		switch key {
		case "image":
			service.Image = string(kwarg.Index(1).(starlark.String))
		case "cpu":
			i, _ := kwarg.Index(1).(starlark.Int).Int64()
			service.CPU = int(i)
		case "count":
			i, _ := kwarg.Index(1).(starlark.Int).Int64()
			service.Count = int(i)
		case "memory":
			i, _ := kwarg.Index(1).(starlark.Int).Int64()
			service.Memory = int(i)
		}
	}
	p.Services[name] = service

	return starlark.None, nil
}

func (p *Project) LoadBalancer(thread *starlark.Thread, fn *starlark.Builtin,
	args starlark.Tuple, kwargs []starlark.Tuple) (v starlark.Value, err error) {

	name := string(args.Index(0).(starlark.String))
	lb := LoadBalancer{}
	lb.Name = name
	for _, kwarg := range kwargs {
		key := string(kwarg.Index(0).(starlark.String))
		switch key {
		case "type":
			lb.Type = string(kwarg.Index(1).(starlark.String))
		case "port":
			i, _ := kwarg.Index(1).(starlark.Int).Int64()
			lb.Port = int(i)
		case "target":

		}
	}
	p.LoadBalancers[name] = lb
	return
}

func main() {

	project := NewProject()

	// Execute Starlark program in a file.
	thread := &starlark.Thread{Name: ""}
	globals, err := starlark.ExecFile(thread, "project.py", nil, starlark.StringDict{
		"service":       starlark.NewBuiltin("service", project.Service),
		"load_balancer": starlark.NewBuiltin("load_balancer", project.LoadBalancer),
		"port":          starlark.NewBuiltin("port", project.Port),
	})
	_ = globals
	panicOnErr(err)
	// startService(service)

}

func startService(service *Service) {
	client, err := nomad.NewClient(&nomad.Config{
		Address: "http://localhost:4646",
		Region:  "dc1-1",
	})
	panicOnErr(err)

	jobs := client.Jobs()
	task := nomad.NewTask(service.Name, "docker")
	task.Resources = &nomad.Resources{
		CPU:      helper.IntToPtr(service.CPU),
		MemoryMB: helper.IntToPtr(service.Memory),
		Networks: []*nomad.NetworkResource{
			{
				MBits: helper.IntToPtr(50),
				// TODO
				DynamicPorts: []nomad.Port{{Label: "http"}},
			},
		},
	}
	task.SetConfig("image", service.Image)

	// TODO
	task.SetConfig("port_map", []map[string]int{{
		"http": 8000,
	}})

	taskGroup := nomad.NewTaskGroup(service.Image, 1)
	taskGroup.AddTask(task)

	job := nomad.NewServiceJob(service.Image, service.Image, "dc1", 1)
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
