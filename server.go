// Copyright 2016 Giulio Iotti. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/dullgiulio/kuradns/cfg"
	"github.com/dullgiulio/kuradns/gen"
)

// Type of request
type reqtype int

const (
	// Add a source
	reqtypeAdd reqtype = iota
	// Remove/delete a source
	reqtypeDel
	// Update a source
	reqtypeUp
)

var (
	// Error returned when too many requests are queued for processing.
	errQueueFull = errors.New("queue full")
	// Invalid request type.
	errUnknownReqType = errors.New("unknown request type")
)

// response represents the value returned from a server operation.
type response error

// request includes a channel to get the response, the source to work on and the type of request.
type request struct {
	resp  chan response
	src   *source
	rtype reqtype
}

// makeRequest allocates a request of type t and source src.
func makeRequest(src *source, t reqtype) request {
	return request{
		src:   src,
		resp:  make(chan response),
		rtype: t,
	}
}

// done marks a request as processed.
func (r request) done() {
	close(r.resp)
}

// fail marks a request as processed with error.
func (r request) fail(err error) {
	r.resp <- err
}

// send tries to queue a request. An error is returned if the queue is full.
func (r request) send(ch chan<- request) error {
	select {
	case ch <- r:
		return nil
	default:
		close(r.resp)
		return errQueueFull
	}
}

// String representation for a request
func (r request) String() string {
	op := "unk"
	switch r.rtype {
	case reqtypeAdd:
		op = "add"
	case reqtypeDel:
		op = "rem"
	case reqtypeUp:
		op = "update"
	}
	return fmt.Sprintf("%s '%s'", op, r.src.name)
}

// server is the single instance that manages all accesses to the repository and
// coordinates access to it, between both the DNS and HTTP interfaces.
type server struct {
	verbose  bool
	fname    string
	srcs     sources
	repo     repository
	zone     host
	self     host
	soa      *soa
	ttl      time.Duration
	respPool sync.Pool
	mux      sync.RWMutex
	requests chan request
}

// newServer allocates a server instance. fname is the file where the session is restored
// and subsequently persisted; verbose controls the logging level; ttl the duration to
// apply to all DNS records served; self is the domain name of the local host.
func newServer(fname string, verbose bool, ttl time.Duration, zone, self host) *server {
	s := &server{
		fname:    fname,
		zone:     zone,
		self:     self,
		ttl:      ttl,
		verbose:  verbose,
		requests: make(chan request, 10), // TODO: buffering is a param
		soa:      newSoa(zone, self),
		repo:     makeRepository(),
		srcs:     makeSources(),
	}
	go s.run()
	if fname != "" {
		s.restoreSources()
	}
	return s
}

// jsonSource represent the persisted list of sources
type jsonSource struct {
	// Name of the source
	Name string
	// All configuration options as key value pairs
	Conf map[string]string
}

// restoreSources reads the JSON file of the sources and restartes
// all sources found. If starting a source failed, an error is logged
// and the source ignored.
func (s *server) restoreSources() {
	f, err := os.Open(s.fname)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatalf("cannot restore sources: %s", err)
		}
		return
	}
	defer f.Close()
	// unset s.fname so that while we add source again, they are not persisted
	// in an intermediate state.
	s.mux.Lock()
	fname := s.fname
	s.fname = ""
	s.mux.Unlock()

	var jsrcs []jsonSource
	if err := json.NewDecoder(f).Decode(&jsrcs); err != nil {
		log.Printf("cannot restore sources, error decoding JSON: %s", err)
		return
	}
	for _, v := range jsrcs {
		stype := v.Conf["source.type"]
		name := v.Name
		if err := s.handleSourceAdd(name, stype, cfg.FromMap(v.Conf)); err != nil {
			log.Printf("cannot restore source %s: %s", name, err)
		}
	}

	// Re-enable persitance.
	s.mux.Lock()
	s.fname = fname
	s.mux.Unlock()
}

// persistSources writes to fname the JSON with the sources
// currently configured and their configuration.
func (s *server) persistSources() {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.fname == "" {
		return
	}

	f, err := os.Create(s.fname)
	if err != nil {
		log.Printf("cannot persist sources: %s", err)
		return
	}
	defer f.Close()
	var i int
	jsrcs := make([]jsonSource, len(s.srcs))
	for _, v := range s.srcs {
		jsrcs[i] = jsonSource{
			Name: v.name,
			Conf: v.conf.Map(),
		}
		i++
	}
	if err := json.NewEncoder(f).Encode(&jsrcs); err != nil {
		log.Printf("cannot persist sources: %s", err)
		return
	}
}

// cloneRepo safely creates and returns a full copy of the current repository.
func (s *server) cloneRepo() repository {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.repo.clone()
}

// setRepo atomically changes the repository used by the server with repo.
func (s *server) setRepo(repo repository) {
	s.mux.Lock()
	s.repo = repo
	s.mux.Unlock()
}

// run serves requests queued on the requests channel. run logs errors and information if
// server is configured as verbose. run does not return.
func (s *server) run() {
	for req := range s.requests {
		switch req.rtype {
		case reqtypeAdd:
			if s.srcs.has(req.src.name) {
				req.fail(fmt.Errorf("%s: source already exists", req.String()))
				log.Printf("[error] sources: not added existing source %s", req.src.name)
				continue
			}
			repo := s.cloneRepo()
			repo.updateSource(req.src, s.zone, s.ttl)
			s.setRepo(repo)
			s.srcs[req.src.name] = req.src
			if s.verbose {
				log.Printf("[info] sources: added source %s", req.src.name)
			}
		case reqtypeDel:
			if !s.srcs.has(req.src.name) {
				req.fail(fmt.Errorf("%s: source not found", req.String()))
				log.Printf("[error] sources: not removed non-existing source %s", req.src.name)
				continue
			}
			repo := s.cloneRepo()
			repo.deleteSource(req.src)
			s.setRepo(repo)
			delete(s.srcs, req.src.name)
			if s.verbose {
				log.Printf("[info] sources: deleted source %s", req.src.name)
			}
		case reqtypeUp:
			if !s.srcs.has(req.src.name) {
				req.fail(fmt.Errorf("%s: source not found", req.String()))
				log.Printf("[error] sources: not updated non-existing source %s", req.src.name)
				continue
			}
			src := s.srcs[req.src.name]
			conf := src.conf
			stype, err := s.getFromConf(conf, "source.type")
			if err != nil {
				req.fail(fmt.Errorf("cannot restart source %s: %s", req.src.name, err))
				continue
			}
			gen, err := gen.MakeGenerator(stype, conf)
			if err != nil {
				req.fail(fmt.Errorf("cannot restart generator: %s", err))
				continue
			}
			src.gen = gen
			repo := s.cloneRepo()
			repo.deleteSource(src)
			repo.updateSource(src, s.zone, s.ttl)
			s.setRepo(repo)
			if s.verbose {
				log.Printf("[info] sources: updated source %s", src.name)
			}
		default:
			req.fail(errUnknownReqType)
		}
		s.persistSources()
		req.done()
	}
}
