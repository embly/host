package host

// import (
// 	"fmt"

// 	nomad "github.com/hashicorp/nomad/api"
// 	"github.com/hashicorp/nomad/helper"
// )

// func startService(service *Service) {

// 	client, err := nomad.NewClient(&nomad.Config{
// 		Address: "http://localhost:4646",
// 		Region:  "dc1-1",
// 	})
// 	panicOnErr(err)

// 	jobs := client.Jobs()
// 	task := nomad.NewTask(service.Name, "docker")
// 	task.Resources = &nomad.Resources{
// 		CPU:      helper.IntToPtr(service.CPU),
// 		MemoryMB: helper.IntToPtr(service.Memory),
// 		Networks: []*nomad.NetworkResource{
// 			{
// 				MBits: helper.IntToPtr(50),
// 				// TODO
// 				DynamicPorts: []nomad.Port{{Label: "http"}},
// 			},
// 		},
// 	}
// 	task.SetConfig("image", service.Image)

// 	// TODO
// 	task.SetConfig("port_map", []map[string]int{{
// 		"http": 8000,
// 	}})

// 	taskGroup := nomad.NewTaskGroup(service.Image, 1)
// 	taskGroup.AddTask(task)

// 	job := nomad.NewServiceJob(service.Image, service.Image, "dc1", 1)
// 	job.AddDatacenter("dc1")
// 	job.AddTaskGroup(taskGroup)

// 	regResp, _, err := jobs.Register(job, nil)
// 	panicOnErr(err)

// 	fmt.Println(regResp.EvalID)
// 	eval, _, err := client.Evaluations().Info(regResp.EvalID, nil)
// 	panicOnErr(err)

// 	fmt.Println(eval.DeploymentID)

// 	allocs, _, err := client.Deployments().Allocations(eval.DeploymentID, nil)
// 	panicOnErr(err)
// 	for _, alloc := range allocs {
// 		for task, state := range alloc.TaskStates {
// 			fmt.Print(task)
// 			for _, event := range state.Events {
// 				fmt.Print(event.Details)
// 			}
// 			fmt.Println()
// 		}
// 	}

// }
