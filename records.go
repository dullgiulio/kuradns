package main

import (
	"fmt"
	"net"

	"github.com/miekg/dns"
)

type source struct {
	name string
}

func newSource(name string) *source {
	return &source{name}
}

func (s source) String() string {
	return s.name
}

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
				Name:   fmt.Sprintf("%s.", shost), // TODO: Make more robust
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

func makeRecords() *records {
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
		recs = makeRecords()
	}
	recs.pushFront(rec)
	r[host] = recs
}

func (r repository) deleteSource(s *source) {
	for _, recs := range r {
		recs.deleteSource(s)
	}
}

func (r repository) clone() repository {
	nr := makeRepository()
	for k, recs := range r {
		nr[k] = recs.clone()
	}
	return nr
}
