package agent

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/google/uuid"
	consul "github.com/hashicorp/consul/api"
	"github.com/maxmcd/tester"
)

func randomPort() int {
	min := 1
	max := 1<<16 - 1
	return rand.Intn(max-min) + min
}

func newCatalogServiceData(name string, count int, tags []string) (string, []string, []*consul.CatalogService) {
	var css []*consul.CatalogService
	for i := 0; i < count; i++ {
		id := uuid.New().String()
		css = append(css, &consul.CatalogService{
			ID:      id,
			Node:    "f0ca171f3b88",
			Address: "127.0.0.1",
			// this is not how consul creates the serviceID, but we just need them to be unique per CatalogService
			// TODO: change 8080 to an actual overrideable port value
			ServiceID:      fmt.Sprintf("_nomad-task-%s-dashboard-%s-%d", id, name, 8080),
			ServiceName:    name,
			ServiceAddress: "127.0.0.1",
			ServiceTags:    tags,
			ServicePort:    randomPort(),
		})
	}
	return name, tags, css
}

func newServiceData(name string, hostname string, protocol string, count int) (string, []string, []*consul.CatalogService) {
	tags := []string{
		fmt.Sprintf("dns_name=%s:%d", hostname, 8080),
		fmt.Sprintf("protocol=%s", protocol),
	}

	if count == 0 {
		count = 1
	}
	return newCatalogServiceData(name, count, tags)
}

func newMockConsulData() (mcc *mockConsulClient, newConsulData func() (*ConsulData, error)) {
	mcc = newFakeConsulClient()
	newConsulData = func() (cd *ConsulData, err error) {
		cd = &ConsulData{
			cc: mcc,
		}
		return cd, err
	}
	return
}

func TestFakeClient(te *testing.T) {
	t := tester.New(te)

	_, _ = NewConsulData()

	mcc, newConsulData := newMockConsulData()
	cd, err := newConsulData()
	t.PanicOnErr(err)

	mcc.services = map[string][]string{
		"44959340f59d497f95b667902990da5f": {"dns_name=dashboard.standalone2:9002", "protocol=tcp"},
		"55e96970a0329e7827c97a64dbafa564": {"dns_name=counter.counter:9001", "protocol=tcp"},
		"cbd014d6d6d0eb94588577eb8cd6aad4": {"dns_name=dashboard.dashboard:9002", "protocol=tcp"},
		"consul":                           {},
	}
	mcc.catalogService = map[string][]*consul.CatalogService{
		"44959340f59d497f95b667902990da5f": {{
			ID:             "12523ec0-4997-4ea7-e776-a8bd6d222461",
			Node:           "f0ca171f3b88",
			Address:        "127.0.0.1",
			ServiceID:      "_nomad-task-16fa5e63-fcbf-54fd-f6f1-88eb57a01590-dashboard-44959340f59d497f95b667902990da5f-9002",
			ServiceName:    "44959340f59d497f95b667902990da5f",
			ServiceAddress: "127.0.0.1",
			ServiceTags:    []string{"dns_name=dashboard.standalone2:9002", "protocol=tcp"},
			ServicePort:    27729,
		}},
		"cbd014d6d6d0eb94588577eb8cd6aad4": {{
			ID:             "12523ec0-4997-4ea7-e776-a8bd6d222461",
			Node:           "f0ca171f3b88",
			Address:        "127.0.0.1",
			ServiceID:      "_nomad-task-5e4ae995-0483-a280-d106-b24dd9251d76-dashboard-cbd014d6d6d0eb94588577eb8cd6aad4-9002",
			ServiceName:    "cbd014d6d6d0eb94588577eb8cd6aad4",
			ServiceAddress: "127.0.0.1",
			ServiceTags:    []string{"dns_name=dashboard.dashboard:9002", "protocol=tcp"},
			ServicePort:    24011,
		}},

		"55e96970a0329e7827c97a64dbafa564": {{
			ID:             "12523ec0-4997-4ea7-e776-a8bd6d222461",
			Node:           "f0ca171f3b88",
			Address:        "127.0.0.1",
			ServiceID:      "_nomad-task-b37ffbde-8990-90a5-1b74-de16225162fd-counter-55e96970a0329e7827c97a64dbafa564-9001",
			ServiceName:    "55e96970a0329e7827c97a64dbafa564",
			ServiceAddress: "127.0.0.1",
			ServiceTags:    []string{"dns_name=counter.counter:9001", "protocol=tcp"},
			ServicePort:    23674,
		}},
	}

	updatesChan := make(chan map[string]Service)
	go cd.Updates(updatesChan)

	{
		services := <-updatesChan
		service := services["counter.counter:9001"]
		t.Assert().Equal("counter.counter", service.hostname)
		t.Assert().Equal(9001, service.port)
		t.Assert().Equal("tcp", service.protocol)
		allocIDs := map[string]string{
			"f0ca171f3b88._nomad-task-16fa5e63-fcbf-54fd-f6f1-88eb57a01590-dashboard-44959340f59d497f95b667902990da5f-9002": "16fa5e63-fcbf-54fd-f6f1-88eb57a01590",
			"f0ca171f3b88._nomad-task-5e4ae995-0483-a280-d106-b24dd9251d76-dashboard-cbd014d6d6d0eb94588577eb8cd6aad4-9002": "5e4ae995-0483-a280-d106-b24dd9251d76",
			"f0ca171f3b88._nomad-task-b37ffbde-8990-90a5-1b74-de16225162fd-counter-55e96970a0329e7827c97a64dbafa564-9001":   "b37ffbde-8990-90a5-1b74-de16225162fd",
		}
		for id, task := range service.inventory {
			t.Assert().Equal(23674, task.port)
			t.Assert().Equal("127.0.0.1", task.address)
			t.Assert().Equal(allocIDs[id], task.allocID, id)
		}
	}

	mcc.pushUpdate("e6306c743ec9d877118629405705a77c",
		[]string{"dns_name=counter.standalone2:9001", "protocol=tcp"},
		[]*consul.CatalogService{{
			ID:             "12523ec0-4997-4ea7-e776-a8bd6d222461",
			Node:           "f0ca171f3b88",
			Address:        "127.0.0.1",
			ServiceID:      "_nomad-task-16fa5e63-fcbf-54fd-f6f1-88eb57a01590-counter-e6306c743ec9d877118629405705a77c-9001",
			ServiceName:    "e6306c743ec9d877118629405705a77c",
			ServiceAddress: "127.0.0.1",
			ServiceTags:    []string{"dns_name=counter.standalone2:9001", "protocol=tcp"},
			ServicePort:    30590,
		}})
	{
		services := <-updatesChan
		service := services["counter.standalone2:9001"]
		t.Assert().Equal("counter.standalone2", service.hostname)
		t.Assert().Equal(9001, service.port)
		for _, task := range service.inventory {
			t.Assert().Equal(30590, task.port)
			t.Assert().Equal("16fa5e63-fcbf-54fd-f6f1-88eb57a01590", task.allocID)
			t.Assert().Equal("127.0.0.1", task.address)
		}
	}
}
