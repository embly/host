package proxy

import (
	"testing"

	consul "github.com/hashicorp/consul/api"
	"github.com/maxmcd/tester"
)

func TestConsulBasic(te *testing.T) {
	t := tester.New(te)
	client, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		panic(err)
	}
	services, _, err := client.Catalog().Services(nil)
	t.PanicOnErr(err)

	// for name := range services {
	// 	service, _, err := client.Catalog().Service(name, "", nil)
	// 	t.Print(service[0])
	// 	break
	// 	t.PanicOnErr(err)
	// 	fmt.Println(name, service)
	// }
}
