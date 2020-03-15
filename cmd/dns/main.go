package main

import (
	"fmt"
	"log"

	"github.com/miekg/dns"
)

var c *dns.Client

func main() {
	c = new(dns.Client)

	server := &dns.Server{Addr: ":5353", Net: "udp"}
	dns.HandleFunc(".", handleRequest)
	fmt.Println("serving", server)
	log.Fatal(server.ListenAndServe())

}
func handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	in, rtt, err := c.Exchange(r, "8.8.8.8:53")
	_ = rtt
	_ = err
	if in != nil {
		fmt.Println(in.Answer[0])
		fmt.Printf("%v", in.Answer[0])
		w.WriteMsg(in)
	}
	fmt.Println(r.Question[0].Name)
	// tester.Print(r, r.Question)
}
