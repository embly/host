package main

import (
	"fmt"
	"log"

	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

var c *dns.Client

func main() {
	c = new(dns.Client)

	server := &dns.Server{Addr: "127.0.0.1:5353", Net: "udp"}
	dns.HandleFunc("x.", handleX)
	dns.HandleFunc(".", forward)
	fmt.Println("serving", server)
	log.Fatal(server.ListenAndServe())
}
func forward(w dns.ResponseWriter, r *dns.Msg) {
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
