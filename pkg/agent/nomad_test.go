package agent

import (
	"testing"

	nomad "github.com/hashicorp/nomad/api"
	"github.com/maxmcd/tester"
)

func TestNomad(te *testing.T) {
	t := tester.New(te)
	t.Skip()
	client, err := nomad.NewClient(nomad.DefaultConfig())
	t.PanicOnErr(err)
	agent := client.Agent()

	myself, err := agent.Self()
	t.PanicOnErr(err)
	nodeID := myself.Stats["client"]["node_id"]

	nodes := client.Nodes()

	allocs, _, err := nodes.Allocations(nodeID, &nomad.QueryOptions{})
	t.PanicOnErr(err)
	t.Print(allocs)
}
func TestNomadEvents(te *testing.T) {
	t := tester.New(te)
	t.Skip()
	nd, err := NewNomadData()
	t.PanicOnErr(err)

	err = nd.SetNodeID()
	t.PanicOnErr(err)

	allocChan := make(chan []*nomad.Allocation)

	go nd.Updates(allocChan)

	for allocations := range allocChan {
		for _, alloc := range allocations {
			tester.Print(alloc)
		}
	}
}
