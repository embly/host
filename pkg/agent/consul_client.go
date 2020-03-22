package agent

import (
	consul "github.com/hashicorp/consul/api"
)

type ConsulClient interface {
	Services(q *consul.QueryOptions) (map[string][]string, *consul.QueryMeta, error)
	Service(name, tag string) ([]*consul.CatalogService, error)
}

var _ ConsulClient = &defaultConsulClient{}

type defaultConsulClient struct {
	client *consul.Client
}

func NewConsulClient(config *consul.Config) (cc ConsulClient, err error) {
	client, err := consul.NewClient(config)
	if err != nil {
		return
	}
	cc = &defaultConsulClient{client: client}
	return
}

func (cc *defaultConsulClient) Services(q *consul.QueryOptions) (map[string][]string, *consul.QueryMeta, error) {
	return cc.client.Catalog().Services(q)
}

func (cc *defaultConsulClient) Service(name, tag string) ([]*consul.CatalogService, error) {
	catalog, _, err := cc.client.Catalog().Service(name, tag, nil)
	return catalog, err
}
