package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

type ipEntry struct {
	ip  net.IP
	err error
}

type ipCache map[string]ipEntry

func makeIPCache() ipCache {
	return make(map[string]ipEntry)
}

func (c ipCache) lookup(host string) (net.IP, error) {
	if res, ok := c[host]; ok {
		return res.ip, res.err
	}
	var ip net.IP
	iplist, err := net.LookupIP(host)
	if err == nil {
		ip = iplist[0]
	}
	c[host] = ipEntry{ip, err}
	return ip, err
}

type rawentry struct {
	source, target string
}

func makeRawentry(s, t string) rawentry {
	return rawentry{s, t}
}

func emptyRawentry() rawentry {
	return rawentry{}
}

func (r rawentry) isEmpty() bool {
	return r.source == "" || r.target == ""
}

type generator interface {
	// generate reuturns an entry between a hostname its destination IP/hostname.
	// When no more pairs are available, generate should return an empty rawentry
	generate() rawentry
}

// staticgen is a generator that yields constant entries. Used for testing.
type staticgen struct {
	ch chan rawentry
}

func newStaticgen() *staticgen {
	return &staticgen{
		ch: make(chan rawentry),
	}
}

func (s *staticgen) run() {
	entries := []rawentry{
		makeRawentry("localhost", "127.0.0.1"),
		makeRawentry("some.host.com", "localhost"),
		makeRawentry("invalid.host", "invalid.host"),
	}
	for _, entry := range entries {
		s.ch <- entry
	}
	close(s.ch)
}

func (s *staticgen) generate() rawentry {
	return <-s.ch
}

// dategen is a generator that yields entries containing the date. Used for testing.
type dategen struct {
	ch   chan rawentry
	date string
}

func newDategen() *dategen {
	return &dategen{
		ch:   make(chan rawentry),
		date: time.Now().UTC().Format("20060102150405"),
	}
}

func (d *dategen) run() {
	d.ch <- makeRawentry(fmt.Sprintf("%s.%s", d.date, "mydomain.test"), "127.0.0.1")
	close(d.ch)
}

func (d *dategen) generate() rawentry {
	return <-d.ch
}

func main() {
	var gen generator

	repo := makeRepository()
	cache := makeIPCache()

	// sgen := newStaticgen()
	// go sgen.run()
	// gen = sgen
	dgen := newDategen()
	go dgen.run()
	gen = dgen

	src := newSource("static")

	for {
		rentry := gen.generate()
		if rentry.isEmpty() {
			break
		}

		// TODO: Lookups are slow: make it so that N can be fired at the same time
		res, err := cache.lookup(rentry.target)
		if err != nil {
			fmt.Printf("%s: %s\n", rentry.source, err)
			continue
		}
		repo.add(rentry.source, makeRecord(rentry.source, rentry.target, res, src))
	}

	fmt.Printf("%s\n", repo.clone())
	os.Exit(1)
}

// Update source:
// 1. get RWLock(), repo.clone(), unlock;
// 2. repo.deleteSource(src)
// 3. loop with generator
// 4. get RWLock(), repo swap, unlock;

// All operations are queued (buffered chan and try-send

// A routine performs queued operations

// A many routines handle DNS queries with RLock()
