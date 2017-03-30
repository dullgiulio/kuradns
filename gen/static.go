// Copyright 2016 Giulio Iotti. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gen

import (
	"errors"

	"github.com/dullgiulio/kuradns/cfg"
)

// staticgen is a generator that yealds a single static entry
type staticgen struct {
	ch chan *RawEntry
}

func newStaticgen(c *cfg.Config) (*staticgen, error) {
	key, ok := c.Get("config.key")
	if !ok {
		return nil, errors.New("key not specified")
	}
	val, ok := c.Get("config.val")
	if !ok {
		return nil, errors.New("val not specified")
	}
	s := &staticgen{
		ch: make(chan *RawEntry),
	}
	go s.run(key, val)
	return s, nil
}

func (s *staticgen) run(key, val string) {
	s.ch <- NewRawEntry(key, val)
	close(s.ch)
}

func (s *staticgen) Generate() (*RawEntry, error) {
	return <-s.ch, nil
}
