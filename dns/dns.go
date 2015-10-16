// Copyright 2015 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Resolve host name stuff.
package dns

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/fcavani/e"
	utilNet "github.com/fcavani/util/net"
	utilUrl "github.com/fcavani/util/net/url"
	"github.com/miekg/dns"
)

const ErrHostNotResolved = "host name not resolved"

var lookuphost func(host string) (addrs []string, err error) = LookupHost

func SetLookupHostFunction(f func(host string) (addrs []string, err error)) {
	lookuphost = f
}

var Timeout = 60 //Seconds
var ConfigurationFile = "/etc/resolv.conf"
var DialTimeout = 30 * time.Second
var ReadTimeout = 15 * time.Second
var WriteTimeout = 15 * time.Second

func LookupIp(ip string) (host string, err error) {
	if !utilNet.IsValidIpv4(ip) && !utilNet.IsValidIpv6(ip) {
		return "", e.New("not a valid ip address")
	}
	config, err := dns.ClientConfigFromFile(ConfigurationFile)
	if err != nil {
		return "", e.Forward(err)
	}
	config.Timeout = Timeout

	c := new(dns.Client)
	c.DialTimeout = DialTimeout
	c.ReadTimeout = ReadTimeout
	c.WriteTimeout = WriteTimeout
	m := new(dns.Msg)
	rev, err := dns.ReverseAddr(ip)
	if err != nil {
		return "", e.Forward(err)
	}
	m.SetQuestion(rev, dns.TypePTR)
	var r *dns.Msg
	for i := 0; i < len(config.Servers); i++ {
		r, _, err = c.Exchange(m, config.Servers[i]+":"+config.Port)
		if err != nil {
			continue
		}
		err = nil
	}
	if err != nil {
		return "", e.Forward(err)
	}
	if r.Rcode != dns.RcodeSuccess {
		return "", e.New("can't resolve %v", ip)
	}

	for _, a := range r.Answer {
		if ptr, ok := a.(*dns.PTR); ok {
			return strings.TrimSuffix(ptr.Ptr, "."), nil
		}
	}
	return "", e.New("no ptr available")
}

func LookupHost(host string) (addrs []string, err error) {
	if utilNet.IsValidIpv4(host) || utilNet.IsValidIpv6(host) {
		return []string{host}, nil
	}
	if host == "localhost" {
		return []string{"127.0.0.1", "::1"}, nil
	}
	config, err := dns.ClientConfigFromFile(ConfigurationFile)
	if err != nil {
		return nil, e.Forward(err)
	}
	config.Timeout = Timeout

	c := new(dns.Client)
	c.DialTimeout = DialTimeout
	c.ReadTimeout = ReadTimeout
	c.WriteTimeout = WriteTimeout
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(host), dns.TypeA)
	var r *dns.Msg
	for i := 0; i < len(config.Servers); i++ {
		r, _, err = c.Exchange(m, config.Servers[i]+":"+config.Port)
		if err != nil {
			continue
		}
		err = nil
	}
	if err != nil {
		return nil, e.Forward(err)
	}
	if r.Rcode != dns.RcodeSuccess {
		return nil, e.New("can't resolve %v", host)
	}

	addrs = make([]string, 0, 10)
	for _, a := range r.Answer {
		if addr, ok := a.(*dns.A); ok {
			addrs = append(addrs, addr.A.String())
		}
	}

	m.SetQuestion(dns.Fqdn(host), dns.TypeAAAA)
	for i := 0; i < len(config.Servers); i++ {
		r, _, err = c.Exchange(m, config.Servers[0]+":"+config.Port)
		if err != nil {
			continue
		}
		err = nil
	}
	if err != nil {
		return nil, e.Forward(err)
	}
	if r.Rcode != dns.RcodeSuccess {
		return nil, e.New("no success")
	}

	for _, a := range r.Answer {
		if addr, ok := a.(*dns.AAAA); ok {
			addrs = append(addrs, addr.AAAA.String())
		}
	}
	return
}

// Resolve simple resolver one host name to one ip
func Resolve(h string) (out string, err error) {
	host, port, err := utilNet.SplitHostPort(h)
	if err != nil && !e.Equal(err, utilNet.ErrCantFindPort) {
		return "", e.Forward(err)
	}

	addrs, err := lookuphost(host)
	if err != nil {
		return "", e.Forward(err)
	}
	if len(addrs) <= 0 {
		return "", e.New(ErrHostNotResolved)
	}

	if strings.Contains(addrs[0], ":") {
		out = "[" + addrs[0] + "]"
	} else {
		out = addrs[0]
	}
	if port != "" {
		out += ":" + port
	}
	return
}

// ResolveUrl replaces the host name with the ip address. Supports ipv4 and ipv6.
// If use in the place of host a path or a scheme for sockets, file or unix,
// ResolveUrl will only copy the url.
func ResolveUrl(url *url.URL) (*url.URL, error) {
	if url.Scheme == "file" || url.Scheme == "socket" || url.Scheme == "unix" {
		return utilUrl.Copy(url), nil
	}
	if len(url.Host) > 0 && url.Host[0] == '/' {
		return utilUrl.Copy(url), nil
	}
	if len(url.Host) >= 3 && url.Host[1] == ':' && url.Host[2] == '/' {
		return utilUrl.Copy(url), nil
	}
	r, err := regexp.Compile(`.*\(.*\)`)
	if err != nil {
		return nil, e.New(err)
	}
	mysqlNotation := r.FindAllString(url.Host, 1)
	if len(mysqlNotation) >= 1 {
		return utilUrl.Copy(url), nil
	}

	out := utilUrl.Copy(url)

	host, err := Resolve(url.Host)
	if err != nil {
		return nil, e.Forward(err)
	}
	out.Host = host
	return out, nil
}
