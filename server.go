package main

import (
	"errors"
	"fmt"
	"sync"
)

type reqtype int

const (
	reqtypeAdd reqtype = iota
	reqtypeDel
	reqtypeUp
)

var (
	errQueueFull      = errors.New("queue full")
	errUnknownReqType = errors.New("unknown request type")
)

type response error

type request struct {
	resp  chan response
	src   *source
	rtype reqtype
}

func makeRequest(src *source, t reqtype) request {
	return request{
		src:   src,
		resp:  make(chan response),
		rtype: t,
	}
}

func (r request) done() {
	close(r.resp)
}

func (r request) fail(err error) {
	r.resp <- err
}

func (r request) send(ch chan<- request) error {
	select {
	case ch <- r:
		return nil
	default:
		return errQueueFull
	}
}

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
	return fmt.Sprintf("[add %s]", op, r.src.name)
}

type server struct {
	// sources of requests status
	srcsReq sources
	// sources of satisfied status
	srcs      sources
	repo      repository
	zone      host
	self      host
	mux       sync.RWMutex
	requests  chan request
	processes chan request
}

func newServer() *server {
	return &server{
		requests:  make(chan request),
		processes: make(chan request, 10), // TODO: buffering is a param
		repo:      makeRepository(),
		srcsReq:   makeSources(),
		srcs:      makeSources(),
	}
}

func (s *server) cloneRepo() repository {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.repo.clone()
}

func (s *server) setRepo(repo repository) {
	s.mux.Lock()
	s.repo = repo
	s.mux.Unlock()
}

func (s *server) runWorker() {
	for req := range s.processes {
		repo := s.cloneRepo()
		switch req.rtype {
		case reqtypeAdd:
			if s.srcs.has(req.src.name) {
				continue
			}
			repo.updateSource(req.src)
			s.setRepo(repo)
			s.srcs[req.src.name] = req.src
		case reqtypeDel:
			if !s.srcs.has(req.src.name) {
				continue
			}
			repo.deleteSource(req.src)
			s.setRepo(repo)
			delete(s.srcs, req.src.name)
		case reqtypeUp:
			if !s.srcs.has(req.src.name) {
				continue
			}
			repo.deleteSource(req.src)
			repo.updateSource(req.src)
			s.setRepo(repo)
			s.srcs[req.src.name] = req.src
		}
	}
}

func (s *server) runHandler() {
	for req := range s.requests {
		switch req.rtype {
		case reqtypeAdd:
			if s.srcsReq.has(req.src.name) {
				req.fail(fmt.Errorf("source %s already exists", req.String()))
				continue
			}
			if err := req.send(s.processes); err != nil {
				req.fail(fmt.Errorf("cannot process %s: %s", req.String(), err))
				continue
			}
			s.srcsReq[req.src.name] = req.src
		case reqtypeDel:
			if !s.srcsReq.has(req.src.name) {
				req.fail(fmt.Errorf("source %s not found", req.String()))
				continue
			}
			if err := req.send(s.processes); err != nil {
				req.fail(fmt.Errorf("cannot process %s: %s", req.String(), err))
				continue
			}
			delete(s.srcsReq, req.src.name)
		case reqtypeUp:
			if !s.srcsReq.has(req.src.name) {
				req.fail(fmt.Errorf("source %s not found", req.String()))
				continue
			}
			if err := req.send(s.processes); err != nil {
				req.fail(fmt.Errorf("cannot process %s: %s", req.String(), err))
			}
		default:
			req.fail(errUnknownReqType)
		}
		req.done()
	}
}

func (s *server) start() {
	go s.runHandler()
	go s.runWorker()
}
