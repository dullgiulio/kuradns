package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/dullgiulio/kuradns/cfg"
	"github.com/dullgiulio/kuradns/gen"
)

var errUnhandledURL = errors.New("unhandled URL")

func (s *server) handleHttpError(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, "An error occurred; please refer to the logs for more information", 500)
	log.Printf("http: %s %s %s: %s", r.RemoteAddr, r.Method, r.URL.Path, err)
}

func (s *server) handleSourceAdd(name, gentype string, conf *cfg.Config) error {
	src := newSource(name)
	gen, err := gen.MakeGenerator(gentype, conf)
	if err != nil {
		return fmt.Errorf("cannot start generator: %s", err)
	}
	src.gen = gen

	req := makeRequest(src, reqtypeAdd)

	s.requests <- req
	err = <-req.resp
	if err != nil {
		return fmt.Errorf("cannot add source: %s", err)
	}
	return nil
}

func (s *server) handleSourceDelete(name string) error {
	src := newSource(name)
	req := makeRequest(src, reqtypeDel)

	s.requests <- req
	err := <-req.resp
	if err != nil {
		return fmt.Errorf("cannot remove source: %s", err)
	}
	return err
}

func (s *server) handleSourceUpdate(name string) error {
	src := newSource(name)
	req := makeRequest(src, reqtypeUp)

	s.requests <- req
	err := <-req.resp
	if err != nil {
		return fmt.Errorf("cannot update source: %s", err)
	}
	return nil
}

func (s *server) handleDnsDump(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/plain")
	wb := bufio.NewWriter(w)

	s.mux.RLock()
	defer s.mux.RUnlock()

	if err := s.repo.WriteTo(wb); err != nil {
		return err
	}

	return wb.Flush()
}

// take last value in case of duplicates
func (s *server) configFromForm(cf *cfg.Config, form url.Values) error {
	for k, vs := range form {
		cf.Put(k, vs[len(vs)-1])
	}
	return nil
}

func (s *server) configFromJSON(cf *cfg.Config, r io.Reader) error {
	if err := cf.FromJSON(r); err != nil {
		return fmt.Errorf("cannot parse JSON: %s", err)
	}
	return nil
}

func (s *server) getFromConf(cf *cfg.Config, key string) (string, error) {
	if v, ok := cf.Get(key); ok {
		return v, nil
	}
	return "", fmt.Errorf("required parameter %s not found", key)
}

func (s *server) parseBodyData(w http.ResponseWriter, r *http.Request) (*cfg.Config, error) {
	cf := cfg.NewConfig()
	// JSON data as POST body
	if r.Header.Get("Content-Type") == "application/json" {
		return cf, s.configFromJSON(cf, r.Body)
	}
	// Normal URL-encoded form
	if err := r.ParseForm(); err != nil {
		return cf, fmt.Errorf("cannot parse form: %s", err)
	}
	return cf, s.configFromForm(cf, r.Form)
}

func (s *server) httpHandlePOST(w http.ResponseWriter, r *http.Request) error {
	conf, err := s.parseBodyData(w, r)
	// TODO: Set server config stuff (zone, self) to be used by generators.
	if err != nil {
		return err
	}
	switch r.URL.Path {
	case "/source/add":
		sname, err := s.getFromConf(conf, "source.name")
		if err != nil {
			return err
		}
		stype, err := s.getFromConf(conf, "source.type")
		if err != nil {
			return err
		}
		return s.handleSourceAdd(sname, stype, conf)
	case "/source/delete":
		sname, err := s.getFromConf(conf, "source.name")
		if err != nil {
			return err
		}
		return s.handleSourceDelete(sname)
	case "/source/update":
		sname, err := s.getFromConf(conf, "source.name")
		if err != nil {
			return err
		}
		return s.handleSourceUpdate(sname)
	}
	return errUnhandledURL
}

func (s *server) httpHandleGET(w http.ResponseWriter, r *http.Request) error {
	// TODO: Help, status, etc.
	switch r.URL.Path {
	case "/dns/dump":
		return s.handleDnsDump(w, r)
	case "/favicon.ico":
		// Shut up on bogus requests
		return nil
	}
	return errUnhandledURL
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	if s.verbose {
		log.Printf("[info] http: request %s %s", r.Method, r.URL.Path)
	}
	switch r.Method {
	case "POST":
		err = s.httpHandlePOST(w, r)
	default:
		err = s.httpHandleGET(w, r)
	}
	if err != nil {
		s.handleHttpError(w, r, err)
	}
}