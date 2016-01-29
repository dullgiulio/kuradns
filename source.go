package main

import (
	"github.com/dullgiulio/kuradns/cfg"
	"github.com/dullgiulio/kuradns/gen"
)

type sources map[string]*source

type source struct {
	name string
	conf *cfg.Config
	gen  gen.Generator
}

func makeSources() sources {
	return make(map[string]*source)
}

func (s sources) has(k string) bool {
	_, ok := s[k]
	return ok
}

func newSource(name string, conf *cfg.Config) *source {
	return &source{
		name: name,
		conf: conf,
	}
}

func (s source) String() string {
	return s.name
}
