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
	case "mysql":
		return newMysql(conf)
	case "date":
		g := newDategen(conf)
		return g, nil
	case "static":
		return newStaticgen(conf)
	default:
		return nil, ErrInvalidGenerator
	}
}
