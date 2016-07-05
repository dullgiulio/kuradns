// Copyright 2016 Giulio Iotti. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kuradns

import (
	"fmt"

	"github.com/dullgiulio/kuradns/cfg"
	"github.com/dullgiulio/kuradns/gen"
)

// sources is the collection of source objects keyed by name.
type sources map[string]*source

// source is the generator of DNS entries with its configuration and name.
type source struct {
	name string
	conf *cfg.Config
	gen  gen.Generator
}

// makeSources allocates a sources collection.
func makeSources() sources {
	return make(map[string]*source)
}

// has returns true if the sources collection contains a source named k.
func (s sources) has(k string) bool {
	_, ok := s[k]
	return ok
}

// newSource allocates a source.
func newSource(name string, conf *cfg.Config) *source {
	return &source{
		name: name,
		conf: conf,
	}
}

// initGenerator initializes the generator for a new production of key/values.
func (s *source) initGenerator() error {
	var err error
	stype, ok := s.conf.Get("source.type")
	if !ok {
		return fmt.Errorf("cannot start generator %s: key source.type not found", s.name)
	}
	s.gen, err = gen.MakeGenerator(stype, s.conf)
	if err != nil {
		return fmt.Errorf("cannot start generator: %s", err)
	}
	return nil
}

// String representation of a source is its name.
func (s *source) String() string {
	return s.name
}
