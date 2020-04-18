package agent

import (
	"time"

	nomad "github.com/hashicorp/nomad/api"
	"github.com/sirupsen/logrus"
)

type NomadClient interface {
	Self() (*nomad.AgentSelf, error)
	Allocations(nodeID string, q *nomad.QueryOptions) ([]*nomad.Allocation, *nomad.QueryMeta, error)
}

var _ NomadClient = &defaultNomadClient{}

type defaultNomadClient struct {
	client *nomad.Client
}

func NewNomadClient(config *nomad.Config) (nc NomadClient, err error) {
	client, err := nomad.NewClient(config)
	return &defaultNomadClient{client: client}, err
}

func (nc *defaultNomadClient) Self() (self *nomad.AgentSelf, err error) {
	return nc.client.Agent().Self()
}

func (nc *defaultNomadClient) Allocations(nodeID string, q *nomad.QueryOptions) ([]*nomad.Allocation, *nomad.QueryMeta, error) {
	return nc.client.Nodes().Allocations(nodeID, q)
}

type NomadData struct {
	client NomadClient
	nodeID string
}

func NewNomadData() (nd *NomadData, err error) {
	client, err := NewNomadClient(nomad.DefaultConfig())
	if err != nil {
		return
	}
	nd = &NomadData{
		client: client,
	}
	return
}

func (c *NomadData) SetNodeID() (err error) {
	self, err := c.client.Self()
	if err != nil {
		return
	}
	c.nodeID = self.Stats["client"]["node_id"]
	return
}

func (c *NomadData) Updates(ch chan []*nomad.Allocation) {
	var lastIndex uint64
	var q *nomad.QueryOptions
	for {
		q = &nomad.QueryOptions{WaitIndex: lastIndex}
		allocations, meta, err := c.client.Allocations(c.nodeID, q)
		if err != nil {
			// TODO: what if we can never reconnect?
			logrus.Error(err)
			time.Sleep(time.Second)
			continue
		}
		lastIndex = meta.LastIndex
		ch <- allocations
	}
}
