package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/miekg/dns"

	"github.com/dullgiulio/kuradns/gen"
)

type record struct {
	host   host
	arec   dns.A
	source *source
}

func makeRecord(shost, dhost string, ip net.IP, src *source) record {
	return record{
		host: host(dhost),
		arec: dns.A{
			Hdr: dns.RR_Header{
				Name:   shost,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    6400, // TODO: Make configurable
			},
			A: ip,
		},
		source: src,
	}
}

func (r record) String() string {
	return fmt.Sprintf("[%s/%s: %s]", r.source.String(), r.host.dns(), &r.arec)
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

type repository map[host]*records

func makeRepository() repository {
	return make(map[host]*records)
}

func (r repository) add(host host, rec record) {
	recs, ok := r[host]
	if !ok {
		recs = newRecords()
	}
	recs.pushFront(rec)
	r[host] = recs
}

func (r repository) deleteSource(s *source) {
	for _, recs := range r {
		recs.deleteSource(s)
	}
}

func (r repository) updateSource(src *source) {
	res := newResolver(src, 6)

	go func() {
		for {
			rentry := src.gen.Generate()
			if rentry.IsEmpty() {
				close(res.rentries)
				return
			}
			// TODO: Check that the source is inside the zone
			res.rentries <- rentry
		}
	}()

	for rec := range res.records {
		r.add(host(rec.arec.Hdr.Name), rec)
	}
}

type resolver struct {
	src      *source
	rentries chan gen.RawEntry
	records  chan record
	wg       sync.WaitGroup
}

func newResolver(src *source, workers int) *resolver {
	r := &resolver{
		src:      src,
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
		ip, err := lookup(rentry.Target)
		if err != nil {
			log.Printf("failed lookup of %s: %s", rentry.Target, err)
			continue
		}
		r.records <- makeRecord(rentry.Source, rentry.Target, ip, r.src)
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

func (r repository) get(key host) *record {
	rs, ok := r[key]
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
		if _, err := fmt.Fprintf(w, "%s\t%s\t#-> %s\n", rs.recs[0].arec.A, key.browser(), rs.recs[0].host.browser()); err != nil {
			return err
		}
		for i := 1; i < len(rs.recs); i++ {
			if _, err := fmt.Fprintf(w, "# %s\t%s\t#-> %s\n", rs.recs[i].arec.A, key.browser(), rs.recs[i].host.browser()); err != nil {
				return err
			}
		}
	}
	return nil
}
