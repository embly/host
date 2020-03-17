package proxy

import (
	"fmt"
	"net"
)

type Proxy struct {
	ip string
}

func (p *Proxy) foo() (err error) {
	addr, err := net.ResolveTCPAddr("tcp", p.ip+":0")
	if err != nil {
		return
	}
	listener, err := net.ListenTCP("tcp", addr)
	_, _ = listener, err
	fmt.Println(listener.Addr().String())
	return nil
}
