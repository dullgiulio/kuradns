package main

import "github.com/dullgiulio/kuradns/gen"

type sources map[string]*source

type source struct {
	name string
	gen  gen.Generator
}

func makeSources() sources {
	return make(map[string]*source)
}

func (s sources) has(k string) bool {
	_, ok := s[k]
	return ok
}

func newSource(name string) *source {
	return &source{
		name: name,
	}
}

func (s source) String() string {
	return s.name
}
