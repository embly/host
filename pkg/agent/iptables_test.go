package agent

import (
	"fmt"
	"log"
	"net"
	"runtime"
	"strings"
	"testing"

	"github.com/embly/host/pkg/docktest"
	"github.com/embly/host/pkg/exec"
	"github.com/embly/host/pkg/iptables"
	"github.com/maxmcd/tester"
)

type fakeIPTables struct {
	rules map[ProxyRule]struct{}
}

func (ipt *fakeIPTables) CreateChains() (err error) { return nil }
func (ipt *fakeIPTables) DeleteChains() (err error) { return nil }
func (ipt *fakeIPTables) GetRules(chain string) (stats []iptables.Stat, err error) {
	panic("unimplemented")
}
func (ipt *fakeIPTables) AddProxyRule(pr ProxyRule) (err error) {
	ipt.rules[pr] = struct{}{}
	return nil
}

func (ipt *fakeIPTables) AddProxyRuleToPrerouting(pr ProxyRule) (err error) {
	ipt.rules[pr] = struct{}{}
	return nil
}

func (ipt *fakeIPTables) CleanUpPreroutingRules() (err error) {
	return nil
}

func (ipt *fakeIPTables) RuleExists(pr ProxyRule) (exists bool, err error) {
	_, exists = ipt.rules[pr]
	return
}
func (ipt *fakeIPTables) DeleteProxyRule(pr ProxyRule) (err error) {
	delete(ipt.rules, pr)
	return
}

func newFakeIptables() (IPTables, error) {
	return &fakeIPTables{
		rules: map[ProxyRule]struct{}{},
	}, nil
}

func isIptablesPermissionsError(err error, t tester.Tester) {
	if runtime.GOOS != "linux" {
		t.Skip("don't test iptables on", runtime.GOOS)
	}
	if err != nil {
		if strings.Contains(err.Error(), "Permission denied") {
			t.Skip("don't have the correct permissions for iptables")
		}
	}
}

func TestPreRouting(te *testing.T) {
	t := tester.New(te)
	ipt, err := NewIPTables(exec.New())()
	t.PanicOnErr(err)

	pr := ProxyRule{
		proxyIP:       "1.2.3.4",
		containerIP:   "2.3.4.5",
		proxyPort:     20201,
		containerPort: 20202,
	}
	err = ipt.AddProxyRuleToPrerouting(pr)
	isIptablesPermissionsError(err, t)
	t.PanicOnErr(err)

	{
		stats, err := ipt.GetRules(prerouting)
		t.PanicOnErr(err)
		var emblyCount int
		for _, stat := range stats {
			if strings.Contains(stat.Options, commentText) {
				emblyCount++
			}
		}
		t.Assert().Equal(emblyCount, 1)
	}

	err = ipt.CleanUpPreroutingRules()
	t.PanicOnErr(err)

	{
		stats, err := ipt.GetRules(prerouting)
		t.PanicOnErr(err)
		var emblyCount int
		for _, stat := range stats {
			if strings.Contains(stat.Options, commentText) {
				emblyCount++
			}
		}
		t.Assert().Equal(emblyCount, 0)
	}

}

func TestBasic(te *testing.T) {
	t := tester.New(te)
	ipt, err := NewIPTables(exec.New())()
	t.PanicOnErr(err)

	t.Print(ipt)
	err = ipt.CreateChains()
	isIptablesPermissionsError(err, t)
	t.PanicOnErr(err)

	pr := ProxyRule{
		proxyIP:       "1.2.3.4",
		containerIP:   "2.3.4.5",
		proxyPort:     20201,
		containerPort: 20202,
	}
	err = ipt.AddProxyRule(pr)
	t.PanicOnErr(err)

	t.Print(ipt.GetRules(emblyPreroutingChain))
	err = ipt.DeleteChains()
	t.PanicOnErr(err)
}

func TestBasicRedirect(te *testing.T) {
	t := tester.New(te)

	ipt, err := NewIPTables(exec.New())()
	t.PanicOnErr(err)

	err = ipt.CreateChains()
	isIptablesPermissionsError(err, t)
	t.PanicOnErr(err)

	pr := ProxyRule{
		proxyIP:       "192.168.86.30",
		containerIP:   "172.17.0.2",
		proxyPort:     80,
		containerPort: 8084,
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
	defer func() { _ = cont.Delete() }()
	t.PanicOnErr(err)
	ipt, err := NewIPTables(docktest.NewExecInterface(cont))()
	t.PanicOnErr(err)

	t.PanicOnErr(ipt.CreateChains())

	pr := ProxyRule{
		proxyIP:       "192.168.86.30",
		containerIP:   "172.17.0.2",
		proxyPort:     80,
		containerPort: 8084,
	}
	t.PanicOnErr(ipt.AddProxyRule(pr))

	exists, err := ipt.RuleExists(pr)
	t.PanicOnErr(err)
	t.Assert().True(exists)

	rules, err := ipt.GetRules(emblyPreroutingChain)
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
	defer func() { _ = contServer.Delete() }()

	contCurl, err := client.ContainerCreateAndStart(docktest.ContainerCreateOptions{
		Name:  "host-simple-curl",
		Image: "nixery.dev/shell/curl",
		Cmd:   []string{"sleep", "1000000"},
	})
	if err != nil {
		t.Error(err)
	}
	defer func() { _ = contCurl.Delete() }()

	contIPTables, err := newDockerIptables("host")
	if err != nil {
		t.Error(err)
	}
	defer func() { _ = contIPTables.Delete() }()

	// TODO: exentually, this should not be allowed, just hitting the container ip and port...
	_, _, err = contCurl.Exec([]string{"curl", fmt.Sprintf("%s:8084", contServer.NetworkSettings.IPAddress)})
	t.PanicOnErr(err)

	ipt, err := NewIPTables(docktest.NewExecInterface(contIPTables))()
	t.PanicOnErr(err)

	pr := ProxyRule{
		containerIP:   contCurl.NetworkSettings.IPAddress,
		proxyIP:       contServer.NetworkSettings.IPAddress,
		proxyPort:     8084,
		containerPort: 8032,
	}
	t.PanicOnErr(ipt.AddProxyRuleToPrerouting(pr))

	stdout, _, err := contCurl.Exec([]string{"curl", fmt.Sprintf("%s:8032", contServer.NetworkSettings.IPAddress)})
	if err != nil {
		fmt.Println(err.Error())
		t.Fatal(err)
	}
	t.Assert().Contains(string(stdout), "hello")

	t.PanicOnErr(ipt.CleanUpPreroutingRules())
	// client.RemoveContainer(docker.RemoveContainerOptions{})
}

func GetFreePort() int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
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
