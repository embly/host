package agent

import (
	"io"
	"net"
	"testing"

	"github.com/maxmcd/tester"
)

type noopProxySocket struct {
	closed bool
}

var _ ProxySocket = &noopProxySocket{}

func (ps *noopProxySocket) Addr() net.Addr             { return nil }
func (ps *noopProxySocket) Close() error               { ps.closed = true; return nil }
func (ps *noopProxySocket) ProxyLoop(service *Service) {}
func (ps *noopProxySocket) ListenPort() int            { return 0 }

func noopNewProxySocket(protocol string, ip net.IP, port int) (ProxySocket, error) {
	return &noopProxySocket{}, nil
}

func echoTCPServer(port int) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: port,
	})
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				panic(err)
			}
			go func() { _, err = io.Copy(conn, conn) }()
		}
	}()
}

func TestProxySocketBasic(te *testing.T) {
	t := tester.New(te)

	fcc, newConsulData := newMockConsulData()
	_, newNomadData := newMockNomadData()
	testProxy, err := newAgent(net.IPv4(127, 0, 0, 1),
		NewProxySocket,
		newConsulData,
		newNomadData,
		newFakeIptables,
		newFakeDocker,
	)
	t.PanicOnErr(err)
	port := GetFreePort()
	name, args, catalog := newServiceData("thing", "foo.bar", "tcp", 1)
	catalog[0].ServicePort = port
	echoTCPServer(port)

	fcc.pushUpdate(name, args, catalog)
	testProxy.Tick()
	var proxySocket ProxySocket
	t.Assert().Len(testProxy.proxies, 1)
	for _, ps := range testProxy.proxies {
		proxySocket = ps
		break
	}

	{
		t.Print(proxySocket.Addr().String())
		conn, err := net.Dial("tcp", proxySocket.Addr().String())
		t.PanicOnErr(err)
		msg := []byte("embly")
		_, err = conn.Write(msg)
		t.PanicOnErr(err)
		out := make([]byte, 5)
		_, err = conn.Read(out)
		t.PanicOnErr(err)
		t.Assert().Equal(msg, out)
	}
}

func echoUDPServer(port int) {
	listener, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: port,
	})
	if err != nil {
		panic(err)
	}
	var buffer [4096]byte
	go func() {
		for {
			n, addr, err := listener.ReadFrom(buffer[0:])
			if err != nil {
				panic(err)
			}
			_, err = listener.WriteTo(buffer[0:n], addr)
			if err != nil {
				panic(err)
			}
		}
	}()
}

func TestProxySocketUDPBasic(te *testing.T) {
	t := tester.New(te)

	fcc, newConsulData := newMockConsulData()
	_, newNomadData := newMockNomadData()
	testAgent, err := newAgent(net.IPv4(127, 0, 0, 1),
		NewProxySocket,
		newConsulData,
		newNomadData,
		newFakeIptables,
		newFakeDocker,
	)
	t.PanicOnErr(err)

	port := GetFreePort()
	name, args, catalog := newServiceData("thing", "foo.bar", "udp", 1)
	catalog[0].ServicePort = port
	echoUDPServer(port)

	fcc.pushUpdate(name, args, catalog)
	testAgent.Tick()

	var proxySocket ProxySocket
	for _, ps := range testAgent.proxies {
		proxySocket = ps
		break
	}

	{
		conn, err := net.Dial("udp", proxySocket.Addr().String())
		t.PanicOnErr(err)
		msg := []byte("embly")
		_, err = conn.Write(msg)
		t.PanicOnErr(err)
		out := make([]byte, 5)
		_, err = conn.Read(out)
		t.PanicOnErr(err)
		t.Assert().Equal(msg, out)
	}
}
