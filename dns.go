package main

import (
	"log"

	"github.com/miekg/dns"
)

func (s *server) handleDnsA(name host, m *dns.Msg) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	rec := s.repo.get(name)
	if rec != nil {
		m.Answer = append(m.Answer, &rec.arec)
	} else {
		m.MsgHdr.Rcode = dns.RcodeNameError
	}
}

func (s *server) handleDnsNS(name host, m *dns.Msg) {
	// TODO: This is easily cacheable.
	rr := &dns.NS{
		Hdr: dns.RR_Header{
			Name:   name.dns(),
			Rrtype: dns.TypeNS,
			Class:  dns.ClassINET,
			Ttl:    6400, // TODO: Make configurable
		},
		Ns: s.self.dns(),
	}
	m.Answer = append(m.Answer, rr)
}

func (s *server) handleQuery(w dns.ResponseWriter, r *dns.Msg) {
	if s.verbose {
		log.Printf("[info] dns: request for %s", r)
	}
	// TODO: Keep a sync.Pool of Msg, both answers and NXDOMAIN.
	m := new(dns.Msg)
	m.SetReply(r)
	name := host(r.Question[0].Name)

	switch r.Question[0].Qtype {
	case dns.TypeNS:
		s.handleDnsNS(name, m)
	case dns.TypeANY, dns.TypeA, dns.TypeAAAA:
		s.handleDnsA(name, m)
	}
	if err := w.WriteMsg(m); err != nil {
		log.Printf("dns: %s: error writing DNS response packet: %s", w.RemoteAddr(), err)
	}
}

func (s *server) serveNetDNS(addr, net string, errCh chan<- error) {
	serverTCP := &dns.Server{Addr: addr, Net: net, TsigSecret: nil}
	errCh <- serverTCP.ListenAndServe()
}

func (s *server) serveDNS(addr string, zone, self host) {
	errCh := make(chan error)

	s.zone = zone
	s.self = self

	dns.HandleFunc(s.zone.dns(), s.handleQuery)

	go s.serveNetDNS(addr, "udp", errCh)
	go s.serveNetDNS(addr, "tcp", errCh)

	if err := <-errCh; err != nil {
		log.Fatal("dns: cannot start DNS server: ", err)
	}
}
