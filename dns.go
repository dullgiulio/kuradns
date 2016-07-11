// Copyright 2016 Giulio Iotti. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kuradns

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// Represent a SOA record shared system-wide.
type soa struct {
	zone host
	self host
	soa  dns.RR
	mux  sync.RWMutex
}

// newSoa allocates a SOA container.
func newSoa(zone, self host) *soa {
	s := &soa{
		zone: zone,
		self: self,
	}
	s.update()
	return s
}

// update changes the SOA record to have a new serial number to reflect changes to the repository.
func (s *soa) update() {
	s.mux.Lock()
	defer s.mux.Unlock()

	tstamp := uint32(time.Now().Unix())
	refresh := 3600
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

// write adds the SOA record to message m in the namespaces section.
func (s *soa) write(m *dns.Msg) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	m.Ns = make([]dns.RR, 1)
	m.Ns[0] = s.soa
}

// newDnsRR allocates a resource record of type NS for name.
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

// handleDnsA modifies m to reply to a A/ANY query by looking up name from the
// repository. NXDOMAIN and the SOA record are returned for nonexisting entries.
func (s *server) handleDnsA(name host, m *dns.Msg) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	// Important: all things set here must be overwritten
	rec := s.repo.get(name)
	if rec != nil && rec.a != nil {
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

// handleDnsAAAA modifies m to reply to a AAAA query by looking up name from the
// repository. NXDOMAIN and the SOA record are returned for nonexisting entries.
func (s *server) handleDnsAAAA(name host, m *dns.Msg) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	// Important: all things set here must be overwritten
	rec := s.repo.get(name)
	if rec != nil && rec.aaaa != nil {
		m.Answer = make([]dns.RR, 1)
		m.Answer[0] = rec.aaaa
		m.Ns = nil
		m.MsgHdr.Rcode = dns.RcodeSuccess
	} else {
		m.Answer = nil
		m.MsgHdr.Rcode = dns.RcodeNameError
		s.soa.write(m)
	}
}

// handleDnsCNAME modifies m to respond to a CNAME query for name by looking it up
// from the repository. NXDOMAIN and the SOA record are returned for nonexisting entries.
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

// handleDnsNS returns a DNS message in response to a NS query using name as the namespace hostname.
func (s *server) handleDnsNS(name host) *dns.Msg {
	m := new(dns.Msg)
	m.Answer = append(m.Answer, s.newDnsRR(name))
	s.soa.write(m)
	return m
}

// handleDnsMX writes a MX record pointing to s.host into m for host name.
func (s *server) handleDnsMX(name host, m *dns.Msg) {
	r := new(dns.MX)
	r.Hdr = dns.RR_Header{
		Name:   name.dns(),
		Rrtype: dns.TypeMX,
		Class:  dns.ClassINET,
		Ttl:    3600,
	}
	r.Preference = 10
	r.Mx = s.self.dns()
	m.Answer = make([]dns.RR, 1)
	m.Answer[0] = r
}

// logDns is an utility to write a log message as coming from the DNS subsystem.
func (server) logDns(w dns.ResponseWriter, level, format string, params ...interface{}) {
	log.Printf("[%s] dns: %s(%s): %s", level, w.RemoteAddr().Network(), w.RemoteAddr().String(), fmt.Sprintf(format, params...))
}

// writeDnsMsg is a utility to write a DNS message and log the possible error.
func (s *server) writeDnsMsg(w dns.ResponseWriter, m *dns.Msg) {
	if err := w.WriteMsg(m); err != nil {
		s.logDns(w, "error", "error writing DNS response packet: %s", err)
	}
}

// handleQuery handles a single DNS query r writing a DNS response message to w.
//
// Currently CNAME, ANY/A/AAAA and NS are supported queries. Other queries will be logged but
// not responded to.
func (s *server) handleQuery(w dns.ResponseWriter, r *dns.Msg) {
	switch r.Question[0].Qtype {
	case dns.TypeANY, dns.TypeA, dns.TypeAAAA:
		if s.verbose {
			s.logDns(w, "info", "request for %s %s", dns.TypeToString[r.Question[0].Qtype], r.Question[0].Name)
		}

		m := s.respPool.Get().(*dns.Msg)

		m.SetReply(r)
		if r.Question[0].Qtype == dns.TypeAAAA {
			s.handleDnsAAAA(host(r.Question[0].Name), m)
		} else {
			s.handleDnsA(host(r.Question[0].Name), m)
		}
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
		m.SetReply(r)
		s.writeDnsMsg(w, m)
	case dns.TypeMX:
		if s.verbose {
			s.logDns(w, "info", "request for MX %s", r.Question[0].Name)
		}

		m := new(dns.Msg)
		s.handleDnsMX(host(r.Question[0].Name), m)
		m.SetReply(r)
		s.writeDnsMsg(w, m)
	default:
		s.logDns(w, "error", "unhandled request: %s", r.Question[0].Qtype)
	}
}

// update performs all operations needed after the repository have been modified.
func (s *server) update() {
	s.soa.update()
}

// serveNetDNS starts a DNS listener on addr:net, writes the first error
// encountered on errCh. When there are no errors, this function doesn't return.
func (s *server) serveNetDNS(addr, net string, errCh chan<- error) {
	serverTCP := &dns.Server{Addr: addr, Net: net, TsigSecret: nil}
	log.Printf("[info] dns: listening on %s (%s)", addr, net)
	errCh <- serverTCP.ListenAndServe()
}

// serveDNS sets up responders to DNS queries on both TCP and UDP. It
// logs the first error encountered and exists the program.
func (s *server) ServeDNS(addr string) {
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
