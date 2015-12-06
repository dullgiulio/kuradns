package cfg

import (
	"encoding/json"
	"io"
	"strings"
)

type Config struct {
	m map[string]string
}

func NewConfig() *Config {
	return &Config{make(map[string]string)}
}

func FromMap(m map[string]string) *Config {
	return &Config{m}
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

func (cf *Config) GetVal(k, defaultVal string) string {
	if v, ok := cf.m[k]; ok {
		return v
	}
	return defaultVal
}

func (cf *Config) FromJSON(r io.Reader) error {
	m := make(map[string]string)
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return err
	}
	for k, v := range m {
		if strings.HasPrefix(k, "config.") || strings.HasPrefix(k, "source.") {
			cf.m[k] = v
		}
	}
	return nil
}
