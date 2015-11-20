package main

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/dullgiulio/kuradns/cfg"
	"github.com/dullgiulio/kuradns/gen"

	"github.com/miekg/dns"
)

type reqtype int

const (
	reqtypeAdd reqtype = iota
	reqtypeRem
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
	case reqtypeRem:
		op = "rem"
	case reqtypeUp:
		op = "update"
	}
	return fmt.Sprintf("[add %s]", op, r.src.name)
}

type server struct {
	debug bool
	// sources of requests status
	srcsReq sources
	// sources of satisfied status
	srcs      sources
	repo      repository
	mux       sync.RWMutex
	requests  chan request
	processes chan request
}

func newServer(debug bool) *server {
	s := &server{
		debug:     debug,
		requests:  make(chan request),
		processes: make(chan request, 10), // TODO: buffering is a param
		repo:      makeRepository(),
		srcsReq:   makeSources(),
		srcs:      makeSources(),
	}
	dns.HandleFunc(".", s.handleQuery) // TODO: Define zone
	return s
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
			repo.updateSource(req.src, s.debug)
			s.setRepo(repo)
			s.srcs[req.src.name] = req.src
		case reqtypeRem:
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
			repo.updateSource(req.src, s.debug)
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
		case reqtypeRem:
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

	req := makeRequest(src, reqtypeAdd)

	// TODO: Try send, don't wait if queue full, return err
	s.requests <- req
	err = <-req.resp
	if err != nil {
		log.Print("cannot add source: ", err)
	}
	return nil
}
