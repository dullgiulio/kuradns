// Copyright 2016 Giulio Iotti. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kuradns

import "testing"

func TestHostHasSuffix(t *testing.T) {
	h1 := host("some.host")
	h2 := host("host")
	h3 := host("not")

	if !h1.hasSuffix(h2) {
		t.Errorf("%s has suffix %s", h1.dns(), h2.dns())
	}

	if h1.hasSuffix(h3) {
		t.Errorf("%s does not have suffix %s", h1.dns(), h3.dns())
	}
}
