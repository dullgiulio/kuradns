package gen

import (
	"errors"

	"github.com/dullgiulio/kuradns/cfg"
)

type Generator interface {
	// generate reuturns an entry between a hostname its destination IP/hostname.
	// When no more pairs are available, generate should return an empty rawentry
	Generate() RawEntry
}

var ErrInvalidGenerator = errors.New("invalid generator name")

func MakeGenerator(name string, conf *cfg.Config) (Generator, error) {
	switch name { // strings.Lower
	case "date":
		g := NewDategen()
		go g.run()
		return g, nil
	case "static":
		g := NewStaticgen()
		go g.run()
		return g, nil
	default:
		return nil, ErrInvalidGenerator
	}
}
