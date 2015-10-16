// Copyright 2015 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"testing"

	"github.com/fcavani/e"
)

type testipstruct struct {
	ip    string
	valid bool
}

var testipv4 []testipstruct = []testipstruct{
	{"127.0.0.1", true},
	{"10.0.0.1", true},
	{"192.168.1.1", true},
	{"1.2.3.4", true},
	{"255.255.255.255", true},
	{"300.2.2.2", false},
	{"ab.cb.3.123", false},
	{"catoto", false},
	{"192.168.10.1a", false},
}

func TestIsValidIpv4(t *testing.T) {
	for i, ip := range testipv4 {
		ok := IsValidIpv4(ip.ip)
		if ok != ip.valid {
			t.Fatal("failed for", i)
		}
	}
}

var testipv6 []testipstruct = []testipstruct{
	{"1:2:3:4:5:6:7:8", true},
	{"1::", true},
	{"1:2:3:4:5:6:7::", true},
	{"1::8", true},
	{"1:2:3:4:5:6::8", true},
	{"1::7:8", true},
	{"1:2:3:4:5::7:8", true},
	{"1:2:3:4:5::8", true},
	{"1::6:7:8", true},
	{"1:2:3:4::6:7:8", true},
	{"1:2:3:4::8", true},
	{"1::5:6:7:8", true},
	{"1:2:3::5:6:7:8", true},
	{"1:2:3::8", true},
	{"1::4:5:6:7:8", true},
	{"1:2::4:5:6:7:8", true},
	{"1:2::8", true},
	{"1::3:4:5:6:7:8", true},
	{"1::3:4:5:6:7:8", true},
	{"1::8", true},
	{"::2:3:4:5:6:7:8", true},
	{"::2:3:4:5:6:7:8", true},
	{"::8", true},
	{"::", true},
	{"fe80::7:8%eth0", true},
	{"fe80::7:8%1", true},
	{"::255.255.255.255", true},
	{"::ffff:255.255.255.255", true},
	{"::ffff:0:255.255.255.255", true},
	{"2001:db8:3:4::192.0.2.33", true},
	{"64:ff9b::192.0.2.33", true},
	{"2001:db8:1f70::999:de8:7648:6e8", true},
	{"catodos", false},
	{"192.168.1.1", false},
	{"2001:db8:1f70::999:de8:7648:6e8z", false},
	{"2001:db8:1f70:0:999:de8:7648:6e8", true},
	{"2001:db8:1f70:x:999:de8:7648:6e8", false},
}

func TestIsValidIpv6(t *testing.T) {
	for i, ip := range testipv6 {
		t.Log(i, ip.ip)
		ok := IsValidIpv6(ip.ip)
		if ok != ip.valid {
			t.Fatal("failed for", i, ip.ip)
		}
	}
}

type testHostPortStruct struct {
	hostport string
	host     string
	port     string
	fail     bool
}

var testhp []testHostPortStruct = []testHostPortStruct{
	{"[2001:db8:1f70::999:de8:7648:6e8]:100", "2001:db8:1f70::999:de8:7648:6e8", "100", false},
	{"127.0.0.1:169", "127.0.0.1", "169", false},
	{"www.isp.net:8080", "www.isp.net", "8080", false},
	{"www.isp.net", "www.isp.net", "", true},
	{"[2001:db8:1f70::999:de8:7648:6e8]", "2001:db8:1f70::999:de8:7648:6e8", "", true},
	{"www.isp.net:", "www.isp.net", "", true},
	{"[2001:db8:1f70::999:de8:7648:6e8]:", "2001:db8:1f70::999:de8:7648:6e8", "", true},
	{"www.isp.net:65536", "", "", true},
	{"www.isp*.net:", "", "", true},
	{"192.168.10.15:48405", "192.168.10.15", "48405", false},
}

func TestSplitHostPort(t *testing.T) {
	for i, thp := range testhp {
		host, port, err := SplitHostPort(thp.hostport)
		if err != nil && !thp.fail {
			t.Fatal(i, e.Trace(e.Forward(err)))
		} else if err == nil && thp.fail {
			t.Fatal(i, "doesn't failed", host, port)
		}
		if host != thp.host {
			t.Fatal("wrong host", i, host)
		}
		if port != thp.port {
			t.Fatal("wrong port", i, port)
		}
	}
}
