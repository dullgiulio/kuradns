package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/dullgiulio/kuradns/cfg"
	"github.com/dullgiulio/kuradns/gen"

	"github.com/miekg/dns"
)

type server struct {
	debug bool
	repo  repository
	mux   sync.RWMutex
	// Sources added or refreshed (TODO: Add needs params)
	in chan *source
	// Sources to be removed
	rm chan string
}

func newServer(debug bool) *server {
	s := &server{
		debug: debug,
		repo:  makeRepository(),
		in:    make(chan *source), // TODO: this will be buffered
		rm:    make(chan string),
	}
	dns.HandleFunc(".", s.handleQuery) // TODO: Define zone
	return s
}

func (s *server) updateSource(src *source, repo repository) error {
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
		repo.add(rentry.Source, makeRecord(rentry.Source, rentry.Target, res, src))
		if s.debug {
			log.Printf("ADD: %s -> %s [%s]\n", rentry.Source, rentry.Target, res)
		}
	}
	return nil
}

func (s *server) run() {
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
			fmt.Printf("RM %s\n", name) // TODO: Remove source
		}
	}
}

// TODO: Routine that handles DNS queries with RLock()
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

func (s *server) serveDNS(addr string) {
	errCh := make(chan error)

	go s.serveNetDNS(addr, "udp", errCh)
	go s.serveNetDNS(addr, "tcp", errCh)

	if err := <-errCh; err != nil {
		log.Fatal("Cannot start DNS server: ", err)
	}
}

// In POST /server/add
//   name=XXX
//   key=value config in cfg
func (s *server) handleAddSource(name, gentype string, conf cfg.Config) error {
	src := newSource(name)
	gen, err := gen.MakeGenerator(gentype, conf)
	if err != nil {
		return fmt.Errorf("cannot start generator: %s", err)
	}
	src.gen = gen

	// TODO: Try send, don't wait if queue full, return err
	s.in <- src
	return nil
}
