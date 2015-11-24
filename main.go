package main

import (
	"log"
	"net/http"

	"github.com/dullgiulio/kuradns/cfg"
)

func main() {
	dnsListen := ":8053"
	httpListen := ":8080"

	srv := newServer()
	srv.start()

	// TODO: Will be called by HTTP handler.
	srv.handleSourceAdd("static", "date", cfg.MakeConfig())

	go srv.serveDNS(dnsListen, ".") // TODO: define zone?

	log.Fatal(http.ListenAndServe(httpListen, srv))
}
