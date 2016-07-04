package main

import (
	"strings"

	"github.com/miekg/dns"
)

// A host is a FQDN or common representation of a DNS address.
type host string

// browser returns the FQDN without the trailing dot.
func (h host) browser() string {
	return strings.TrimSuffix(string(h), ".")
}

// dns returns the FQDN.
func (h host) dns() string {
	return dns.Fqdn(string(h))
}

// hasSuffix returns true if h has suffix h2.
func (h host) hasSuffix(h2 host) bool {
	return strings.HasSuffix(h.browser(), h2.browser())
}

// hasWildcard returns true if h is a wildcard host.
func (h host) hasWildcard() bool {
	return countByte(string(h), '*') > 0
}

// matchWildcard matches a wildcard host h with non-wildcard host h2.
func (h host) matchWildcard(h2 host) bool {
	// TODO: Rewrite this without using splitting to save on allocations.
	hs := strings.Split(h.browser(), ".")
	h2s := strings.Split(h2.browser(), ".")
	if len(hs) != len(h2s) {
		return false
	}
	for i := 0; i < len(hs); i++ {
		if !matchWildcard(hs[i], h2s[i]) {
			return false
		}
	}
	return true
}

// match matches host h with host h2. Either can be a wildcard host.
// If both hosts are wildcard, match will return true if they are the
// exact same string.
func (h host) match(h2 host) bool {
	if h.hasWildcard() {
		if h2.hasWildcard() {
			return h == h2
		}
		return h.matchWildcard(h2)
	}
	if h2.hasWildcard() {
		return h2.matchWildcard(h)
	}
	return h == h2
}

// countByte returns the number of occurrences of b in s.
func countByte(s string, b byte) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			n++
		}
	}
	return n
}

// matchWildcard matches wildcard string w against normal string s.
// Wilcard string w can only contain one star but can contain suffix
// or prefix.
func matchWildcard(w, s string) bool {
	hasWildcard := countByte(w, '*')
	if hasWildcard == 0 {
		return w == s
	}
	// XXX: We do not support matching against multiple wildcards.
	if hasWildcard > 1 {
		return false
	}
	if w == "*" {
		return true
	}
	ws := strings.Split(w, "*")
	if ws[0] == "" {
		return strings.HasSuffix(s, ws[1])
	}
	if ws[1] == "" {
		return strings.HasPrefix(s, ws[0])
	}
	return strings.HasPrefix(s, ws[0]) && strings.HasSuffix(s, ws[1])
}
