// Copyright 2016 Giulio Iotti. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gen

import (
	"fmt"
	"time"

	"github.com/dullgiulio/kuradns/cfg"
)

// dategen is a generator that yields entries containing the date. Used for testing.
type dategen struct {
	ch   chan RawEntry
	date string
	zone string
}

func newDategen(c *cfg.Config) *dategen {
	d := &dategen{
		ch:   make(chan RawEntry),
		date: time.Now().UTC().Format("20060102150405"),
		zone: c.GetVal("dns.zone", "lan"),
	}
	go d.run()
	return d
}

func (d *dategen) run() {
	d.ch <- MakeRawEntry(fmt.Sprintf("%s.%s", d.date, d.zone), "127.0.0.1")
	close(d.ch)
}

func (d *dategen) Generate() RawEntry {
	return <-d.ch
}
