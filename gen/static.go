package gen

import (
	"fmt"

	"github.com/dullgiulio/kuradns/cfg"
)

// staticgen is a generator that yields constant entries. Used for testing.
type staticgen struct {
	zone string
	ch   chan RawEntry
}

func newStaticgen(c *cfg.Config) *staticgen {
	s := &staticgen{
		zone: "." + c.GetVal("dns.zone", "lan"),
		ch:   make(chan RawEntry),
	}
	go s.run()
	return s
}

func (s *staticgen) run() {
	entries := []RawEntry{
		MakeRawEntry(fmt.Sprintf("localhost%s", s.zone), "127.0.0.1"),
		MakeRawEntry(fmt.Sprintf("some.host%s", s.zone), "localhost"),
		MakeRawEntry(fmt.Sprintf("invalid-host%s", s.zone), "invalid.host"),
	}
	for _, entry := range entries {
		s.ch <- entry
	}
	close(s.ch)
}

func (s *staticgen) Generate() RawEntry {
	return <-s.ch
}
