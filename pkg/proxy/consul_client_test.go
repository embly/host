package proxy

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
		index:          1,
	}
}

func (fcc *fakeConsulClient) pushUpdate(name string, tags []string, css []*consul.CatalogService) {
	fcc.services[name] = tags
	fcc.catalogService[name] = css
	fcc.index += 1
	fcc.cond.Broadcast()
}

func (fcc *fakeConsulClient) Services(q *consul.QueryOptions) (map[string][]string, *consul.QueryMeta, error) {
	for {
		if q.WaitIndex < fcc.index {
			return fcc.services, &consul.QueryMeta{LastIndex: fcc.index}, nil
		}
		fcc.cond.L.Lock()
		fcc.cond.Wait()
		fcc.cond.L.Unlock()
	}
}

func (fcc *fakeConsulClient) Service(name, tag string) ([]*consul.CatalogService, error) {
	return fcc.catalogService[name], nil
}
