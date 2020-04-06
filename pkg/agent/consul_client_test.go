package agent

import (
	"sync"

	consul "github.com/hashicorp/consul/api"
)

type fakeConsulClient struct {
	services       map[string][]string
	catalogService map[string][]*consul.CatalogService

	index uint64
	cond  sync.Cond
}

var _ ConsulClient = &fakeConsulClient{}

func newFakeConsulClient() *fakeConsulClient {
	return &fakeConsulClient{
		services:       map[string][]string{},
		catalogService: map[string][]*consul.CatalogService{},
		cond:           sync.Cond{L: &sync.Mutex{}},
		index:          0,
	}
}

func (fcc *fakeConsulClient) pushUpdate(name string, tags []string, css []*consul.CatalogService) {
	fcc.cond.L.Lock()
	defer fcc.cond.L.Unlock()
	fcc.services[name] = tags
	fcc.catalogService[name] = css
	fcc.index++
	fcc.cond.Signal()
}

func (fcc *fakeConsulClient) deleteService(name string) {
	fcc.cond.L.Lock()
	defer fcc.cond.L.Unlock()
	delete(fcc.services, name)
	delete(fcc.catalogService, name)
	fcc.index++
	fcc.cond.Signal()
}

func (fcc *fakeConsulClient) Services(q *consul.QueryOptions) (map[string][]string, *consul.QueryMeta, error) {
	fcc.cond.L.Lock()
	defer fcc.cond.L.Unlock()
	if len(fcc.services) > 0 && fcc.index == 0 {
		fcc.index = 1
	}
	for {
		if q.WaitIndex < fcc.index && fcc.index != 0 {
			return fcc.services, &consul.QueryMeta{LastIndex: fcc.index}, nil
		}
		fcc.cond.Wait()
	}
}

func (fcc *fakeConsulClient) Service(name, tag string) ([]*consul.CatalogService, error) {
	fcc.cond.L.Lock()
	defer fcc.cond.L.Unlock()
	return fcc.catalogService[name], nil
}
