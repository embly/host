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
	{
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
	{
		m1 := new(dns.Msg)
		m1.Id = dns.Id()
		m1.RecursionDesired = true
		m1.Question = make([]dns.Question, 1)
		m1.Question[0] = dns.Question{
			Name:   "google.com.",
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET,
		}
		t.Assert().False(tp.findDNSResponse(m1, &addr))
	}
}

func TestHandleDNS(te *testing.T) {
	t := tester.New(te)
	ta := NewTestAgent()
	testData := ta.createTestData(map[string][]mockTask{
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

	ta.loadAllTestData(&t, testData)

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
	w := &mockDNSResponseWriter{
		remoteAddr: &addr,
	}
	ta.handleDNS(w, m1)
	t.Assert().Equal(w.written[0].Answer[0].(*dns.A).A.String(), dashboardIP)
}

type mockDNSResponseWriter struct {
	written    []*dns.Msg
	localAddr  net.Addr
	remoteAddr net.Addr
}

func (m *mockDNSResponseWriter) WriteMsg(msg *dns.Msg) error {
	m.written = append(m.written, msg)
	return nil
}
func (m *mockDNSResponseWriter) LocalAddr() net.Addr       { return m.localAddr }
func (m *mockDNSResponseWriter) RemoteAddr() net.Addr      { return m.remoteAddr }
func (m *mockDNSResponseWriter) Write([]byte) (int, error) { return 0, nil }
func (m *mockDNSResponseWriter) Close() error              { return nil }
func (m *mockDNSResponseWriter) TsigStatus() error         { return nil }
func (m *mockDNSResponseWriter) TsigTimersOnly(bool)       {}
func (m *mockDNSResponseWriter) Hijack()                   {}
