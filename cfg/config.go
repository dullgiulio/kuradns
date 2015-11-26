package cfg

import (
	"encoding/json"
	"io"
)

type Config struct {
	m map[string]string
}

func NewConfig() *Config {
	return &Config{make(map[string]string)}
}

func (cf *Config) Map() map[string]string {
	return cf.m
}

func (cf *Config) Put(k, v string) {
	cf.m[k] = v
}

func (cf *Config) Get(k string) (string, bool) {
	v, ok := cf.m[k]
	return v, ok
}

func (cf *Config) FromJSON(r io.Reader) error {
	return json.NewDecoder(r).Decode(&cf.m)
}
