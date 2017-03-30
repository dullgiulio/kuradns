// Copyright 2016 Giulio Iotti. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kuradns

import (
	"testing"
)

func TestMatchWildcardString(t *testing.T) {
	for _, p := range []struct {
		a   string
		b   string
		res bool
	}{
		{"aaa", "bbb", false},
		{"*", "bbb", true},
		{"aa*", "aaa", true},
		{"aa*", "aa", true},
		{"*aa", "baa", true},
		{"*aa", "bba", false},
		{"a*a", "aba", true},
		{"a*ab", "aabab", true},
	} {
		if res := matchWildcard(p.a, p.b); res != p.res {
			t.Errorf("'%s' == '%s' should be %t but was %t", p.a, p.b, p.res, res)
		}
	}
}

func TestMatchWildcardHost(t *testing.T) {
	for _, p := range []struct {
		a   string
		b   string
		res bool
	}{
		{"*.test.local", "a.test.local", true},
		{"a.*.local", "a.blah.local", true},
		{"*.*.local", "a.test.local", true},
		{"a.b.c", "a.b.c", true},
		{"a.c.b", "a.b.c", false},
		{"*.test.*", "a.test.local", true},
		{"a*.test.l*", "aa.test.local", true},
		{"a*.test.l*", "a.test.l", true},
		{"*a.test.*l", "ba.test.bl", true},
	} {
		if res := host(p.a).match(host(p.b)); res != p.res {
			t.Errorf("'%s' == '%s' should be %t but was %t", p.a, p.b, p.res, res)
		}
	}
}
