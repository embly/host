package agent

import (
	"fmt"
	"net"
	"strings"

	"github.com/maxmcd/tester"
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
	tester.Print(cont)

	for _, q := range m.Question {
		// poor way to check if this is just looking for a top level domain
		// proxy domain can just resolve to the same ip (unless there is a port conflict)
		if strings.Index(q.Name, ".") < len(q.Name)-1 {
			rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, a.ip.String()))
			if err == nil {
				m.Answer = append(m.Answer, rr)
			}
		} else {
			for name := range a.allocations[cont.TaskID.allocID].TaskResources {
				if q.Name == name+"." {
					otherCont, ok := a.containers[TaskID{
						allocID: cont.TaskID.allocID,
						name:    name,
					}]
					if ok {
						rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, otherCont.IPAddress))
						if err == nil {
							m.Answer = append(m.Answer, rr)
						}
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
