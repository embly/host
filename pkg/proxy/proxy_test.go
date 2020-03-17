package proxy

import (
	"testing"
)

func TestFoo(te *testing.T) {
	// t := tester.New(te)
	p := Proxy{ip: "127.0.0.1"}
	p.foo()

}
