package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type soa struct {
	zone host
	self host
	soa  dns.RR
	mux  sync.RWMutex
}

func newSoa(zone, self host) *soa {
	s := &soa{
		zone: zone,
		self: self,
	}
	s.update()
	return s
}

func (s *soa) update() {
	s.mux.Lock()
	defer s.mux.Unlock()

	tstamp := uint32(time.Now().Unix())
	refresh := 86400
	retry := refresh
	expire := refresh
	minttl := 100
	soa := fmt.Sprintf("%s 1000 SOA %s %s %d %d %d %d %d", s.zone.dns(), s.self.dns(), s.self.dns(),
		tstamp, refresh, retry, expire, minttl)
	var err error
	if s.soa, err = dns.NewRR(soa); err != nil {
		log.Fatalf("invalid SOA record: %s", err)
	}
}

func (s *soa) write(m *dns.Msg) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	m.Ns = make([]dns.RR, 1)
	m.Ns[0] = s.soa
}

func (s *server) newDnsRR(name host) dns.RR {
	return &dns.NS{
		Hdr: dns.RR_Header{
			Name:   name.dns(),
			Rrtype: dns.TypeNS,
			Class:  dns.ClassINET,
		},
		Ns: s.self.dns(),
	}
}

func (s *server) handleDnsA(name host, m *dns.Msg) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	// Important: all things set here must be overwritten
	rec := s.repo.get(name)
	if rec != nil {
		m.Answer = make([]dns.RR, 1)
		m.Answer[0] = rec.a
		m.Ns = nil
		m.MsgHdr.Rcode = dns.RcodeSuccess
	} else {
		m.Answer = nil
		m.MsgHdr.Rcode = dns.RcodeNameError
		s.soa.write(m)
	}
}

func (s *server) handleDnsCNAME(name host, m *dns.Msg) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	rec := s.repo.get(name)
	if rec != nil && rec.cname != nil {
		m.Answer = append(m.Answer, rec.cname)
		m.Ns = nil
		m.MsgHdr.Rcode = dns.RcodeSuccess
	} else {
		m.Answer = nil
		m.MsgHdr.Rcode = dns.RcodeNameError
		s.soa.write(m)
	}
}

func (s *server) handleDnsNS(name host) *dns.Msg {
	m := new(dns.Msg)
	m.Answer = append(m.Answer, s.newDnsRR(name))
	s.soa.write(m)
	return m
}

func (server) logDns(w dns.ResponseWriter, level, format string, params ...interface{}) {
	log.Printf("[%s] dns: %s(%s): %s", level, w.RemoteAddr().Network(), w.RemoteAddr().String(), fmt.Sprintf(format, params...))
}

func (s *server) writeDnsMsg(w dns.ResponseWriter, m *dns.Msg) {
	if err := w.WriteMsg(m); err != nil {
		s.logDns(w, "error", "error writing DNS response packet: %s", err)
	}
}

func (s *server) handleQuery(w dns.ResponseWriter, r *dns.Msg) {
	switch r.Question[0].Qtype {
	case dns.TypeANY, dns.TypeA, dns.TypeAAAA:
		if s.verbose {
			s.logDns(w, "info", "request for A/ANY %s", r.Question[0].Name)
		}

		m := s.respPool.Get().(*dns.Msg)

		m.SetReply(r)
		s.handleDnsA(host(r.Question[0].Name), m)
		s.writeDnsMsg(w, m)

		s.respPool.Put(m)
	case dns.TypeCNAME:
		if s.verbose {
			s.logDns(w, "info", "request for CNAME %s", r.Question[0].Name)
		}

		m := s.respPool.Get().(*dns.Msg)

		m.SetReply(r)
		s.handleDnsCNAME(host(r.Question[0].Name), m)
		s.writeDnsMsg(w, m)

		s.respPool.Put(m)
	case dns.TypeNS:
		if s.verbose {
			s.logDns(w, "info", "request for NS %s", r.Question[0].Name)
		}

		m := s.handleDnsNS(host(r.Question[0].Name))
		m.SetReply(m)
		s.writeDnsMsg(w, m)
	default:
		s.logDns(w, "error", "unhandled request: %s", r.Question[0].Qtype)
	}
}

func (s *server) update() {
	s.soa.update()
}

func (s *server) serveNetDNS(addr, net string, errCh chan<- error) {
	serverTCP := &dns.Server{Addr: addr, Net: net, TsigSecret: nil}
	log.Printf("[info] dns: listening on %s (%s)", addr, net)
	errCh <- serverTCP.ListenAndServe()
}

func (s *server) serveDNS(addr string) {
	errCh := make(chan error)

	s.respPool.New = func() interface{} {
		return new(dns.Msg)
	}

	dns.HandleFunc(s.zone.dns(), s.handleQuery)

	go s.serveNetDNS(addr, "udp", errCh)
	go s.serveNetDNS(addr, "tcp", errCh)

	if err := <-errCh; err != nil {
		log.Fatal("dns: cannot start DNS server: ", err)
	}
}
