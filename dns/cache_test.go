// Copyright 2015 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dns

import (
	"testing"

	"github.com/fcavani/e"
)

func TestLocalHost(t *testing.T) {
	addrs, err := LookupHostCache("localhost")
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}
	t.Log(addrs)
	for _, addr := range addrs {
		if addr != "::1" && addr != "127.0.0.1" {
			t.Fatal("wrong address")
		}
	}

}
