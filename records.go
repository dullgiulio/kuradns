package main

import (
	"fmt"
	"log"
	"net"

	"github.com/miekg/dns"
)

type record struct {
	host   string
	arec   dns.A
	source *source
}

func makeRecord(shost, dhost string, ip net.IP, src *source) record {
	return record{
		host: dhost,
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
	return fmt.Sprintf("[%s/%s: %s]", r.source.String(), r.host, &r.arec)
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
		if rec.source != s {
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

type repository map[string]*records

func makeRepository() repository {
	return make(map[string]*records)
}

func (r repository) add(host string, rec record) {
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

func (r repository) updateSource(src *source, debug bool) {
	cache := makeIPCache()

	for {
		rentry := src.gen.Generate()
		if rentry.IsEmpty() {
			break
		}

		// TODO: Lookups are slow: make it so that N can be fired at the same time
		res, err := cache.lookup(rentry.Target)
		if err != nil {
			log.Print("failed to lookup %s: %s\n", rentry.Source, err)
			continue
		}
		r.add(rentry.Source, makeRecord(rentry.Source, rentry.Target, res, src))
		if debug {
			log.Printf("ADD: %s -> %s [%s]\n", rentry.Source, rentry.Target, res)
		}
	}
}

func (r repository) get(key string) *record {
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
