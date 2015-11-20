package main

import (
	"github.com/dullgiulio/kuradns/cfg"
)

func main() {
	srv := newServer(true)
	srv.start()

	// TODO: Will be called by HTTP handler.
	srv.handleAddSource("static", "date", cfg.MakeConfig())

	/* go */ srv.serveDNS(":8053")

	// TODO: Http handler loop
}
