package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

var errInvalidGenerator = errors.New("invalid generator name")

type ipentry struct {
	ip  net.IP
	err error
}

type ipcache map[string]ipentry

func makeIPCache() ipcache {
	return make(map[string]ipentry)
}

func (c ipcache) lookup(host string) (net.IP, error) {
	if res, ok := c[host]; ok {
		return res.ip, res.err
	}
	var ip net.IP
	iplist, err := net.LookupIP(host)
	if err == nil {
		ip = iplist[0]
	}
	c[host] = ipentry{ip, err}
	return ip, err
}

// TODO: should have a type like "A" or "CNAME" (already validated?)
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

func makeGenerator(name string, cfg config) (generator, error) {
	switch name { // strings.Lower
	case "date":
		g := newDategen()
		go g.run()
		return g, nil
	case "static":
		g := newStaticgen()
		go g.run()
		return g, nil
	default:
		return nil, errInvalidGenerator
	}
}

type sources struct {
	repo repository
	mux  sync.RWMutex
	// Sources added or refreshed (TODO: Add needs params)
	in chan *source
	// Sources to be removed
	rm chan string
}

func newSources() *sources {
	return &sources{
		repo: makeRepository(),
		in:   make(chan *source), // TODO: this will be buffered
		rm:   make(chan string),
	}
}

func (s *sources) updateSource(src *source, repo repository) error {
	cache := makeIPCache()

	for {
		rentry := src.gen.generate()
		if rentry.isEmpty() {
			break
		}

		// TODO: Lookups are slow: make it so that N can be fired at the same time
		res, err := cache.lookup(rentry.target)
		if err != nil {
			// TODO: where does this go? logging?
			fmt.Printf("failed to lookup %s: %s\n", rentry.source, err)
			continue
		}
		repo.add(rentry.source, makeRecord(rentry.source, rentry.target, res, src))
	}
	return nil
}

func (s *sources) run() {
	for {
		select {
		case src := <-s.in:
			// Clone the repo
			s.mux.RLock()
			repo := s.repo.clone()
			s.mux.RUnlock()

			// Update the cloned repo
			s.updateSource(src, repo)

			// Swap with the updated version
			s.mux.Lock()
			s.repo = repo
			s.mux.Unlock()
		case name := <-s.rm:
			fmt.Printf("RM %s\n", name)
		}
	}
}

// TODO: Routine that handles DNS queries with RLock()

// In POST /sources/add
//   name=XXX
//   key=value config in cfg
func (s *sources) handleAddSource(name, gentype string, cfg config) error {
	src := newSource(name)
	gen, err := makeGenerator(gentype, cfg)
	if err != nil {
		return fmt.Errorf("cannot start generator: %s", err)
	}
	src.gen = gen

	// TODO: Try send, don't wait if queue full, return err
	s.in <- src
	return nil
}

func main() {
	srcs := newSources()
	go srcs.run()

	srcs.handleAddSource("static", "date", makeConfig())

	// TODO: Http handler loop
	time.Sleep(1 * time.Second)
	fmt.Printf("%s\n", srcs.repo)
	os.Exit(1)
}
