package agent

import (
	"net"
	"testing"

	"github.com/maxmcd/tester"
	"github.com/miekg/dns"
)

func TestDnsSibling(te *testing.T) {
	t := tester.New(te)
	tp := NewTestAgent()
	testData := tp.createTestData(map[string][]mockTask{
		"standalone2": {{
			name: "counter",
			ports: []mockTaskPort{{
				label: "9001",
			}},
		}, {
			name: "dashboard",
			ports: []mockTaskPort{{
				label: "9002",
			}},
		}},
	})

	tp.loadAllTestData(&t, testData)

	counterIP := testData["standalone2"].containers["counter"].NetworkSettings.IPAddress
	dashboardIP := testData["standalone2"].containers["dashboard"].NetworkSettings.IPAddress
	addr := net.UDPAddr{
		IP:   net.ParseIP(counterIP),
		Port: 0,
	}
	m1 := new(dns.Msg)
	m1.Id = dns.Id()
	m1.RecursionDesired = true
	m1.Question = make([]dns.Question, 1)
	m1.Question[0] = dns.Question{
		Name:   "dashboard.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	}
	t.Assert().True(tp.findDNSResponse(m1, &addr))
	t.Assert().Equal(m1.Answer[0].(*dns.A).A.String(), dashboardIP)
}

func TestDnsService(te *testing.T) {
	t := tester.New(te)
	tp := NewTestAgent()
	testData := tp.createTestData(map[string][]mockTask{
		"counter": {{
			name: "counter",
			ports: []mockTaskPort{{
				label: "9001",
			}},
		}},
		"dashboard": {{
			name: "dashboard",
			ports: []mockTaskPort{{
				label: "9002",
			}},
			connectTo: []string{"counter.counter:9001"},
		}},
	})

	tp.loadAllTestData(&t, testData)

	dashboardIP := testData["dashboard"].containers["dashboard"].NetworkSettings.IPAddress
	addr := net.UDPAddr{
		IP:   net.ParseIP(dashboardIP),
		Port: 0,
	}
	m1 := new(dns.Msg)
	m1.Id = dns.Id()
	m1.RecursionDesired = true
	m1.Question = make([]dns.Question, 1)
	m1.Question[0] = dns.Question{
		Name:   "counter.counter.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	}
	t.Assert().True(tp.findDNSResponse(m1, &addr))
	t.Assert().Equal(m1.Answer[0].(*dns.A).A.String(), tp.ip.String())
}
