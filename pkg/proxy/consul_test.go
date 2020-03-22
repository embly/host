package proxy

import (
	"testing"

	consul "github.com/hashicorp/consul/api"
	"github.com/maxmcd/tester"
)

func TestFakeClient(te *testing.T) {
	t := tester.New(te)

	_, _ = NewConsulData()

	fcc := newFakeConsulClient()
	cd := &defaultConsulData{
		cc: fcc,
	}
	_ = cd

	fcc.services = map[string][]string{
		"44959340f59d497f95b667902990da5f": []string{"dns-name=dashboard.standalone2:9002", "protocol=tcp"},
		"55e96970a0329e7827c97a64dbafa564": []string{"dns-name=counter.counter:9001", "protocol=tcp"},
		"cbd014d6d6d0eb94588577eb8cd6aad4": []string{"dns-name=dashboard.dashboard:9002", "protocol=tcp"},
		"consul":                           []string{},
	}
	fcc.catalogService = map[string][]*consul.CatalogService{
		"44959340f59d497f95b667902990da5f": []*consul.CatalogService{{
			ID:             "12523ec0-4997-4ea7-e776-a8bd6d222461",
			Node:           "f0ca171f3b88",
			Address:        "127.0.0.1",
			ServiceID:      "_nomad-task-16fa5e63-fcbf-54fd-f6f1-88eb57a01590-dashboard-44959340f59d497f95b667902990da5f-9002",
			ServiceName:    "44959340f59d497f95b667902990da5f",
			ServiceAddress: "127.0.0.1",
			ServiceTags:    []string{"dns-name=dashboard.standalone2:9002", "protocol=tcp"},
			ServicePort:    27729,
		}},
		"cbd014d6d6d0eb94588577eb8cd6aad4": []*consul.CatalogService{{
			ID:             "12523ec0-4997-4ea7-e776-a8bd6d222461",
			Node:           "f0ca171f3b88",
			Address:        "127.0.0.1",
			ServiceID:      "_nomad-task-5e4ae995-0483-a280-d106-b24dd9251d76-dashboard-cbd014d6d6d0eb94588577eb8cd6aad4-9002",
			ServiceName:    "cbd014d6d6d0eb94588577eb8cd6aad4",
			ServiceAddress: "127.0.0.1",
			ServiceTags:    []string{"dns-name=dashboard.dashboard:9002", "protocol=tcp"},
			ServicePort:    24011,
		}},

		"55e96970a0329e7827c97a64dbafa564": []*consul.CatalogService{{
			ID:             "12523ec0-4997-4ea7-e776-a8bd6d222461",
			Node:           "f0ca171f3b88",
			Address:        "127.0.0.1",
			ServiceID:      "_nomad-task-b37ffbde-8990-90a5-1b74-de16225162fd-counter-55e96970a0329e7827c97a64dbafa564-9001",
			ServiceName:    "55e96970a0329e7827c97a64dbafa564",
			ServiceAddress: "127.0.0.1",
			ServiceTags:    []string{"dns-name=counter.counter:9001", "protocol=tcp"},
			ServicePort:    23674,
		}},
	}

	updatesChan := make(chan map[string]Service)
	go cd.Updates(updatesChan)

	{
		inventory := <-updatesChan
		service := inventory["counter.counter:9001"]
		t.Assert().Equal("counter.counter", service.hostname)
		t.Assert().Equal(9001, service.port)
		for _, task := range service.inventory {
			t.Assert().Equal(23674, task.port)
			t.Assert().Equal("127.0.0.1", task.address)
		}
	}

	fcc.pushUpdate("e6306c743ec9d877118629405705a77c",
		[]string{"dns-name=counter.standalone2:9001", "protocol=tcp"},
		[]*consul.CatalogService{{
			ID:             "12523ec0-4997-4ea7-e776-a8bd6d222461",
			Node:           "f0ca171f3b88",
			Address:        "127.0.0.1",
			ServiceID:      "_nomad-task-16fa5e63-fcbf-54fd-f6f1-88eb57a01590-counter-e6306c743ec9d877118629405705a77c-9001",
			ServiceName:    "e6306c743ec9d877118629405705a77c",
			ServiceAddress: "127.0.0.1",
			ServiceTags:    []string{"dns-name=counter.standalone2:9001", "protocol=tcp"},
			ServicePort:    30590,
		}})

	{
		inventory := <-updatesChan
		service := inventory["counter.standalone2:9001"]
		t.Assert().Equal("counter.standalone2", service.hostname)
		t.Assert().Equal(9001, service.port)
		for _, task := range service.inventory {
			t.Assert().Equal(30590, task.port)
			t.Assert().Equal("127.0.0.1", task.address)
		}
	}

}
