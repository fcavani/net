// Copyright 2015 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Network util functions.
package net

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/fcavani/e"
	"github.com/fcavani/util/text"
)

const ErrCantGetIp = "can't get remote ip"
const ErrCantSplitHostPort = "can't split host port"
const ErrCantFindHost = "can't find the host"
const ErrCantFindPort = "can't find the port number"

type RegularExp string

func (r RegularExp) Clean() string {
	buf := make([]byte, len(Ipv6Regex))
	j := 0
	for _, re := range r {
		if re != '\n' && re != '\t' && re != ' ' {
			if re > 255 { // rune is int32
				panic("no multibyte characters")
			}
			buf[j] = byte(re)
			j++
		}
	}
	buf = buf[:j]
	return string(buf)
}

//Regex from https://gist.github.com/syzdek/6086792
var Ipv6Regex RegularExp = `(
	(([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))|
	(::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))|
	(fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,})|
	(:((:[0-9a-fA-F]{1,4}){1,7}|:))|
	(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}) |
	([0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6}))|
	([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|
	(([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4})|
	(([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3})|
	(([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2})|
	(([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4})|
	(([0-9a-fA-F]{1,4}:){1,7}:)
)`

//Regex from https://gist.github.com/syzdek/6086792
var Ipv4Regex RegularExp = `(
	(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}
	(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]
)`

var reCompIpv4 *regexp.Regexp
var reCompIpv6 *regexp.Regexp
var reCompIpv6Port *regexp.Regexp

func init() {
	ipv6re := Ipv6Regex.Clean()
	ipv6portre := `\[` + ipv6re + `\]\:([0-9]*)`

	reCompIpv4 = regexp.MustCompile(Ipv4Regex.Clean())
	reCompIpv6 = regexp.MustCompile(ipv6re)
	reCompIpv6Port = regexp.MustCompile(ipv6portre)
}

func IsValidIpv4(ip string) bool {
	if ip != reCompIpv4.FindString(ip) {
		return false
	}
	return true
}

func IsValidIpv6(ip string) bool {
	ip = strings.TrimSuffix(strings.TrimPrefix(ip, "["), "]")

	if ip != reCompIpv6.FindString(ip) {
		return false
	}

	return true
}

// SplitHostPort splits a string with a ipv6, ipv4 or hostname with a port number.
func SplitHostPort(hp string) (host, port string, err error) {
	if len(hp) == 0 {
		return "", "", e.New("invalid host length")
	}
	if hp[0] == '[' {
		// ipv6 - [2001:db8:1f70::999:de8:7648:6e8]:100
		x := reCompIpv6Port.FindAllStringSubmatch(hp, -1)
		if len(x) == 0 {
			if IsValidIpv6(hp) {
				host = strings.TrimSuffix(strings.TrimPrefix(hp, "["), "]")
				port = ""
			} else {
				return "", "", e.New(ErrCantGetIp)
			}
		} else {
			if len(x[0]) >= 3 {
				host = x[0][1]
				port = x[0][len(x[0])-1] //Last is the port
			} else if len(x[0]) >= 2 {
				host = x[0][1]
				port = ""
			} else {
				return "", "", e.New(ErrCantGetIp)
			}
		}
	} else {
		//ip4 and host name
		ipport := strings.SplitN(hp, ":", 2)
		if len(ipport) == 1 {
			host = ipport[0]
			port = ""
		} else if len(ipport) == 2 {
			host = ipport[0]
			port = ipport[1]
		} else {
			return "", "", e.New(ErrCantSplitHostPort)
		}
		if !IsValidIpv4(host) {
			err := text.CheckDomain(host)
			if err != nil {
				return "", "", e.New("invalid domain name or ipv4")
			}
		}
	}
	if host == "" {
		return "", "", e.New(ErrCantFindHost)
	}
	if port == "" {
		return host, "", e.New(ErrCantFindPort)
	}
	_, err = strconv.ParseUint(port, 10, 16)
	if err != nil {
		return "", "", e.Push(e.New(err), "invalid port number")
	}
	host = strings.ToLower(host)
	return
}

func IpPort(ip, port string) (addr string, err error) {
	if IsValidIpv4(ip) {
		addr = ip + ":" + port
	} else if IsValidIpv6(ip) {
		addr = "[" + ip + "]:" + port
	} else {
		return "", e.New("invalid ip adderess")
	}
	return
}
