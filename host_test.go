package main

import (
	"testing"
)

func TestMatchWithWildcard(t *testing.T) {
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
			t.Errorf("'%s' == '%s' should be %s but was %s", p.a, p.b, p.res, res)
		}
	}
}
