package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	var (
		dnsListen  = flag.String("dns", ":8053", "`HOST:PORT` to listen for DNS requests (both UDP and TCP)")
		httpListen = flag.String("http", ":8080", "`HOST:PORT` to listen for HTTP requests")
		zone       = flag.String("zone", "lan", "`ZONE` domain name to serve, without preceding dot")
		hostname   = flag.String("host", "localhost", "Hostname `HOSTNAME` representing this DNS server itself")
		info       = flag.Bool("info", false, "Show log messages on client requests")
	)
	flag.Usage = func() {
		// TODO: Write extensive usage of HTTP API
		fmt.Fprintf(os.Stderr, "Usage of kuradns:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	srv := newServer(*info)
	srv.start()

	go srv.serveDNS(*dnsListen, host(*zone), host(*hostname))
	log.Fatal(http.ListenAndServe(*httpListen, srv))
}
