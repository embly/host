package agent

import (
	"sync"
	"testing"

	"github.com/maxmcd/tester"
)

func TestProxyBasic(te *testing.T) {
	t := tester.New(te)

	fcc := newFakeConsulClient()
	cd := &defaultConsulData{
		cc: fcc,
	}

	testProxy := &Proxy{
		proxyGenerator: &noopProxySocketGen{},
		cd:             cd,
		cond:           sync.Cond{L: &sync.Mutex{}},
	}

	go testProxy.Start()

	fcc.pushUpdate(newServiceData("thing", "foo.bar", 8080, "tcp", 1))
	testProxy.wait()
	var oldService *Service
	var oldProxy ProxySocket
	{
		testProxy.lock.Lock()
		service, ok := testProxy.inventory["foo.bar:8080"]
		oldService = service
		t.Assert().True(ok)
		t.Assert().Equal(service.hostname, "foo.bar")
		t.Assert().Equal(service.port, 8080)

		proxy, ok := testProxy.proxies["foo.bar:8080"]
		t.Assert().True(ok)
		oldProxy = proxy
		testProxy.lock.Unlock()
	}
	t.Assert().True(oldService.alive)

	fcc.pushUpdate(newServiceData("thing", "foo.bar", 8080, "tcp", 2))
	testProxy.wait()

	{
		testProxy.lock.Lock()
		service, ok := testProxy.inventory["foo.bar:8080"]
		t.Assert().True(ok)
		t.Assert().Equal(service.hostname, "foo.bar")
		t.Assert().Equal(service.port, 8080)
		t.Assert().Equal(len(service.inventory), 2)
		testProxy.lock.Unlock()
	}

	fcc.pushUpdate(newServiceData("otherthing", "foo.baz", 8080, "tcp", 1))
	testProxy.wait()

	{
		testProxy.lock.Lock()
		t.Assert().Equal(2, len(testProxy.inventory))
		testProxy.lock.Unlock()
	}

	fcc.deleteService("thing")
	testProxy.wait()
	testProxy.lock.Lock()
	t.Assert().False(oldService.alive)
	{
		t.Assert().Equal(1, len(testProxy.inventory))
	}
	t.Assert().True(oldProxy.(*noopProxySocket).closed)
	testProxy.lock.Unlock()
}
