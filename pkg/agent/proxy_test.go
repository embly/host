package agent

import (
	"net"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
	nomad "github.com/hashicorp/nomad/api"
	"github.com/maxmcd/tester"
	"github.com/sirupsen/logrus"
)

func TestNewProxy(te *testing.T) {
	t := tester.New(te)
	_, newConsulData := newMockConsulData()
	_, newNomadData := newMockNomadData()

	_, err := NewProxy(net.IPv4(127, 0, 0, 1),
		noopNewProxySocket,
		newConsulData,
		newNomadData,
		newFakeIptables,
		newFakeDocker,
	)
	t.PanicOnErr(err)
}
func TestProxyBasic(te *testing.T) {
	t := tester.New(te)

	mcc, newConsulData := newMockConsulData()
	_, newNomadData := newMockNomadData()
	testProxy, err := newProxy(net.IPv4(127, 0, 0, 1),
		noopNewProxySocket,
		newConsulData,
		newNomadData,
		newFakeIptables,
		newFakeDocker,
	)
	t.PanicOnErr(err)

	mcc.pushUpdate(newServiceData("thing", "foo.bar", "tcp", 1))
	testProxy.Tick()
	var oldService *Service
	var oldProxy ProxySocket
	{
		service, ok := testProxy.services["foo.bar:8080"]
		oldService = service
		t.Assert().True(ok)
		t.Assert().Equal(service.hostname, "foo.bar")
		t.Assert().Equal(service.port, 8080)

		proxy, ok := testProxy.proxies["foo.bar:8080"]
		t.Assert().True(ok)
		oldProxy = proxy
	}
	t.Assert().True(oldService.alive)

	mcc.pushUpdate(newServiceData("thing", "foo.bar", "tcp", 2))
	testProxy.Tick()

	{
		service, ok := testProxy.services["foo.bar:8080"]
		t.Assert().True(ok)
		t.Assert().Equal(service.hostname, "foo.bar")
		t.Assert().Equal(service.port, 8080)
		t.Assert().Equal(len(service.inventory), 2)
	}

	mcc.pushUpdate(newServiceData("otherthing", "foo.baz", "tcp", 1))
	testProxy.Tick()

	{
		t.Assert().Equal(2, len(testProxy.services))
	}

	mcc.deleteService("thing")
	testProxy.Tick()
	t.Assert().False(oldService.alive)
	{
		t.Assert().Equal(1, len(testProxy.services))
	}
	t.Assert().True(oldProxy.(*noopProxySocket).closed)
}

func TestProxyDockerAndConsulEvents(te *testing.T) {
	logrus.SetReportCaller(true)
	t := tester.New(te)
	mcc, newConsulData := newMockConsulData()
	mnc, newNomadData := newMockNomadData()
	testProxy, err := newProxy(net.IPv4(127, 0, 0, 1),
		noopNewProxySocket,
		newConsulData,
		newNomadData,
		newFakeIptables,
		newFakeDocker,
	)
	t.PanicOnErr(err)

	fd := testProxy.docker.(*fakeDocker)
	fipt := testProxy.ipt.(*fakeIPTables)

	containerIPAddress := "1.2.3.4"
	dockerID := "foo"
	var taskID TaskID
	address := "foo.bar:8080"
	{
		// create one consul service and send the event
		name, tags, css := newServiceData("thing", "foo.bar", "tcp", 1)
		mcc.pushUpdate(name, tags, css)
		t.Assert().Equal(testProxy.Tick(), TickConsul)
	}
	{
		alloc := mockAllocation("job-name", []mockTask{{
			name:      "thang",
			connectTo: []string{address},
		}})
		taskID.allocID = alloc.ID
		taskID.name = "thang"

		mnc.setAllocations([]*nomad.Allocation{alloc})
		t.Assert().Equal(testProxy.Tick(), TickNomad)
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
		t.Assert().Equal(testProxy.Tick(), TickDocker)

		pr, ok := testProxy.rules[taskID.toProxyKey(address)]
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
		testProxy.Tick()
		_, ok := testProxy.rules[taskID.toProxyKey(address)]
		t.Assert().False(ok)
	}
	{
		// delete service, ensure all state is deleted
		mcc.deleteService("thing")
		testProxy.Tick()
		t.Assert().Len(testProxy.services, 0)
		t.Assert().Len(testProxy.proxies, 0)
		t.Assert().Len(testProxy.containers, 0)
		t.Assert().Len(testProxy.rules, 0)
	}
	{
		mnc.setAllocations([]*nomad.Allocation{})
		testProxy.Tick()
		t.Assert().Len(testProxy.connectRequests, 0)
	}
}

func TestProxyDockerAndConsulEventsOutOfOrder(te *testing.T) {
	logrus.SetReportCaller(true)
	t := tester.New(te)
	mcc, newConsulData := newMockConsulData()
	mnc, newNomadData := newMockNomadData()
	testProxy, err := newProxy(net.IPv4(127, 0, 0, 1),
		noopNewProxySocket,
		newConsulData,
		newNomadData,
		newFakeIptables,
		newFakeDocker,
	)
	t.PanicOnErr(err)

	fd := testProxy.docker.(*fakeDocker)

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
		mnc.setAllocations([]*nomad.Allocation{alloc})
		testProxy.Tick()

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
		testProxy.Tick()
		// no rules yet
		t.Assert().Len(testProxy.rules, 0)

	}
	{
		name, tags, css := newServiceData("thing", "foo.bar", "tcp", 1)
		mcc.pushUpdate(name, tags, css)
		testProxy.Tick()
		t.Assert().Equal(testProxy.rules[taskID.toProxyKey(hostname)], ProxyRule{
			proxyIP:       "127.0.0.1",
			containerIP:   "1.2.3.4",
			proxyPort:     0,
			containerPort: 8080,
		})
	}
	{
		// delete service, ensure all state is deleted
		mcc.deleteService("thing")
		testProxy.Tick()
		t.Assert().Len(testProxy.services, 0)
		t.Assert().Len(testProxy.proxies, 0)
		t.Assert().Len(testProxy.containers, 1)
		t.Assert().Len(testProxy.rules, 1)
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
		testProxy.Tick()
		t.Assert().Len(testProxy.services, 0)
		t.Assert().Len(testProxy.proxies, 0)
		t.Assert().Len(testProxy.containers, 0)
		t.Assert().Len(testProxy.rules, 0)
	}
}
