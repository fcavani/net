// Copyright 2015 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// Start date:        2014-10-10
// Last modification: 2014-

package url

import (
	"testing"

	"github.com/fcavani/e"
)

func TestSocket(t *testing.T) {
	method, path, err := Socket("unix(/path)")
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}
	if method != "unix" {
		t.Fatal("wrong path", method)
	}
	if path != "/path" {
		t.Fatal("wrong path", path)
	}
}

func TestParse(t *testing.T) {
	url, err := ParseWithSocket("http://localhost")
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}
	t.Logf("%#v\n", url)
	if url.Scheme != "http" {
		t.Fatal("wrong scheme")
	}
	if url.Host != "localhost" {
		t.Fatal("wrong host")
	}

	url, err = ParseWithSocket("http://www.isp.net:80/path#frag")
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}
	t.Logf("%#v\n", url)
	if url.Scheme != "http" {
		t.Fatal("wrong scheme")
	}
	if url.Host != "www.isp.net:80" {
		t.Fatal("wrong host")
	}
	if url.Path != "/path" {
		t.Fatal("wrong path")
	}
	if url.Fragment != "frag" {
		t.Fatal("wrong fragment")
	}

	url, err = ParseWithSocket("http://www.isp.net:80/path?q=1")
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}
	t.Logf("%#v\n", url)
	if url.Scheme != "http" {
		t.Fatal("wrong scheme")
	}
	if url.Host != "www.isp.net:80" {
		t.Fatal("wrong host")
	}
	if url.Path != "/path" {
		t.Fatal("wrong path")
	}
	if url.RawQuery != "q=1" {
		t.Fatal("wrong fragment")
	}

	url, err = ParseWithSocket("http://www.isp.net:80/path?q=1")
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}
	t.Logf("%#v\n", url)
	if url.Scheme != "http" {
		t.Fatal("wrong scheme")
	}
	if url.Host != "www.isp.net:80" {
		t.Fatal("wrong host")
	}
	if url.Path != "/path" {
		t.Fatal("wrong path")
	}
	if url.RawQuery != "q=1" {
		t.Fatal("wrong fragment")
	}

	url, err = ParseWithSocket("http://www.isp.net:80/path?q=1#frag")
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}
	t.Logf("%#v\n", url)
	if url.Scheme != "http" {
		t.Fatal("wrong scheme")
	}
	if url.Host != "www.isp.net:80" {
		t.Fatal("wrong host")
	}
	if url.Path != "/path" {
		t.Fatal("wrong path")
	}
	if url.RawQuery != "q=1" {
		t.Fatal("wrong fragment")
	}
	if url.Fragment != "frag" {
		t.Fatal("wrong fragment")
	}

	url, err = ParseWithSocket("http://unix(/var/run/app.socket)/path#frag")
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}
	t.Logf("%#v\n", url)
	if url.Scheme != "http" {
		t.Fatal("wrong scheme")
	}
	if url.Host != "unix(/var/run/app.socket)" {
		t.Fatal("wrong host")
	}
	if url.Path != "/path" {
		t.Fatal("wrong path")
	}
	if url.Fragment != "frag" {
		t.Fatal("wrong fragment")
	}

	url, err = ParseWithSocket("mysql://tcp(www.isp.net)/path#frag")
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}
	t.Logf("%#v\n", url)
	if url.Scheme != "mysql" {
		t.Fatal("wrong scheme")
	}
	if url.Host != "tcp(www.isp.net)" {
		t.Fatal("wrong host")
	}
	if url.Path != "/path" {
		t.Fatal("wrong path")
	}
	if url.Fragment != "frag" {
		t.Fatal("wrong fragment")
	}

	url, err = ParseWithSocket("socket:///var/run/file.socket#frag")
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}
	t.Logf("%#v\n", url)
	if url.Scheme != "socket" {
		t.Fatal("wrong scheme")
	}
	if url.Host != "/var/run/file.socket" {
		t.Fatal("wrong host")
	}
	if url.Path != "" {
		t.Fatal("wrong path")
	}
	if url.Fragment != "frag" {
		t.Fatal("wrong fragment")
	}

	url, err = ParseWithSocket("socket:///var/run/file.socket")
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}
	t.Logf("%#v\n", url)
	if url.Scheme != "socket" {
		t.Fatal("wrong scheme")
	}
	if url.Host != "/var/run/file.socket" {
		t.Fatal("wrong host")
	}
	if url.Path != "" {
		t.Fatal("wrong path")
	}
	if url.Fragment != "" {
		t.Fatal("wrong fragment")
	}

	url, err = ParseWithSocket("foo://C:/bar/file.socket#frag")
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}
	t.Logf("%#v\n", url)
	if url.Scheme != "foo" {
		t.Fatal("wrong scheme")
	}
	if url.Host != "C:/bar/file.socket" {
		t.Fatal("wrong host")
	}
	if url.Path != "" {
		t.Fatal("wrong path")
	}
	if url.Fragment != "frag" {
		t.Fatal("wrong fragment")
	}
	url, err = ParseWithSocket("mysql://cbm:cbm369@unix(/var/run/mysqld/mysqld.sock)/cbm#frag")
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}
	t.Logf("%#v\n", url)
	if url.Scheme != "mysql" {
		t.Fatal("wrong scheme")
	}
	if url.Host != "unix(/var/run/mysqld/mysqld.sock)" {
		t.Fatal("wrong host")
	}
	if url.Path != "/cbm" {
		t.Fatal("wrong path")
	}
	if url.Fragment != "frag" {
		t.Fatal("wrong fragment")
	}
	url, err = ParseWithSocket("ldap://x:x@tcp(127.0.0.1:389)/ou=People,dc=fcavani,dc=com#frag")
	if err != nil {
		t.Fatal(e.Trace(e.Forward(err)))
	}
	t.Logf("%#v\n", url)
	if url.Scheme != "ldap" {
		t.Fatal("wrong scheme")
	}
	if url.Host != "tcp(127.0.0.1:389)" {
		t.Fatal("wrong host")
	}
	if url.Path != "/ou=People,dc=fcavani,dc=com" {
		t.Fatal("wrong path")
	}
	if url.Fragment != "frag" {
		t.Fatal("wrong fragment")
	}
}

type onlyhost struct {
	rawurl string
	host   string
}

var onlyhosttests []onlyhost = []onlyhost{
	{"http://www.google.com/", "www.google.com"},
	{"http://www.google.com:666/", "www.google.com"},
	{"http://www.google.com/blá/blé/?as=234", "www.google.com"},
	{"http://localhost", "localhost"},
	{"http://localhost:123", "localhost"},
	{"http://192.168.0.1:123/?q=query", "192.168.0.1"},
	{"http://[2001:db8:1f70::999:de8:7648:6e8]", "2001:db8:1f70::999:de8:7648:6e8"},
	{"http://[2001:db8:1f70::999:de8:7648:6e8]:100/?q=query", "2001:db8:1f70::999:de8:7648:6e8"},
}

func TestOnlyHost(t *testing.T) {
	for i, test := range onlyhosttests {
		host, err := OnlyHost(test.rawurl)
		if err != nil {
			t.Fatal(e.Trace(e.Forward(err)))
		}
		if host != test.host {
			t.Fatal("host doesn't match", i, host)
		}
	}
}
