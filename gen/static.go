package gen

// staticgen is a generator that yields constant entries. Used for testing.
type Staticgen struct {
	ch chan RawEntry
}

func NewStaticgen() *Staticgen {
	return &Staticgen{
		ch: make(chan RawEntry),
	}
}

func (s *Staticgen) run() {
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

func (s *Staticgen) Generate() RawEntry {
	return <-s.ch
}
