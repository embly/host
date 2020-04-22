package agent

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

var c *dns.Client

func (a *Agent) StartDNS() (err error) {
	c = new(dns.Client)
	addr := "172.17.0.1:53"
	server := &dns.Server{Addr: addr, Net: "udp"}
	// dns.HandleFunc("x.", handleX)
	dns.HandleFunc(".", a.handleDNS)
	fmt.Println("serving dns on", addr)
	return server.ListenAndServe()
}

func (a *Agent) getContainerFromAddr(addr string) *Container {
	for _, cont := range a.containers {
		if cont.IPAddress == addr {
			return &cont
		}
	}
	return nil
}

func (a *Agent) addARecord(m *dns.Msg, host, ipaddr string) {
	rr, err := dns.NewRR(fmt.Sprintf("%s A %s", host, ipaddr))
	if err == nil {
		m.Answer = append(m.Answer, rr)
	}
}

func (a *Agent) findDNSResponse(m *dns.Msg, addr net.Addr) bool {
	var addrString string
	if a, ok := addr.(*net.UDPAddr); ok {
		addrString = a.IP.String()
	} else if a, ok := addr.(*net.TCPAddr); ok {
		addrString = a.IP.String()
	}
	if addrString == "" {
		return false
	}

	cont := a.getContainerFromAddr(addrString)
	if cont == nil {
		return false
	}

	for _, q := range m.Question {
		// poor way to check if this is just looking for a top level domain
		// proxy domain can just resolve to the same ip (unless there is a port conflict)
		if strings.Index(q.Name, ".") < len(q.Name)-1 {
			if a.dnsIndex.lookupHost(q.Name) {
				a.addARecord(m, q.Name, a.ip.String())
			}
		} else {
			for name := range a.allocations[cont.TaskID.allocID].TaskResources {
				if q.Name == name+"." {
					otherCont, ok := a.containers[TaskID{
						allocID: cont.TaskID.allocID,
						name:    name,
					}]
					if ok {
						a.addARecord(m, q.Name, otherCont.IPAddress)
						break
					}
				}
			}
		}
	}

	return len(m.Answer) > 0
}

func (a *Agent) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	var err error
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	if !a.findDNSResponse(m, w.RemoteAddr()) {
		m, _, err = c.Exchange(r, "8.8.8.8:53")
		if err != nil {
			logrus.Error(err)
		}
	}

	if err = w.WriteMsg(m); err != nil {
		logrus.Error(err)
	}
}

// TODO track ip addrs here
type DNSIndex struct {
	hostnames map[string]int
	lock      sync.RWMutex
}

func newDNSIndex() *DNSIndex {
	return &DNSIndex{
		hostnames: map[string]int{},
	}
}

func (idx *DNSIndex) lookupHost(host string) bool {
	idx.lock.RLock()
	defer idx.lock.RUnlock()
	_, ok := idx.hostnames[strings.TrimRight(host, ".")]
	return ok
}

func (idx *DNSIndex) addService(s Service) {
	idx.lock.Lock()
	defer idx.lock.Unlock()
	idx.hostnames[s.hostname]++
}

func (idx *DNSIndex) removeService(s Service) {
	idx.lock.Lock()
	defer idx.lock.Unlock()
	idx.hostnames[s.hostname]--
	if idx.hostnames[s.hostname] == 0 {
		delete(idx.hostnames, s.hostname)
	}
}
