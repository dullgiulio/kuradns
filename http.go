package main

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/dullgiulio/kuradns/cfg"
	"github.com/dullgiulio/kuradns/gen"
)

func (s *server) handleSourceAdd(name, gentype string, conf cfg.Config) {
	src := newSource(name)
	gen, err := gen.MakeGenerator(gentype, conf)
	if err != nil {
		log.Printf("cannot start generator: %s", err)
		return
	}
	src.gen = gen

	req := makeRequest(src, reqtypeAdd)

	s.requests <- req
	err = <-req.resp
	if err != nil {
		log.Print("cannot add source: ", err)
	}
}

func (s *server) handleSourceDelete(name string) {
	src := newSource(name)
	req := makeRequest(src, reqtypeDel)

	s.requests <- req
	err := <-req.resp
	if err != nil {
		log.Print("cannot remove source: ", err)
	}
}

func (s *server) handleSourceUpdate(name string) {
	src := newSource(name)
	req := makeRequest(src, reqtypeUp)

	s.requests <- req
	err := <-req.resp
	if err != nil {
		log.Print("cannot update source: ", err)
	}
}

// take last value in case of duplicates
func (s *server) configFromForm(form url.Values) cfg.Config {
	cf := cfg.MakeConfig()
	for k, vs := range form {
		if strings.HasPrefix(k, "conf.") {
			cf[k[5:]] = vs[len(vs)-1]
		}
	}
	return cf
}

func (s *server) httpHandlePOST(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	conf := s.configFromForm(r.Form)
	switch r.URL.Path {
	case "/source/add":
		// TODO: Take name and type from form
		s.handleSourceAdd("name", "gentype", conf)
	case "/source/delete":
		s.handleSourceDelete("name")
		// TODO: Take name from form
	case "/source/update":
		// TODO: Take name from form
		s.handleSourceUpdate("name")
	}
	return nil
}

func (s *server) httpHandleGET(w http.ResponseWriter, r *http.Request) {
	// Help, status, etc.
	log.Print("GET requests not implemented")
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		s.httpHandlePOST(w, r)
	default:
		s.httpHandleGET(w, r)
	}
}
