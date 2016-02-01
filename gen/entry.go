// Copyright 2016 Giulio Iotti. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gen

// RawEntry is a pair of source and target addresses or domains to be resolved by the DNS server.
type RawEntry struct {
	Source, Target string
}

// MakeRawEntry allocates a raw entry for source s and target t.
func MakeRawEntry(s, t string) RawEntry {
	return RawEntry{s, t}
}

// EmptyRawEntry allocates an empty raw entry.
func EmptyRawEntry() RawEntry {
	return RawEntry{}
}

// IsEmpty checks whether an entry is empty.
func (r RawEntry) IsEmpty() bool {
	return r.Source == "" || r.Target == ""
}
