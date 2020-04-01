package agent

import (
	"net"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/maxmcd/tester"
	"github.com/sirupsen/logrus"
)

func TestNewProxy(te *testing.T) {
	t := tester.New(te)
	fcc, newConsulData := newFakeConsulData()

	testProxy, err := NewProxy(net.IPv4(127, 0, 0, 1),
		noopNewProxySocket,
		newConsulData,
		newFakeIptables,
		newFakeDocker,
	)
	t.PanicOnErr(err)
	_, _ = testProxy, fcc
}
func TestProxyBasic(te *testing.T) {
	t := tester.New(te)

	fcc, newConsulData := newFakeConsulData()

	testProxy, err := newProxy(net.IPv4(127, 0, 0, 1),
		noopNewProxySocket,
		newConsulData,
		newFakeIptables,
		newFakeDocker,
	)
	t.PanicOnErr(err)

	fcc.pushUpdate(newServiceData("thing", "foo.bar", "tcp", 1))
	_ = testProxy.Tick()
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

	fcc.pushUpdate(newServiceData("thing", "foo.bar", "tcp", 2))
	_ = testProxy.Tick()

	{
		service, ok := testProxy.services["foo.bar:8080"]
		t.Assert().True(ok)
		t.Assert().Equal(service.hostname, "foo.bar")
		t.Assert().Equal(service.port, 8080)
		t.Assert().Equal(len(service.inventory), 2)
	}

	fcc.pushUpdate(newServiceData("otherthing", "foo.baz", "tcp", 1))
	_ = testProxy.Tick()

	{
		t.Assert().Equal(2, len(testProxy.services))
	}

	fcc.deleteService("thing")
	_ = testProxy.Tick()
	t.Assert().False(oldService.alive)
	{
		t.Assert().Equal(1, len(testProxy.services))
	}
	t.Assert().True(oldProxy.(*noopProxySocket).closed)
}

func TestProxyDockerAndConsulEvents(te *testing.T) {
	logrus.SetReportCaller(true)
	t := tester.New(te)
	fcc, newConsulData := newFakeConsulData()

	testProxy, err := newProxy(net.IPv4(127, 0, 0, 1),
		noopNewProxySocket,
		newConsulData,
		newFakeIptables,
		newFakeDocker,
	)
	t.PanicOnErr(err)

	fd := testProxy.docker.(*fakeDocker)
	fipt := testProxy.ipt.(*fakeIPTables)
	_, _, _ = fd, fipt, fcc

	containerIPAddress := "1.2.3.4"
	dockerID := "foo"
	var allocID string
	{
		// create one consul service and send the event
		name, tags, css := newServiceData("thing", "foo.bar", "tcp", 1)
		fcc.pushUpdate(name, tags, css)
		allocID = parseAllocIDFromServiceID(css[0].ServiceID)
		_ = testProxy.Tick()
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
					NomadAllocKey: allocID,
				},
				ID: dockerID,
			},
		}
		_ = testProxy.Tick()
		pr, ok := testProxy.rules[allocID]
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
					NomadAllocKey: allocID,
				},
				ID: dockerID,
			},
		}
		_ = testProxy.Tick()
		_, ok := testProxy.rules[allocID]
		t.Assert().False(ok)
	}
	{
		// delete service, ensure all state is deleted
		fcc.deleteService("thing")
		_ = testProxy.Tick()
		t.Assert().Len(testProxy.services, 0)
		t.Assert().Len(testProxy.proxies, 0)
		t.Assert().Len(testProxy.containerAllocs, 0)
		t.Assert().Len(testProxy.rules, 0)
	}
}

func TestProxyDockerAndConsulEventsOutOfOrder(te *testing.T) {
	logrus.SetReportCaller(true)
	t := tester.New(te)
	fcc, newConsulData := newFakeConsulData()

	testProxy, err := newProxy(net.IPv4(127, 0, 0, 1),
		noopNewProxySocket,
		newConsulData,
		newFakeIptables,
		newFakeDocker,
	)
	t.PanicOnErr(err)

	fd := testProxy.docker.(*fakeDocker)
	fipt := testProxy.ipt.(*fakeIPTables)
	_, _, _ = fd, fipt, fcc

	containerIPAddress := "1.2.3.4"
	dockerID := "foo"
	var allocID string
	{
		name, tags, css := newServiceData("thing", "foo.bar", "tcp", 1)
		_, _, _ = name, tags, css
		allocID = parseAllocIDFromServiceID(css[0].ServiceID)
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
					NomadAllocKey: allocID,
				},
				ID: dockerID,
			},
		}
		_ = testProxy.Tick()
		fcc.pushUpdate(name, tags, css)
		_ = testProxy.Tick()
		pr, ok := testProxy.rules[allocID]
		t.Assert().True(ok)
		t.Assert().Equal(containerIPAddress, pr.containerIP)
		_, ok = fipt.rules[pr]
		t.Assert().True(ok)
	}
	{
		// delete service, ensure all state is deleted
		fcc.deleteService("thing")
		_ = testProxy.Tick()
		t.Assert().Len(testProxy.services, 0)
		t.Assert().Len(testProxy.proxies, 0)
		t.Assert().Len(testProxy.containerAllocs, 1)
		t.Assert().Len(testProxy.rules, 1)
	}

	{
		// stop the container, ensure the proxyRule is cleared
		fd.listener <- &docker.APIEvents{
			Action: "stop",
			Type:   "container",
			Actor: docker.APIActor{
				Attributes: map[string]string{
					NomadAllocKey: allocID,
				},
				ID: dockerID,
			},
		}
		_ = testProxy.Tick()
		t.Assert().Len(testProxy.services, 0)
		t.Assert().Len(testProxy.proxies, 0)
		t.Assert().Len(testProxy.containerAllocs, 0)
		t.Assert().Len(testProxy.rules, 0)
	}
}
