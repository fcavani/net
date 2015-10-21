// Copyright 2015 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"testing"
)

func TestHostnameFqdn(t *testing.T) {
	fqdn, err := HostnameFqdn()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(fqdn)
}
