// Copyright 2016 Giulio Iotti. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kuradns

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/dullgiulio/kuradns/cfg"
)

var errUnhandledURL = errors.New("unhandled URL")

func (s *server) handleHttpError(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, "An error occurred; please refer to the logs for more information", 500)
	log.Printf("[error] http: %s %s %s: %s", r.RemoteAddr, r.Method, r.URL.Path, err)
}

func (s *server) handleSourceAdd(name, gentype string, conf *cfg.Config) error {
	src := newSource(name, conf)
	if err := src.initGenerator(); err != nil {
		return fmt.Errorf("cannot start generator: %s", err)
	}

	req := makeRequest(src, reqtypeAdd)

	if err := req.send(s.requests); err != nil {
		return fmt.Errorf("cannot process %s: %s", req.String(), err)
	}
	if err := <-req.resp; err != nil {
		return fmt.Errorf("cannot add source: %s", err)
	}
	return nil
}

func (s *server) handleSourceDelete(name string) error {
	src := newSource(name, nil)
	req := makeRequest(src, reqtypeDel)

	if err := req.send(s.requests); err != nil {
		return fmt.Errorf("cannot process %s: %s", req.String(), err)
	}
	if err := <-req.resp; err != nil {
		return fmt.Errorf("cannot remove source: %s", err)
	}
	return nil
}

func (s *server) handleSourceUpdate(name string) error {
	src := newSource(name, nil)
	req := makeRequest(src, reqtypeUp)

	if err := req.send(s.requests); err != nil {
		return fmt.Errorf("cannot process %s: %s", req.String(), err)
	}
	if err := <-req.resp; err != nil {
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

func (s *server) handleSourceList(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/plain")
	wb := bufio.NewWriter(w)

	s.mux.RLock()
	defer s.mux.RUnlock()

	for _, src := range s.srcs {
		fmt.Fprintf(wb, "%s %s\n", src.name, src.conf.GetVal("source.type", "unknown"))
	}

	return wb.Flush()
}

// take last value in case of duplicates
func (s *server) configFromForm(cf *cfg.Config, form url.Values) error {
	for k, vs := range form {
		if strings.HasPrefix(k, "config.") || strings.HasPrefix(k, "source.") {
			cf.Put(k, vs[len(vs)-1])
		}
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
		if v == "" {
			return "", fmt.Errorf("required parameter %s is empty", key)
		}
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
	var err error
	conf, err := s.parseBodyData(w, r)
	if err != nil {
		return err
	}
	conf.Put("dns.zone", s.zone.browser())
	conf.Put("dns.self", s.self.browser())

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
		err = s.handleSourceAdd(sname, stype, conf)
	case "/source/delete":
		sname, err := s.getFromConf(conf, "source.name")
		if err != nil {
			return err
		}
		err = s.handleSourceDelete(sname)
	case "/source/update":
		sname, err := s.getFromConf(conf, "source.name")
		if err != nil {
			return err
		}
		err = s.handleSourceUpdate(sname)
	default:
		return errUnhandledURL
	}
	if err == nil {
		s.update()
	}
	return err
}

func (s *server) httpHandleGET(w http.ResponseWriter, r *http.Request) error {
	// TODO: Help, status, etc.
	switch r.URL.Path {
	case "/source/list":
		return s.handleSourceList(w, r)
	case "/dns/dump":
		return s.handleDnsDump(w, r)
	case "/favicon.ico":
		// Shut up on bogus requests
		http.NotFound(w, r)
		return nil
	}
	return errUnhandledURL
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	if s.verbose {
		log.Printf("[info] http: tcp(%s): request %s %s", r.RemoteAddr, r.Method, r.URL.Path)
	}
	switch r.Method {
	case "POST", "PUT":
		err = s.httpHandlePOST(w, r)
	default:
		err = s.httpHandleGET(w, r)
	}
	if err != nil {
		s.handleHttpError(w, r, err)
	}
}
