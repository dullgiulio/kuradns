// Copyright 2016 Giulio Iotti. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kuradns

import (
	"fmt"
	"io"
	"log"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/miekg/dns"

	"github.com/dullgiulio/kuradns/gen"
)

// A record containse the source and destination from the generator,
// a precomputed A and CNAME record and a pointer to the source that
// generated this entry.
type record struct {
	shost, dhost host
	a            dns.RR
	aaaa         dns.RR
	cname        dns.RR
	source       *source
}

// Allocate a new record. For the A record, a resoled IP is needed.
func makeRecord(shost, dhost host, cname bool, ip4, ip6 net.IP, ttl time.Duration, src *source) record {
	r := record{
		shost:  shost,
		dhost:  dhost,
		source: src,
	}
	if ip4 != nil {
		r.a = &dns.A{
			Hdr: dns.RR_Header{
				Name:   shost.dns(),
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    uint32(ttl.Seconds()),
			},
			A: ip4,
		}
	}
	if ip6 != nil {
		r.aaaa = &dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   shost.dns(),
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    uint32(ttl.Seconds()),
			},
			AAAA: ip6,
		}
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

// target returns the human DNS representation of the destination/target or a record.
func (r record) target() string {
	return r.dhost.browser()
}

// String representation of a record.
func (r record) String() string {
	return fmt.Sprintf("[%s %s]", r.source.String(), r.shost.dns())
}

// Collection of records.
type records struct {
	recs []record
}

// Allocate a new collection of records.
func newRecords() *records {
	return &records{recs: make([]record, 0)}
}

// String representation of a collection of records.
func (r *records) String() string {
	return fmt.Sprintf("%s", r.recs)
}

// deleteSource removes all record that were added by source s.
// Returns the number of records left.
func (r *records) deleteSource(s *source) int {
	res := make([]record, 0)
	for _, rec := range r.recs {
		if rec.source.name != s.name {
			res = append(res, rec)
		}
	}
	// Free up empty arrays
	if len(res) == 0 {
		res = nil
	}
	r.recs = res
	return len(r.recs)
}

// pushFront adds a record to the collection.
func (r *records) pushFront(rec record) {
	r.recs = append([]record{rec}, r.recs...)
}

// clone is the utility function to duplicate a collection.
func (r *records) clone() *records {
	nr := &records{
		recs: make([]record, len(r.recs)),
	}
	for i := 0; i < len(r.recs); i++ {
		nr.recs[i] = r.recs[i]
	}
	return nr
}

// A repository maps hosts to the records that can resolve them (record collection).
// repository is not thread safe.
type repository map[string]*records

// makeRepository allocates a new repository.
func makeRepository() repository {
	return make(map[string]*records)
}

// add inserts record rec for host as the default record.
func (r repository) add(host host, rec record) {
	key := host.browser()
	recs, ok := r[key]
	if !ok {
		recs = newRecords()
	}
	recs.pushFront(rec)
	r[key] = recs
}

// deleteSource removes all records that were inserted by source s.
// Entries that result in no records are removed completely.
func (r repository) deleteSource(s *source) {
	emptyKeys := make([]string, 0)
	for k, recs := range r {
		if recs.deleteSource(s) == 0 {
			emptyKeys = append(emptyKeys, k)
		}
	}
	// remove entries which have no values
	for _, k := range emptyKeys {
		delete(r, k)
	}
}

// updateSource removes and generate again all records for source src.
func (r repository) updateSource(src *source, zone host, ttl time.Duration) {
	res := newResolver(src, ttl, 6)

	go func() {
		for {
			rentry := src.gen.Generate()
			if rentry.IsEmpty() {
				close(res.rentries)
				// Free up resources used by the generator
				src.gen = nil
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

// resolver is a worker that resolves strings into IPs.
type resolver struct {
	cname    bool
	src      *source
	ttl      time.Duration
	rentries chan gen.RawEntry
	records  chan record
	wg       sync.WaitGroup
}

// Allocate a new resolver. Subsequent records will be generated with ttl set as given here.
// workers is number of workers to be run in parallel.
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

// run resolves incoming entries and emits records. It is called automatically.
func (r *resolver) run() {
	for rentry := range r.rentries {
		var cname bool
		var ip4, ip6 net.IP
		ip := net.ParseIP(rentry.Target)
		if ip != nil {
			if ip.To4() == nil {
				ip6 = ip
			} else {
				ip4 = ip
			}
		} else {
			var err error
			// It's an hostname: resolve it and make both A and CNAME records
			ip4, ip6, err = lookup(rentry.Target)
			if err != nil {
				log.Printf("[error] repository: failed lookup of %s: %s", rentry.Target, err)
				continue
			}
			cname = true
		}
		r.records <- makeRecord(host(rentry.Source), host(rentry.Target), cname, ip4, ip6, r.ttl, r.src)
	}
	r.wg.Done()
}

// lookup is a utility to lookup an IP for host (standard format).
func lookup(host string) (ip4, ip6 net.IP, err error) {
	var iplist []net.IP
	iplist, err = net.LookupIP(host)
	if err != nil {
		return
	}
	for _, ip := range iplist {
		if len(ip) == net.IPv4len {
			if ip4 == nil {
				ip4 = ip
			}
			continue
		}
		if ip6 == nil {
			ip6 = ip
		}
	}
	return
}

// get returns the default record for host hs or nil if not found.
// host will also be matched against all wildcards; first matching wildcard entry is returned.
func (r repository) get(hs host) *record {
	rs, ok := r[hs.browser()]
	if ok {
		return &rs.recs[0]
	}
	for k := range r {
		khost := host(k)
		if !khost.hasWildcard() {
			continue
		}
		if khost.match(hs) {
			rs = r[khost.browser()]
			return &rs.recs[0]
		}
	}
	return nil
}

// clone duplicates the whole repository.
func (r repository) clone() repository {
	nr := makeRepository()
	for k, recs := range r {
		nr[k] = recs.clone()
	}
	return nr
}

// WriteTo writes the repository contents in hosts format to w.
func (r repository) WriteTo(w io.Writer) error {
	frs := makeFlatRecords()
	for key, rs := range r {
		if len(rs.recs) == 0 {
			continue
		}
		grpKey := rs.recs[0].target()
		for i := range rs.recs {
			frs = append(frs, newFlatRecord(i, grpKey, rs.recs[i].target(), key))
		}
	}
	sort.Sort(frs)
	for i := range frs {
		fmtstr := "%s\t%s\n"
		if frs[i].pos > 0 {
			fmtstr = "# %s\t%s\n"
		}
		if _, err := fmt.Fprintf(w, fmtstr, frs[i].dst, frs[i].src); err != nil {
			return err
		}
	}
	return nil
}

// flatRecords are convenience data structure to simplify sorting of
// all record entries.
//
// Each entry has a source and destination. As there can be multiple
// shadowed entries for each source/destination pair, these are sorted
// by pos and grouped together by having the same grp.
type flatRecord struct {
	pos int
	grp string
	dst string
	src string
}

// flatRecords is a sortable slice of flatRecord elements.
type flatRecords []*flatRecord

// makeFlatRecords allocates an empty slice of flatRecords
func makeFlatRecords() flatRecords {
	return flatRecords(make([]*flatRecord, 0))
}

// newFlatRecord allocates a flatRecord
func newFlatRecord(i int, grp, dst, src string) *flatRecord {
	return &flatRecord{i, grp, dst, src}
}

// Len to implement Sort interface
func (f flatRecords) Len() int {
	return len(f)
}

// Swap to implement Sort interface
func (f flatRecords) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

// Less orders by destination dst, position pos and source src, grouping by group grp/
func (f flatRecords) Less(i, j int) bool {
	if f[i].dst < f[j].dst {
		return true
	}
	if f[i].dst > f[j].dst {
		return false
	}
	if f[i].grp < f[j].grp {
		return true
	}
	if f[i].grp > f[j].grp {
		return false
	}
	if f[i].pos < f[j].pos {
		return true
	}
	if f[i].pos > f[j].pos {
		return false
	}
	return f[i].src < f[j].src

}
