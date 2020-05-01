package agent

import (
	"sync"

	consul "github.com/hashicorp/consul/api"
)

type mockConsulClient struct {
	services       map[string][]string
	catalogService map[string][]*consul.CatalogService

	index uint64
	cond  sync.Cond
}

var _ ConsulClient = &mockConsulClient{}

func newFakeConsulClient() *mockConsulClient {
	return &mockConsulClient{
		services:       map[string][]string{},
		catalogService: map[string][]*consul.CatalogService{},
		cond:           sync.Cond{L: &sync.Mutex{}},
		index:          0,
	}
}

func (mcc *mockConsulClient) pushUpdate(name string, tags []string, css []*consul.CatalogService) {
	mcc.cond.L.Lock()
	defer mcc.cond.L.Unlock()
	mcc.services[name] = tags
	mcc.catalogService[name] = css
	mcc.index++
	mcc.cond.Signal()
}

func (mcc *mockConsulClient) deleteService(name string) {
	mcc.cond.L.Lock()
	defer mcc.cond.L.Unlock()
	if _, ok := mcc.services[name]; !ok {
		panic(name)
	}
	delete(mcc.services, name)
	delete(mcc.catalogService, name)
	mcc.index++
	mcc.cond.Signal()
}

func (mcc *mockConsulClient) Services(q *consul.QueryOptions) (map[string][]string, *consul.QueryMeta, error) {
	mcc.cond.L.Lock()
	defer mcc.cond.L.Unlock()
	if len(mcc.services) > 0 && mcc.index == 0 {
		mcc.index = 1
	}
	for {
		if q.WaitIndex < mcc.index && mcc.index != 0 {
			return mcc.services, &consul.QueryMeta{LastIndex: mcc.index}, nil
		}
		mcc.cond.Wait()
	}
}

func (mcc *mockConsulClient) Service(name, tag string) ([]*consul.CatalogService, error) {
	mcc.cond.L.Lock()
	defer mcc.cond.L.Unlock()
	return mcc.catalogService[name], nil
}
