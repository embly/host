package agent

import (
	"fmt"
	"log"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

var c *dns.Client

func StartDNS() (err error) {
	c = new(dns.Client)
	addr := "172.17.0.1:53"
	// server := &dns.Server{Addr: "192.168.86.30:53", Net: "udp"}
	server := &dns.Server{Addr: addr, Net: "udp"}
	dns.HandleFunc("x.", handleX)
	dns.HandleFunc(".", forward)
	fmt.Println("serving dns", server)
	return server.ListenAndServe()
}
func forward(w dns.ResponseWriter, r *dns.Msg) {
	log.Println(w, r, w.RemoteAddr())
	m, _, err := c.Exchange(r, "8.8.8.8:53")
	if err != nil {
		logrus.Error(err)
	}
	if err = w.WriteMsg(m); err != nil {
		logrus.Error(err)
	}
}

func handleX(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false
	if r.Opcode == dns.OpcodeQuery {
		for _, q := range m.Question {
			if q.Qtype == dns.TypeA {
				log.Printf("Query for %s\n", q.Name)
				rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, "127.0.0.1"))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
		}
	}
	if err := w.WriteMsg(m); err != nil {
		logrus.Error(err)
	}
}
