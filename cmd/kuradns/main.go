// Copyright 2016 Giulio Iotti. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dullgiulio/kuradns"
)

func main() {
	var (
		dnsListen  = flag.String("dns", ":8053", "`HOST:PORT` to listen for DNS requests (both UDP and TCP)")
		httpListen = flag.String("http", ":8080", "`HOST:PORT` to listen for HTTP requests")
		zone       = flag.String("zone", "lan", "`ZONE` domain name to serve, without preceding dot")
		hostname   = flag.String("host", "localhost", "Hostname `HOSTNAME` representing this DNS server itself")
		save       = flag.String("save", "", "Save or restore sources from/to file `F`")
		info       = flag.Bool("info", false, "Show log messages on client requests")
		ttl        = flag.Duration("ttl", 1*time.Hour, "Duration `D` to be cached for DNS responses")
	)
	flag.Usage = func() {
		// TODO: Write extensive usage of HTTP API
		fmt.Fprintf(os.Stderr, "Usage of kuradns:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	srv := kuradns.NewServer(*save, *zone, *hostname, *info, *ttl)

	go srv.ServeDNS(*dnsListen)
	log.Printf("[info] http: listening on %s", *httpListen)
	log.Fatal(http.ListenAndServe(*httpListen, srv))
}
