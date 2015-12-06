package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"

	"github.com/dullgiulio/kuradns/gen"
)

type record struct {
	shost, dhost host
	a            dns.RR
	cname        dns.RR
	source       *source
}

func makeRecord(shost, dhost host, cname bool, ip net.IP, ttl time.Duration, src *source) record {
	r := record{
		shost: shost,
		dhost: dhost,
		a: &dns.A{
			Hdr: dns.RR_Header{
				Name:   shost.dns(),
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    uint32(ttl.Seconds()),
			},
			A: ip,
		},
		source: src,
	}
	if cname {
		r.cname = &dns.CNAME{
			Hdr: dns.RR_Header{
				Name:   shost.dns(),
				Rrtype: dns.TypeCNAME,
				Class:  dns.ClassINET,
				Ttl:    uint32(ttl.Seconds()),
			},
			Target: host(dhost).dns(),
		}
	}
	return r
}

func (r record) target() string {
	return r.dhost.browser()
}

func (r record) String() string {
	return fmt.Sprintf("[%s %s]", r.source.String(), r.shost.dns())
}

type records struct {
	recs []record
}

func newRecords() *records {
	return &records{recs: make([]record, 0)}
}

func (r *records) String() string {
	return fmt.Sprintf("%s", r.recs)
}

func (r *records) deleteSource(s *source) {
	res := r.recs[:0]
	for _, rec := range r.recs {
		if rec.source.name != s.name {
			res = append(res, rec)
		}
	}
	r.recs = res
}

func (r *records) pushFront(rec record) {
	r.recs = append([]record{rec}, r.recs...)
}

func (r *records) clone() *records {
	nr := &records{
		recs: make([]record, len(r.recs)),
	}
	for i := 0; i < len(r.recs); i++ {
		nr.recs[i] = r.recs[i]
	}
	return nr
}

type host string

func (h host) browser() string {
	return strings.TrimSuffix(string(h), ".")
}

func (h host) dns() string {
	return dns.Fqdn(string(h))
}

func (h host) hasSuffix(h2 host) bool {
	return strings.HasSuffix(h.browser(), h2.browser())
}

type repository map[string]*records

func makeRepository() repository {
	return make(map[string]*records)
}

func (r repository) add(host host, rec record) {
	key := host.browser()
	recs, ok := r[key]
	if !ok {
		recs = newRecords()
	}
	recs.pushFront(rec)
	r[key] = recs
}

func (r repository) deleteSource(s *source) {
	for _, recs := range r {
		recs.deleteSource(s)
	}
}

func (r repository) updateSource(src *source, zone host, ttl time.Duration) {
	res := newResolver(src, ttl, 6)

	go func() {
		for {
			rentry := src.gen.Generate()
			if rentry.IsEmpty() {
				close(res.rentries)
				return
			}
			src := host(rentry.Source)
			if !src.hasSuffix(zone) {
				log.Printf("[error] repository: domain %s is not inside zone %s, skipped", src.dns(), zone.dns())
				continue
			}
			res.rentries <- rentry
		}
	}()

	for rec := range res.records {
		r.add(rec.shost, rec)
	}
}

type resolver struct {
	cname    bool
	src      *source
	ttl      time.Duration
	rentries chan gen.RawEntry
	records  chan record
	wg       sync.WaitGroup
}

func newResolver(src *source, ttl time.Duration, workers int) *resolver {
	r := &resolver{
		src:      src,
		ttl:      ttl,
		rentries: make(chan gen.RawEntry),
		records:  make(chan record),
	}
	r.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go r.run()
	}
	go func() {
		r.wg.Wait()
		close(r.records)
	}()
	return r
}

func (r *resolver) run() {
	for rentry := range r.rentries {
		var cname bool
		ip := net.ParseIP(rentry.Target)
		if ip == nil {
			var err error
			// It's an hostname: resolve it and make both A and CNAME records
			ip, err = lookup(rentry.Target)
			if err != nil {
				log.Printf("[error] repository: failed lookup of %s: %s", rentry.Target, err)
				continue
			}
			cname = true
		}
		r.records <- makeRecord(host(rentry.Source), host(rentry.Target), cname, ip, r.ttl, r.src)
	}
	r.wg.Done()
}

func lookup(host string) (net.IP, error) {
	var ip net.IP
	iplist, err := net.LookupIP(host)
	if err == nil {
		ip = iplist[0]
	}
	return ip, err
}

func (r repository) get(host host) *record {
	rs, ok := r[host.browser()]
	if !ok {
		return nil
	}
	return &rs.recs[0]
}

func (r repository) clone() repository {
	nr := makeRepository()
	for k, recs := range r {
		nr[k] = recs.clone()
	}
	return nr
}

func (r repository) WriteTo(w io.Writer) error {
	for key, rs := range r {
		if len(rs.recs) == 0 {
			continue
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\n", rs.recs[0].target(), key); err != nil {
			return err
		}
		for i := 1; i < len(rs.recs); i++ {
			if _, err := fmt.Fprintf(w, "# %s\t%s\n", rs.recs[i].target(), key); err != nil {
				return err
			}
		}
	}
	return nil
}
