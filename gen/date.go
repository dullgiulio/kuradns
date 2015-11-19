package gen

import (
	"fmt"
	"time"
)

// Dategen is a generator that yields entries containing the date. Used for testing.
type Dategen struct {
	ch   chan RawEntry
	date string
}

func NewDategen() *Dategen {
	return &Dategen{
		ch:   make(chan RawEntry),
		date: time.Now().UTC().Format("20060102150405"),
	}
}

func (d *Dategen) run() {
	d.ch <- MakeRawEntry(fmt.Sprintf("%s.%s", d.date, "mydomain.test."), "127.0.0.1")
	close(d.ch)
}

func (d *Dategen) Generate() RawEntry {
	return <-d.ch
}
