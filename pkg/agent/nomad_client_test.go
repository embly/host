package agent

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	nomad "github.com/hashicorp/nomad/api"
	"github.com/maxmcd/tester"
)

type mockNomadClient struct {
	allocations []*nomad.Allocation
	self        nomad.AgentSelf

	index uint64
	cond  sync.Cond
}

var _ NomadClient = &mockNomadClient{}

func newMockNomadClient() *mockNomadClient {
	return &mockNomadClient{
		cond:  sync.Cond{L: &sync.Mutex{}},
		index: 0,
	}
}
func newMockNomadData() (mnc *mockNomadClient, newConsulData func() (*NomadData, error)) {
	mnc = newMockNomadClient()
	newConsulData = func() (cd *NomadData, err error) {
		cd = &NomadData{
			client: mnc,
		}
		return cd, err
	}
	return
}

func (mnc *mockNomadClient) setSelf(self nomad.AgentSelf) {
	mnc.cond.L.Lock()
	defer mnc.cond.L.Unlock()
	mnc.self = self
}

func (mnc *mockNomadClient) setAllocations(allocations []*nomad.Allocation) {
	mnc.cond.L.Lock()
	defer mnc.cond.L.Unlock()
	mnc.allocations = allocations
	mnc.index++
	mnc.cond.Signal()
}

func (mnc *mockNomadClient) Allocations(nodeID string, q *nomad.QueryOptions) ([]*nomad.Allocation, *nomad.QueryMeta, error) {
	mnc.cond.L.Lock()
	defer mnc.cond.L.Unlock()
	if len(mnc.allocations) > 0 && mnc.index == 0 {
		mnc.index = 1
	}
	for {
		if q.WaitIndex < mnc.index && mnc.index != 0 {
			return mnc.allocations, &nomad.QueryMeta{LastIndex: mnc.index}, nil
		}
		mnc.cond.Wait()
	}
}
func (mnc *mockNomadClient) Self() (*nomad.AgentSelf, error) {
	mnc.cond.L.Lock()
	defer mnc.cond.L.Unlock()
	return &mnc.self, nil
}

func intPtr(i int) *int {
	return &i
}

type mockTask struct {
	name      string
	ports     []mockTaskPort
	connectTo []string
}

type mockTaskPort struct {
	label string
	value int
}

func mockAllocation(jobName string, mockTasks []mockTask) *nomad.Allocation {
	taskResources := map[string]*nomad.Resources{}
	tasks := []*nomad.Task{}
	for _, task := range mockTasks {
		ports := []nomad.Port{}
		for _, port := range task.ports {
			ports = append(ports, nomad.Port{
				Label: port.label,
				Value: port.value,
			})
		}
		tasks = append(tasks, &nomad.Task{
			Name: task.name,
			Meta: map[string]string{
				"connect_to": strings.Join(task.connectTo, ","),
			},
		})
		taskResources[task.name] = &nomad.Resources{
			CPU:      intPtr(500),
			MemoryMB: intPtr(256),
			Networks: []*nomad.NetworkResource{{
				IP:           "127.0.0.1",
				MBits:        intPtr(20),
				DynamicPorts: ports,
			}},
		}

	}
	return &nomad.Allocation{
		ID:            uuid.New().String(),
		Name:          fmt.Sprintf("%s.%s[0]", jobName, jobName),
		JobID:         jobName,
		TaskResources: taskResources,
		Job: &nomad.Job{
			TaskGroups: []*nomad.TaskGroup{{
				Tasks: tasks,
			}},
		},
	}
}

func TestMockAllocation(te *testing.T) {
	t := tester.New(te)
	alloc := mockAllocation("dashboard2", []mockTask{{
		name: "counter",
		ports: []mockTaskPort{{
			label: "9002",
			value: 23423,
		}},
	}})
	t.Assert().Equal(23423, alloc.TaskResources["counter"].Networks[0].DynamicPorts[0].Value)
}
