// Copyright 2015 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !windows

// Resolve host name stuff.
package dns

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/fcavani/e"
	"github.com/fcavani/log"
	utilNet "github.com/fcavani/net"
	utilUrl "github.com/fcavani/net/url"
	"github.com/miekg/dns"
)

const ErrHostNotResolved = "host name not resolved"

var Timeout = 5 //Seconds
var ConfigurationFile = "/etc/resolv.conf"
var DialTimeout = 10 * time.Second
var ReadTimeout = 500 * time.Millisecond
var WriteTimeout = 500 * time.Millisecond

var cache Cacher
var config *dns.ClientConfig

func init() {
	var err error
	s := NewMem()
	cache = NewCache(s, DefaultExpire, Sleep)
	cache.PutAddrs("localhost", []string{"127.0.0.1", "::1"})
	cache.PutPtr("127.0.0.1", "localhost")
	cache.PutPtr("::1", "localhost")
	config, err = dns.ClientConfigFromFile(ConfigurationFile)
	if err != nil {
		log.ErrorLevel().Tag("dns", "config").Println("config failed:", err)
		config = new(dns.ClientConfig)
		config.Attempts = 3
		config.Ndots = 1
		config.Port = "53"
		config.Servers = []string{"8.8.8.8", "8.8.4.4"}
	}
	config.Timeout = Timeout
}

func LookupIp(ip string) (host string, err error) {
	start := time.Now()
	defer func() {
		log.DebugLevel().Tag("dns").Printf("LookupIp %v took: %v", ip, time.Since(start))
	}()

	h := cache.Get(ip)
	if h != nil {
		return h.ReturnPtr()
	}

	if ip == "127.0.0.1" || ip == "::1" {
		return "localhost", nil
	}

	if !utilNet.IsValidIpv4(ip) && !utilNet.IsValidIpv6(ip) {
		return "", e.New("not a valid ip address")
	}

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
			log.DebugLevel().Tag("dns").Printf("Lookup %v ptr fail: %v", ip, err)
			continue
		}
		err = nil
		break
	}
	if err != nil {
		cache.PutServFail(ip)
		return "", e.Forward(err)
	}
	if r.Rcode != dns.RcodeSuccess {
		cache.PutServFail(ip)
		return "", e.New("can't resolve %v", ip)
	}

	for _, a := range r.Answer {
		if ptr, ok := a.(*dns.PTR); ok {
			ptraddr := strings.TrimSuffix(ptr.Ptr, ".")
			cache.PutPtr(ip, ptraddr)
			return ptraddr, nil
		}
	}
	cache.PutServFail(ip)
	return "", e.New("no ptr available")
}

func LookupHost(host string) (addrs []string, err error) {
	start := time.Now()
	defer func() {
		log.DebugLevel().Tag("dns").Printf("LookupHost %v took: %v", host, time.Since(start))
	}()

	addrs, err = lookupHost(host, config)
	if err != nil {
		return nil, e.Forward(err)
	}
	return
}

func LookupHostWithServers(host string, servers []string, attempts int, timeout time.Duration) (addrs []string, err error) {
	start := time.Now()
	defer func() {
		log.DebugLevel().Tag("dns").Printf("LookupHostWithServers %v took: %v", host, time.Since(start))
	}()

	addrs, err = lookupHost(host, config)
	if err != nil {
		return nil, e.Forward(err)
	}
	return
}

func lookupHost(host string, config *dns.ClientConfig) (addrs []string, err error) {
	start := time.Now()
	defer func() {
		log.DebugLevel().Tag("dns").Printf("lookupHost %v took: %v", host, time.Since(start))
	}()

	h := cache.Get(host)
	if h != nil {
		return h.ReturnAddrs()
	}

	if host == "localhost" {
		return []string{"127.0.0.1", "::1"}, nil
	}

	if utilNet.IsValidIpv4(host) || utilNet.IsValidIpv6(host) {
		return []string{host}, nil
	}

	defer func() {
		if len(addrs) == 0 {
			cache.PutServFail(host)
		}
		cache.PutAddrs(host, addrs)
	}()

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
			log.DebugLevel().Tag("dns").Printf("Lookup addrs A %v fail: %v", host, err)
			continue
		}
		err = nil
		break
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
			log.DebugLevel().Tag("dns").Printf("Lookup addrs AAAA %v fail: %v", host, err)
			continue
		}
		err = nil
		break
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
	start := time.Now()
	defer func() {
		log.DebugLevel().Tag("dns").Printf("Resolve %v took: %v", h, time.Since(start))
	}()

	host, port, err := utilNet.SplitHostPort(h)
	if err != nil && !e.Equal(err, utilNet.ErrCantFindPort) {
		return "", e.Forward(err)
	}

	addrs, err := LookupHost(host)
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

var regExpResolveUrl = regexp.MustCompile(`.*\(.*\)`)

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

	mysqlNotation := regExpResolveUrl.FindAllString(url.Host, 1)
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
