// Copyright 2016 Giulio Iotti. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cfg

import (
	"encoding/json"
	"io"
	"strings"
)

// Config is a map containing configuration key-value pairs.
type Config struct {
	m map[string]string
}

// NewConfig allocates a configuration map.
func NewConfig() *Config {
	return &Config{make(map[string]string)}
}

// FromMap converts a string map into a Config object.
func FromMap(m map[string]string) *Config {
	return &Config{m}
}

// Map returns the underlying map of a Config object.
func (cf *Config) Map() map[string]string {
	return cf.m
}

// Put adds a key value pair, overriding any previous entry.
func (cf *Config) Put(k, v string) {
	cf.m[k] = v
}

// Get returns the value for a key k or false if not present.
func (cf *Config) Get(k string) (string, bool) {
	v, ok := cf.m[k]
	return v, ok
}

// GetVal returns the value for a key k or defaultVal if not present.
func (cf *Config) GetVal(k, defaultVal string) string {
	if v, ok := cf.m[k]; ok {
		return v
	}
	return defaultVal
}

// FromJSON unmarshals JSON data read from r into a Config object.
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
