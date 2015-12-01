package gen

// staticgen is a generator that yields constant entries. Used for testing.
type staticgen struct {
	ch chan RawEntry
}

func newStaticgen() *staticgen {
	s := &staticgen{
		ch: make(chan RawEntry),
	}
	go s.run()
	return s
}

func (s *staticgen) run() {
	entries := []RawEntry{
		MakeRawEntry("localhost", "127.0.0.1"),
		MakeRawEntry("some.host.com", "localhost"),
		MakeRawEntry("invalid.host", "invalid.host"),
	}
	for _, entry := range entries {
		s.ch <- entry
	}
	close(s.ch)
}

func (s *staticgen) Generate() RawEntry {
	return <-s.ch
}
