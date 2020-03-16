package proxy

import (
	"fmt"
	"log"
	"net"
	"os/user"
	"runtime"
	"testing"

	"github.com/embly/host/pkg/docktest"
	"github.com/embly/host/pkg/exec"
	"github.com/maxmcd/tester"
)

func shouldWeTestLocalIPTables(t tester.Tester) {
	if runtime.GOOS != "linux" {
		t.Skip("don't test iptables on", runtime.GOOS)
	}
	user, err := user.Current()
	t.PanicOnErr(err)
	if user.Name != "root" {
		t.Skip("don't run iptables tests if you're not root")
	}
}

func TestBasic(te *testing.T) {
	t := tester.New(te)
	shouldWeTestLocalIPTables(t)
	ipt, err := NewIPTables(exec.New())
	t.PanicOnErr(err)

	t.Print(ipt)
	err = ipt.CreateChains()
	t.PanicOnErr(err)

	pr := ProxyRule{
		proxyIP:         "1.2.3.4",
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

func TestBasicRedirect(te *testing.T) {
	t := tester.New(te)
	shouldWeTestLocalIPTables(t)

	ipt, err := NewIPTables(exec.New())
	t.PanicOnErr(err)

	t.PanicOnErr(ipt.CreateChains())

	pr := ProxyRule{
		proxyIP:         "192.168.86.30",
		containerIP:     "172.17.0.2",
		requestedPort:   80,
		destinationPort: 8084,
	}
	t.PanicOnErr(ipt.AddProxyRule(pr))

	exists, err := ipt.RuleExists(pr)
	t.PanicOnErr(err)
	t.Assert().True(exists)

	t.PanicOnErr(
		ipt.DeleteProxyRule(pr))

	t.PanicOnErr(
		ipt.DeleteChains())

}

func newDockerIptables(networkMode ...string) (cont *docktest.Container, err error) {
	client, err := docktest.NewClient()
	if err != nil {
		return
	}
	netMode := ""
	if len(networkMode) > 0 {
		netMode = networkMode[0]
	}
	cont, err = client.ContainerCreateAndStart(docktest.ContainerCreateOptions{
		Name:        "host-iptables",
		Image:       "nixery.dev/shell/iptables/busybox",
		Cmd:         []string{"sleep", "1000000"},
		CapAdd:      []string{"NET_ADMIN"},
		NetworkMode: netMode,
	})
	if err != nil {
		return
	}
	if _, _, err = cont.Exec([]string{"bash", "-c", "mkdir -p /run && touch /run/xtables.lock"}); err != nil {
		return
	}
	return
}

func TestBasicRedirectWithinDocker(te *testing.T) {
	t := tester.New(te)

	cont, err := newDockerIptables()
	defer cont.Delete()
	t.PanicOnErr(err)
	ipt, err := NewIPTables(docktest.NewExecInterface(cont))
	t.PanicOnErr(err)

	t.PanicOnErr(ipt.CreateChains())

	pr := ProxyRule{
		proxyIP:         "192.168.86.30",
		containerIP:     "172.17.0.2",
		requestedPort:   80,
		destinationPort: 8084,
	}
	t.PanicOnErr(ipt.AddProxyRule(pr))

	exists, err := ipt.RuleExists(pr)
	t.PanicOnErr(err)
	t.Assert().True(exists)

	rules, err := ipt.GetRules()
	t.PanicOnErr(err)

	if len(rules) == 0 {
		t.Fatal("should have returned rules")
	}
	t.Assert().Equal(rules[0].Target, "DNAT")
	t.Assert().Equal(rules[0].Protocol, "tcp")
	t.PanicOnErr(
		ipt.DeleteProxyRule(pr))

	t.PanicOnErr(
		ipt.DeleteChains())

}

func TestFull(te *testing.T) {
	// just for iptables:
	//
	// spin up a container with a simple server
	// spin up a container with curl installed
	//
	// spin up a container with net=host and cap-add=net_admin and run the
	// iptables commands in that container
	//
	// then check if the curl container can access the simple server

	t := tester.New(te)
	t.Print(GetOutboundIP())
	client, err := docktest.NewClient()
	t.PanicOnErr(err)
	contServer, err := client.ContainerCreateAndStart(docktest.ContainerCreateOptions{
		Name:  "host-simple-server",
		Image: "maxmcd/host-simple-server",
		Cmd:   []string{"/bin/hello", "-ip", "0.0.0.0", "-port", "8084"},
		Ports: []string{"8084"}})
	if err != nil {
		t.Error(err)
	}
	defer contServer.Delete()

	contCurl, err := client.ContainerCreateAndStart(docktest.ContainerCreateOptions{
		Name:  "host-simple-curl",
		Image: "nixery.dev/shell/curl",
		Cmd:   []string{"sleep", "1000000"},
	})
	if err != nil {
		t.Error(err)
	}
	defer contCurl.Delete()

	contIpTables, err := newDockerIptables("host")
	if err != nil {
		t.Error(err)
	}
	defer contIpTables.Delete()

	// TODO: exentually, this should not be allowed, just hitting the container ip and port...
	_, _, err = contCurl.Exec([]string{"curl", fmt.Sprintf("%s:8084", contServer.NetworkSettings.IPAddress)})
	t.PanicOnErr(err)

	ipt, err := NewIPTables(docktest.NewExecInterface(contIpTables))
	t.PanicOnErr(err)

	t.PanicOnErr(ipt.CreateChains())

	pr := ProxyRule{
		proxyIP:         contServer.NetworkSettings.IPAddress,
		containerIP:     contCurl.NetworkSettings.IPAddress,
		requestedPort:   8032,
		destinationPort: 8084,
	}
	t.PanicOnErr(ipt.AddProxyRule(pr))

	stdout, _, err := contCurl.Exec([]string{"curl", fmt.Sprintf("%s:8032", contServer.NetworkSettings.IPAddress)})
	t.PanicOnErr(err)
	t.Assert().Contains(string(stdout), "hello")

	t.PanicOnErr(ipt.DeleteChains())
	// client.RemoveContainer(docker.RemoveContainerOptions{})
}

func TestFullProxy(te *testing.T) {
	// for the full proxy:
	//
	// spin up a container with a simple server
	// spin up a container with curl installed
	//
	// spin up a container with net=host and cap-add=net_admin and run the
	// iptables commands in that container, this container also needs to be hosting the proxy
	// or maybe we run the proxy locally? yeah that's not too bad
	//
	// then check if the curl container can access the simple server
	// t := tester.New(te)

}

// Get preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	// https://stackoverflow.com/questions/23558425/how-do-i-get-the-local-ip-address-in-go#comment100001538_37382208
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
