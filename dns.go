package main

import (
	"log"

	"github.com/miekg/dns"
)

func (s *server) handleQuery(w dns.ResponseWriter, r *dns.Msg) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	// TODO: Keep a sync.Pool of Msg, both answers and NXDOMAIN.
	m := new(dns.Msg)
	m.SetReply(r)
	rec := s.repo.get(r.Question[0].Name)
	if rec == nil {
		m.MsgHdr.Rcode = dns.RcodeNameError
	} else {
		m.Answer = append(m.Answer, &rec.arec)
	}
	w.WriteMsg(m)
}

func (s *server) serveNetDNS(addr, net string, errCh chan<- error) {
	serverTCP := &dns.Server{Addr: addr, Net: net, TsigSecret: nil}
	errCh <- serverTCP.ListenAndServe()
}

func (s *server) serveDNS(addr, root string) {
	errCh := make(chan error)

	dns.HandleFunc(root, s.handleQuery)

	go s.serveNetDNS(addr, "udp", errCh)
	go s.serveNetDNS(addr, "tcp", errCh)

	if err := <-errCh; err != nil {
		log.Fatal("Cannot start DNS server: ", err)
	}
}
