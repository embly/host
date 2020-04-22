package agent

import (
	"fmt"
	"net"

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
func (a *Agent) findDNSResponse(m *dns.Msg, q dns.Question, addr net.Addr) {
	var addrString string
	if a, ok := addr.(*net.UDPAddr); ok {
		addrString = a.IP.String()
	} else if a, ok := addr.(*net.TCPAddr); ok {
		addrString = a.IP.String()
	}
	if addrString == "" {
		return
	}
	for _, cont := range a.containers {
		if cont.IPAddress == addrString {
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
						return

					}
				}
			}
		}
	}
}
func (a *Agent) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false
	for _, q := range r.Question {
		fmt.Println(r)
		a.findDNSResponse(m, q, w.RemoteAddr())
		if len(m.Answer) != 0 {
			if err := w.WriteMsg(m); err != nil {
				logrus.Error(err)
			}
			return
		}
	}

	m, _, err := c.Exchange(r, "8.8.8.8:53")
	if err != nil {
		logrus.Error(err)
	}
	if err = w.WriteMsg(m); err != nil {
		logrus.Error(err)
	}
}

// func (a *Agent) handleX(w dns.ResponseWriter, r *dns.Msg) {
// 	m := new(dns.Msg)
// 	m.SetReply(r)
// 	m.Compress = false
// 	if r.Opcode == dns.OpcodeQuery {
// 		for _, q := range m.Question {
// 			if q.Qtype == dns.TypeA {
// 				log.Printf("Query for %s\n", q.Name)
// 				rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, "127.0.0.1"))
// 				if err == nil {
// 					m.Answer = append(m.Answer, rr)
// 				}
// 			}
// 		}
// 	}
// 	if err := w.WriteMsg(m); err != nil {
// 		logrus.Error(err)
// 	}
// }
