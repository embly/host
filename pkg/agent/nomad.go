package agent

import (
	"go4.org/sort"
)

type ConnectRequest struct {
	// service addresses I'd like to connect to
	desiredServices []string

	taskID TaskID
}

type Allocation struct {
	ID            string
	TaskResources map[string]TaskResource
}

func (alloc *Allocation) allTaskResourcePairs(cb func(a, b TaskResource)) {
	if len(alloc.TaskResources) <= 1 {
		return
	}
	var trs []TaskResource
	for _, tr := range alloc.TaskResources {
		trs = append(trs, tr)
	}
	sort.Slice(trs, func(i, j int) bool { return trs[i].Name > trs[j].Name })

	for _, a := range trs {
		for _, b := range trs {
			if a.Name == b.Name {
				continue
			}
			if len(b.Ports) == 0 {
				// we don't nede to set up network links if it's not listening
				continue
			}
			cb(a, b)
		}
	}
}

type TaskResource struct {
	Name      string
	IPAddress string
	Ports     []ResourcePort
}

type ResourcePort struct {
	// Value is the port the service is actually broascasting on
	Value int
	// Listening is the value the port the container thinks it's listening on
	Listening int
}
