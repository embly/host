package agent

import (
	"net"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/google/uuid"
	consul "github.com/hashicorp/consul/api"
	nomad "github.com/hashicorp/nomad/api"
	"github.com/maxmcd/tester"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/rand"
)

type TestAgent struct {
	*Agent
	mcc *mockConsulClient
	mnc *mockNomadClient
}

func NewTestAgent() *TestAgent {
	tp := TestAgent{}
	mcc, newConsulData := newMockConsulData()
	mnc, newNomadData := newMockNomadData()
	tp.mcc = mcc
	tp.mnc = mnc

	proxy, err := newAgent(net.IPv4(127, 0, 0, 1),
		noopNewProxySocket,
		newConsulData,
		newNomadData,
		newFakeIptables,
		newFakeDocker,
	)
	if err != nil {
		panic(err)
	}
	tp.Agent = proxy
	return &tp
}

type jobTestData struct {
	alloc          *nomad.Allocation
	containers     map[string]docker.Container
	startEvents    map[string]*docker.APIEvents
	stopEvents     map[string]*docker.APIEvents
	catalogService map[string]*consul.CatalogService
}

func randomIP() string {
	x := rand.Uint32()
	return net.IPv4(
		byte(x>>24&0xff),
		byte(x>>16&0xff),
		byte(x>>8&0xff),
		byte(x&0xff),
	).String()
}

func (tp *TestAgent) createTestData(info map[string][]mockTask) map[string]jobTestData {
	out := map[string]jobTestData{}
	for jobName, tasks := range info {
		jtd := jobTestData{}
		jtd.alloc = mockAllocation(jobName, tasks)
		jtd.containers = map[string]docker.Container{}
		jtd.startEvents = map[string]*docker.APIEvents{}
		jtd.stopEvents = map[string]*docker.APIEvents{}
		jtd.catalogService = map[string]*consul.CatalogService{}

		for _, task := range tasks {
			dockerID := uuid.New().String()
			jtd.containers[task.name] = docker.Container{
				ID: dockerID,
				NetworkSettings: &docker.NetworkSettings{
					IPAddress: randomIP(),
				},
			}
			jtd.startEvents[task.name] = &docker.APIEvents{
				Action: "start",
				Type:   "container",
				Actor: docker.APIActor{
					Attributes: map[string]string{
						"name": task.name + "-" + jtd.alloc.ID,
					},
					ID: dockerID,
				},
			}
			_, _, css := newServiceData(task.name, task.name+"."+jobName, "tcp", 1)
			jtd.catalogService[task.name] = css[0]
		}

		out[jobName] = jtd
	}
	return out
}

func TestFakeData(te *testing.T) {
	t := tester.New(te)
	_ = t
	tp := NewTestAgent()
	tp.createTestData(map[string][]mockTask{
		"standalone2": {{
			name: "counter",
			ports: []mockTaskPort{{
				label: "9001",
			}},
		}, {
			name: "dashboard",
			ports: []mockTaskPort{{
				label: "9002",
			}},
		}},
	})
}

func TestNewAgent(te *testing.T) {
	t := tester.New(te)
	_, newConsulData := newMockConsulData()
	_, newNomadData := newMockNomadData()

	_, err := NewAgent(net.IPv4(127, 0, 0, 1),
		noopNewProxySocket,
		newConsulData,
		newNomadData,
		newFakeIptables,
		newFakeDocker,
	)
	t.PanicOnErr(err)
}
func TestAgentBasic(te *testing.T) {
	t := tester.New(te)

	mcc, newConsulData := newMockConsulData()
	_, newNomadData := newMockNomadData()
	testAgent, err := newAgent(net.IPv4(127, 0, 0, 1),
		noopNewProxySocket,
		newConsulData,
		newNomadData,
		newFakeIptables,
		newFakeDocker,
	)
	t.PanicOnErr(err)

	mcc.pushUpdate(newServiceData("thing", "foo.bar", "tcp", 1))
	testAgent.Tick()
	var oldService *Service
	var oldProxy ProxySocket
	{
		service, ok := testAgent.services["foo.bar:8080"]
		oldService = service
		t.Assert().True(ok)
		t.Assert().Equal(service.hostname, "foo.bar")
		t.Assert().Equal(service.port, 8080)

		proxy, ok := testAgent.proxies["foo.bar:8080"]
		t.Assert().True(ok)
		oldProxy = proxy
	}
	t.Assert().True(oldService.alive)

	mcc.pushUpdate(newServiceData("thing", "foo.bar", "tcp", 2))
	testAgent.Tick()

	{
		service, ok := testAgent.services["foo.bar:8080"]
		t.Assert().True(ok)
		t.Assert().Equal(service.hostname, "foo.bar")
		t.Assert().Equal(service.port, 8080)
		t.Assert().Equal(len(service.inventory), 2)
	}

	mcc.pushUpdate(newServiceData("otherthing", "foo.baz", "tcp", 1))
	testAgent.Tick()

	{
		t.Assert().Equal(2, len(testAgent.services))
	}

	mcc.deleteService("thing")
	testAgent.Tick()
	t.Assert().False(oldService.alive)
	{
		t.Assert().Equal(1, len(testAgent.services))
	}
	t.Assert().True(oldProxy.(*noopProxySocket).closed)
}

func TestAgentDockerAndConsulEvents(te *testing.T) {
	logrus.SetReportCaller(true)
	t := tester.New(te)
	mcc, newConsulData := newMockConsulData()
	mnc, newNomadData := newMockNomadData()
	testAgent, err := newAgent(net.IPv4(127, 0, 0, 1),
		noopNewProxySocket,
		newConsulData,
		newNomadData,
		newFakeIptables,
		newFakeDocker,
	)
	t.PanicOnErr(err)

	fd := testAgent.docker.(*fakeDocker)
	fipt := testAgent.ipt.(*fakeIPTables)

	containerIPAddress := "1.2.3.4"
	dockerID := "foo"
	var taskID TaskID
	address := "foo.bar:8080"
	{
		// create one consul service and send the event
		name, tags, css := newServiceData("thing", "foo.bar", "tcp", 1)
		mcc.pushUpdate(name, tags, css)
		t.Assert().Equal(testAgent.Tick(), TickConsul)
	}
	{
		alloc := mockAllocation("job-name", []mockTask{{
			name:      "thang",
			connectTo: []string{address},
		}})
		taskID.allocID = alloc.ID
		taskID.name = "thang"

		mnc.setAllocations([]*nomad.Allocation{alloc})
		t.Assert().Equal(testAgent.Tick(), TickNomad)
	}
	{
		// send the event for that docker container starting
		fd.containers[dockerID] = docker.Container{
			ID: dockerID,
			NetworkSettings: &docker.NetworkSettings{
				IPAddress: containerIPAddress,
			},
		}
		fd.listener <- &docker.APIEvents{
			Action: "start",
			Type:   "container",
			Actor: docker.APIActor{
				Attributes: map[string]string{
					"name": taskID.name + "-" + taskID.allocID,
				},
				ID: dockerID,
			},
		}
		t.Assert().Equal(testAgent.Tick(), TickDocker)

		pr, ok := testAgent.rules[taskID.toProxyKey(address)]
		t.Assert().True(ok)
		t.Assert().Equal(containerIPAddress, pr.containerIP)
		_, ok = fipt.rules[pr]
		t.Assert().True(ok)
	}
	{
		// stop the container, ensure the proxyRule is cleared
		fd.listener <- &docker.APIEvents{
			Action: "stop",
			Type:   "container",
			Actor: docker.APIActor{
				Attributes: map[string]string{
					"name": taskID.name + "-" + taskID.allocID,
				},
				ID: dockerID,
			},
		}
		testAgent.Tick()
		_, ok := testAgent.rules[taskID.toProxyKey(address)]
		t.Assert().False(ok)
	}
	{
		// delete service, ensure all state is deleted
		mcc.deleteService("thing")
		testAgent.Tick()
		t.Assert().Len(testAgent.services, 0)
		t.Assert().Len(testAgent.proxies, 0)
		t.Assert().Len(testAgent.containers, 0)
		t.Assert().Len(testAgent.rules, 0)
	}
	{
		mnc.setAllocations([]*nomad.Allocation{})
		testAgent.Tick()
		t.Assert().Len(testAgent.connectRequests, 0)
	}
}

func TestAgentDockerAndConsulEventsOutOfOrder(te *testing.T) {
	logrus.SetReportCaller(true)
	t := tester.New(te)

	proxy := NewTestAgent()

	fd := proxy.docker.(*fakeDocker)

	containerIPAddress := "1.2.3.4"
	dockerID := "foo"
	hostname := "foo.bar:8080"
	var taskID TaskID
	{
		alloc := mockAllocation("job-name", []mockTask{{
			name:      "thang",
			connectTo: []string{hostname},
		}})
		taskID.allocID = alloc.ID
		taskID.name = "thang"
		proxy.mnc.setAllocations([]*nomad.Allocation{alloc})
		proxy.Tick()

		// create a connectTo request and send the docker event first
		fd.containers[dockerID] = docker.Container{
			ID: dockerID,
			NetworkSettings: &docker.NetworkSettings{
				IPAddress: containerIPAddress,
			},
		}
		fd.listener <- &docker.APIEvents{
			Action: "start",
			Type:   "container",
			Actor: docker.APIActor{
				Attributes: map[string]string{
					"name": taskID.name + "-" + taskID.allocID,
				},
				ID: dockerID,
			},
		}
		proxy.Tick()
		// no rules yet
		t.Assert().Len(proxy.rules, 0)
	}
	{
		name, tags, css := newServiceData("thing", "foo.bar", "tcp", 1)
		proxy.mcc.pushUpdate(name, tags, css)
		proxy.Tick()
		t.Assert().Equal(proxy.rules[taskID.toProxyKey(hostname)], ProxyRule{
			proxyIP:       "127.0.0.1",
			containerIP:   "1.2.3.4",
			proxyPort:     0,
			containerPort: 8080,
		})
	}
	{
		// delete service, ensure all state is deleted
		proxy.mcc.deleteService("thing")
		proxy.Tick()
		t.Assert().Len(proxy.services, 0)
		t.Assert().Len(proxy.proxies, 0)
		t.Assert().Len(proxy.containers, 1)
		t.Assert().Len(proxy.rules, 1)
	}

	{
		// stop the container, ensure the proxyRule is cleared
		fd.listener <- &docker.APIEvents{
			Action: "stop",
			Type:   "container",
			Actor: docker.APIActor{
				Attributes: map[string]string{
					"name": taskID.name + "-" + taskID.allocID,
				},
				ID: dockerID,
			},
		}
		proxy.Tick()
		t.Assert().Len(proxy.services, 0)
		t.Assert().Len(proxy.proxies, 0)
		t.Assert().Len(proxy.containers, 0)
		t.Assert().Len(proxy.rules, 0)
	}
}

func TestNomadServicesWithSiblings(te *testing.T) {
	t := tester.New(te)
	tp := NewTestAgent()
	testData := tp.createTestData(map[string][]mockTask{
		"standalone2": {{
			name: "counter",
			ports: []mockTaskPort{{
				label: "9001",
			}},
		}, {
			name: "dashboard",
			ports: []mockTaskPort{{
				label: "9002",
			}},
		}},
	})["standalone2"]

	tp.mnc.setAllocations([]*nomad.Allocation{testData.alloc})
	tp.Tick()
	t.Assert().NotEmpty(tp.allocations[testData.alloc.ID])
	t.Assert().Equal(tp.allocations[testData.alloc.ID].TaskResources["counter"].Name, "counter")

	fd := tp.docker.(*fakeDocker)

	counterCont := testData.containers["counter"]
	fd.containers[counterCont.ID] = counterCont
	fd.listener <- testData.startEvents["counter"]
	tp.Tick()

	t.Assert().NotEmpty(tp.containers[TaskID{
		allocID: testData.alloc.ID,
		name:    "counter",
	}])

	dashboardCont := testData.containers["dashboard"]
	fd.containers[dashboardCont.ID] = dashboardCont
	fd.listener <- testData.startEvents["dashboard"]
	tp.Tick()

	t.Assert().NotEmpty(tp.containers[TaskID{
		allocID: testData.alloc.ID,
		name:    "dashboard",
	}])
}
