package agent

import (
	"fmt"
	"strings"

	"github.com/embly/host/pkg/exec"
	"github.com/embly/host/pkg/iptables"
)

type IPTables struct {
	ipt *iptables.IPTables
}

func NewIPTables(execInterface exec.Interface) (ipt *IPTables, err error) {
	ipt = &IPTables{}
	ipt.ipt, err = iptables.NewWithProtocol(iptables.ProtocolIPv4, execInterface)
	return
}

var (
	emblyPreroutingChain = "EMBLY_PREROUTING"
)

func (ipt *IPTables) CreateChains() (err error) {
	err = ipt.ipt.NewChain("nat", emblyPreroutingChain)
	if isChainExistsErr(err) {
		return nil
	}
	var exists bool
	if exists, err = ipt.ipt.Exists("nat", "PREROUTING", "-j", emblyPreroutingChain); err != nil {
		return
	}
	if !exists {
		if err = ipt.ipt.Append("nat", "PREROUTING", "-j", emblyPreroutingChain); err != nil {
			return
		}
	}
	return
}

func (ipt *IPTables) DeleteChains() (err error) {
	if err = ipt.ipt.ClearChain("nat", emblyPreroutingChain); err != nil {
		return
	}
	if err = ipt.ipt.Delete("nat", "PREROUTING", "-j", emblyPreroutingChain); err != nil {
		return
	}

	return ipt.ipt.DeleteChain("nat", emblyPreroutingChain)
}

type ProxyRule struct {
	proxyIP         string
	containerIP     string
	requestedPort   int
	destinationPort int
}

func (pr ProxyRule) preroutingArgs() []string {
	// sudo iptables -t nat -A PREROUTING -p tcp -d 192.168.86.30 --dport 80 -j DNAT --to-destination 192.168.86.30:8084
	return []string{
		"--protocol", "tcp",
		"--destination", pr.proxyIP,
		"--source", pr.containerIP,
		"--dport", fmt.Sprint(pr.requestedPort),
		"--jump", "DNAT",
		"--to-destination", fmt.Sprintf("%s:%d", pr.proxyIP, pr.destinationPort),
	}
}

func (ipt *IPTables) natAppendIfNoExist(chain string, args []string) (err error) {
	return ipt.ipt.AppendUnique("nat", chain, args...)
}

func (ipt *IPTables) AddProxyRule(pr ProxyRule) (err error) {
	return ipt.natAppendIfNoExist(emblyPreroutingChain, pr.preroutingArgs())
}

func (ipt *IPTables) RuleExists(pr ProxyRule) (exists bool, err error) {
	return ipt.ipt.Exists("nat", emblyPreroutingChain, pr.preroutingArgs()...)
}

func (ipt *IPTables) DeleteProxyRule(pr ProxyRule) (err error) {
	return ipt.ipt.Delete("nat", emblyPreroutingChain, pr.preroutingArgs()...)
}

func (ipt *IPTables) GetRules() (stats []iptables.Stat, err error) {
	return ipt.ipt.StructuredStats("nat", emblyPreroutingChain)
}

func isChainExistsErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "Chain already exists")
}
