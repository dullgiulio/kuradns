// Copyright 2016 Giulio Iotti. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gen

type RawEntry struct {
	Source, Target string
}

func MakeRawEntry(s, t string) RawEntry {
	return RawEntry{s, t}
}

func EmptyRawEntry() RawEntry {
	return RawEntry{}
}

func (r RawEntry) IsEmpty() bool {
	return r.Source == "" || r.Target == ""
}
