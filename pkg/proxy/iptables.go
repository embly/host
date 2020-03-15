package proxy

import (
	"fmt"
	"strings"

	"github.com/coreos/go-iptables/iptables"
)

type IPTables struct {
	ipt *iptables.IPTables
}

func NewIPTables() (ipt *IPTables, err error) {
	ipt = &IPTables{}
	ipt.ipt, err = iptables.NewWithProtocol(iptables.ProtocolIPv4)
	return
}

var (
	emblyOutputChain     = "EMBLY_OUTPUT"
	emblyPreroutingChain = "EMBLY_PREROUTING"
)

func (ipt *IPTables) CreateChains() (err error) {
	err = ipt.ipt.NewChain("nat", emblyOutputChain)
	if err != nil && !isChainExistsErr(err) {
		return
	}
	err = ipt.ipt.NewChain("nat", emblyPreroutingChain)
	if isChainExistsErr(err) {
		return nil
	}
	var exists bool
	if exists, err = ipt.ipt.Exists("nat", "OUTPUT", "-j", emblyOutputChain); err != nil {
		return
	}
	if !exists {
		if err = ipt.ipt.Append("nat", "OUTPUT", "-j", emblyOutputChain); err != nil {
			return
		}
	}
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
	if err = ipt.ipt.ClearChain("nat", emblyOutputChain); err != nil {
		return
	}
	if err = ipt.ipt.ClearChain("nat", emblyPreroutingChain); err != nil {
		return
	}
	if err = ipt.ipt.Delete("nat", "OUTPUT", "-j", emblyOutputChain); err != nil {
		return
	}
	if err = ipt.ipt.Delete("nat", "PREROUTING", "-j", emblyPreroutingChain); err != nil {
		return
	}
	if err = ipt.ipt.DeleteChain("nat", emblyOutputChain); err != nil {
		return
	}

	return ipt.ipt.DeleteChain("nat", emblyPreroutingChain)
}

type ProxyRule struct {
	localIP         string
	containerIP     string
	requestedPort   int
	destinationPort int
}

func (pr ProxyRule) outputArgs() []string {
	return []string{
		"-p", "tcp",
		"-d", pr.localIP,
		"-s", pr.containerIP,
		"--dport", fmt.Sprint(pr.requestedPort),
		"-j", "DNAT",
		"--to-destination", fmt.Sprintf(":%d", pr.destinationPort),
	}
}

func (pr ProxyRule) preroutingArgs() []string {
	return []string{
		"-p", "tcp",
		"-d", pr.localIP,
		"-s", pr.containerIP,
		"--dport", fmt.Sprint(pr.requestedPort),
		"-j", "REDIRECT",
		"--to-ports", fmt.Sprint(pr.destinationPort),
	}
}

func (ipt *IPTables) natAppendIfNoExist(chain string, args []string) (err error) {
	return ipt.ipt.AppendUnique("nat", chain, args...)
}

func (ipt *IPTables) AddProxyRule(pr ProxyRule) (err error) {
	if err = ipt.natAppendIfNoExist(emblyOutputChain, pr.outputArgs()); err != nil {
		return
	}
	return ipt.natAppendIfNoExist(emblyPreroutingChain, pr.preroutingArgs())
}

func (ipt *IPTables) DeleteProxyRule(pr ProxyRule) (err error) {
	err = ipt.ipt.Delete("nat", emblyOutputChain, pr.outputArgs()...)
	if err != nil {
		return
	}
	return ipt.ipt.Delete("nat", emblyPreroutingChain, pr.preroutingArgs()...)
}

func (ipt *IPTables) GetRules() (stats []iptables.Stat, err error) {
	stats, err = ipt.ipt.StructuredStats("nat", emblyOutputChain)
	if err != nil {
		return
	}
	stats2, err := ipt.ipt.StructuredStats("nat", emblyPreroutingChain)
	stats = append(stats, stats2...)
	return
}

func isChainExistsErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "Chain already exists")
}
