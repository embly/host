package proxy

import (
	"testing"

	"github.com/maxmcd/tester"
)

func TestBasic(te *testing.T) {
	t := tester.New(te)
	ipt, err := NewIPTables()
	t.PanicOnErr(err)

	t.Print(ipt)
	err = ipt.CreateChains()
	t.PanicOnErr(err)

	pr := ProxyRule{
		localIP:         "1.2.3.4",
		containerIP:     "2.3.4.5",
		requestedPort:   20201,
		destinationPort: 20202,
	}
	err = ipt.AddProxyRule(pr)
	t.PanicOnErr(err)

	t.Print(ipt.GetRules())
	err = ipt.DeleteChains()
	t.PanicOnErr(err)

}
