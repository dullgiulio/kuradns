package main

import "github.com/dullgiulio/kuradns/gen"

type source struct {
	name string
	gen  gen.Generator
}

func newSource(name string) *source {
	return &source{
		name: name,
	}
}

func (s source) String() string {
	return s.name
}
