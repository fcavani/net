// Copyright 2015 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dns

import (
	"net/url"
	"testing"

	"github.com/fcavani/e"
)

func TestResolveUrl(t *testing.T) {
	url, err := url.Parse("http://localhost:8080/foo.html?q=search#fragment")
	if err != nil {
		t.Fatal("parse failed", err)
	}
	u, err := ResolveUrl(url)
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}
	if u.Host != "127.0.0.1:8080" && u.Host != "[::1]:8080" {
		t.Fatal("can't resolve", u)
	}
	t.Log(u)
}

func TestPtr(t *testing.T) {
	host, err := LookupIp("200.149.119.183")
	if err != nil {
		t.Fatal(err)
	}
	if host != "183.119.149.200.in-addr.arpa.telemar.net.br" {
		t.Fatal("wrong host")
	}
	host, err = LookupIp("2800:3f0:4004:800::1013")
	if err != nil && !e.Contains(err, "can't resolve") {
		t.Fatal(err)
	}

	host, err = LookupIp("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if host != "localhost" {
		t.Fatal("wrong host")
	}
}
