package gen

import (
	"fmt"
	"time"
)

// dategen is a generator that yields entries containing the date. Used for testing.
type dategen struct {
	ch   chan RawEntry
	date string
}

func newDategen() *dategen {
	d := &dategen{
		ch:   make(chan RawEntry),
		date: time.Now().UTC().Format("20060102150405"),
	}
	go d.run()
	return d
}

func (d *dategen) run() {
	d.ch <- MakeRawEntry(fmt.Sprintf("%s.%s", d.date, "mydomain.test."), "127.0.0.1")
	close(d.ch)
}

func (d *dategen) Generate() RawEntry {
	return <-d.ch
}
