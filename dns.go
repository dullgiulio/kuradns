package main

import (
	"log"

	"github.com/miekg/dns"
)

func (s *server) newDnsRR(name host) dns.RR {
	return &dns.NS{
		Hdr: dns.RR_Header{
			Name:   name.dns(),
			Rrtype: dns.TypeNS,
			Class:  dns.ClassINET,
			Ttl:    uint32(s.ttl.Seconds()),
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
		m.Answer = append(m.Answer, rec.a)
		m.MsgHdr.Rcode = dns.RcodeSuccess
	} else {
		m.Answer = nil
		m.MsgHdr.Rcode = dns.RcodeNameError
	}
}

func (s *server) handleDnsCNAME(name host, m *dns.Msg) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	rec := s.repo.get(name)
	if rec != nil && rec.cname != nil {
		m.Answer = append(m.Answer, rec.cname)
		m.MsgHdr.Rcode = dns.RcodeSuccess
	} else {
		m.Answer = nil
		m.MsgHdr.Rcode = dns.RcodeNameError
	}
}

func (s *server) handleDnsNS(name host) *dns.Msg {
	m := new(dns.Msg)
	m.Answer = append(m.Answer, s.newDnsRR(name))
	return m
}

func (s *server) writeDnsMsg(w dns.ResponseWriter, m *dns.Msg) {
	if err := w.WriteMsg(m); err != nil {
		log.Printf("[error] dns: %s: error writing DNS response packet: %s", w.RemoteAddr(), err)
	}
}

func (s *server) handleQuery(w dns.ResponseWriter, r *dns.Msg) {
	switch r.Question[0].Qtype {
	case dns.TypeANY, dns.TypeA, dns.TypeAAAA:
		if s.verbose {
			log.Printf("[info] dns: request for A/ANY %s", r.Question[0].Name)
		}

		m := s.respPool.Get().(*dns.Msg)

		m.SetReply(r)
		s.handleDnsA(host(r.Question[0].Name), m)
		s.writeDnsMsg(w, m)

		s.respPool.Put(m)
	case dns.TypeCNAME:
		if s.verbose {
			log.Printf("[info] dns: request for CNAME %s", r.Question[0].Name)
		}

		m := s.respPool.Get().(*dns.Msg)

		m.SetReply(r)
		s.handleDnsCNAME(host(r.Question[0].Name), m)
		s.writeDnsMsg(w, m)

		s.respPool.Put(m)
	case dns.TypeNS:
		if s.verbose {
			log.Print("[info] dns: request for NS %s", r.Question[0].Name)
		}

		m := s.handleDnsNS(host(r.Question[0].Name))
		m.SetReply(m)
		s.writeDnsMsg(w, m)
	default:
		log.Printf("[error] dns: unhandled request: %s", r.Question[0].Qtype)
	}
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
