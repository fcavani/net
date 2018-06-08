// Copyright 2015 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !windows

// Resolve host name stuff.
package dns

import (
	"context"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/fcavani/e"
	utilNet "github.com/fcavani/net"
	utilUrl "github.com/fcavani/net/url"
	log "github.com/fcavani/slog"
	mdns "github.com/fcavani/zeroconf"
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
var resolver *mdns.Resolver

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

	// lo, err := net.InterfaceByName("lo0")
	// if err != nil {
	// 	log.Tag("dns", "config", "interfaces").Errorln("Failed get lo interface:", err)
	// 	return
	// }
	// en, err := net.InterfaceByName("en1")
	// if err != nil {
	// 	log.Tag("dns", "config", "interfaces").Errorln("Failed get en interface:", err)
	// 	return
	// }
	// ifaces := mdns.SelectIfaces([]net.Interface{*lo})

	resolver, err = mdns.NewResolver(nil)
	if err != nil {
		log.Tag("dns", "config").Errorln("Failed to initialize resolver:", err)
	}
}

func MulticastDNSResolverConfig(ifs []net.Interface) error {
	var err error

	if len(ifs) == 0 {
		return e.New("invalid interfaces")
	}

	ifaces := mdns.SelectIfaces(ifs)

	resolver, err = mdns.NewResolver(ifaces)
	if err != nil {
		return e.New(err)
	}

	return nil
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

	addrs, err = lookupHost(host, true, config)
	if err != nil {
		return nil, e.Forward(err)
	}
	return addrs, nil
}

func LookupHostNoCache(host string) (addrs []string, err error) {
	start := time.Now()
	defer func() {
		log.DebugLevel().Tag("dns").Printf("LookupHostNoCache %v took: %v", host, time.Since(start))
	}()

	addrs, err = lookupHost(host, false, config)
	if err != nil {
		return nil, e.Forward(err)
	}
	return addrs, nil
}

func LookupHostWithServers(host string, servers []string, attempts, timeout int) (addrs []string, err error) {
	start := time.Now()
	defer func() {
		log.DebugLevel().Tag("dns").Printf("LookupHostWithServers %v took: %v", host, time.Since(start))
	}()

	cfg := new(dns.ClientConfig)
	cfg.Attempts = attempts
	cfg.Ndots = config.Ndots
	cfg.Port = config.Port
	cfg.Servers = servers
	cfg.Timeout = timeout

	addrs, err = lookupHost(host, true, cfg)
	if err != nil {
		return nil, e.Forward(err)
	}
	return addrs, nil
}

func lookupHost(host string, useCache bool, config *dns.ClientConfig) (addrs []string, err error) {
	addrs, err = queryDNS(host, useCache, config)
	if err != nil && !e.Equal(err, ErrCantResolve) {
		return nil, e.Forward(err)
	}
	if len(addrs) > 0 {
		return addrs, nil
	}
	addrs, err = querymDNS(host, useCache)
	if err != nil {
		return nil, e.Forward(err)
	}
	return addrs, nil
}

const ErrCantResolve = "can't resolve the address"

func queryDNS(host string, useCache bool, config *dns.ClientConfig) (addrs []string, err error) {
	start := time.Now()
	defer func() {
		log.DebugLevel().Tag("dns").Printf("lookupHost %v took: %v", host, time.Since(start))
	}()

	if useCache {
		h := cache.Get(host)
		if h != nil {
			addrs, err = h.ReturnAddrs()
			if err == nil {
				return addrs, nil
			} else if err != nil && !e.Equal(err, ErrServFail) {
				return nil, e.Forward(err)
			}
		}
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
			return
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
		return nil, e.New(ErrCantResolve)
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
		if len(addrs) > 0 {
			return addrs, nil
		}
		return nil, e.New(ErrCantResolve)
	}

	for _, a := range r.Answer {
		if addr, ok := a.(*dns.AAAA); ok {
			addrs = append(addrs, addr.AAAA.String())
		}
	}

	return
}

func querymDNS(host string, useCache bool) (addrs []string, err error) {
	start := time.Now()
	defer func() {
		log.DebugLevel().Tag("dns", "mdns").Printf("lookupHost %v took: %v", host, time.Since(start))
	}()

	addrs = make([]string, 0, 10)

	if useCache {
		h := cache.Get(host)
		if h != nil {
			addrs, err = h.ReturnAddrs()
			if err == nil {
				return addrs, nil
			} else if err != nil && !e.Equal(err, ErrServFail) {
				return nil, e.Forward(err)
			}
		}
	}

	defer func() {
		if len(addrs) == 0 {
			cache.PutServFail(host)
			return
		}
		cache.PutAddrs(host, addrs)
	}()

	entries := make(chan *mdns.ServiceEntry, 10)

	go func(results <-chan *mdns.ServiceEntry) {
		for entry := range results {
			log.DebugLevel().Tag("dns", "mdns").Println("mDNS entry:", entry)
			for _, ip4 := range entry.AddrIPv4 {
				addrs = append(addrs, ip4.String())
			}
			for _, ip6 := range entry.AddrIPv6 {
				addrs = append(addrs, ip6.String())
			}
		}
		log.DebugLevel().Tag("dns", "mdns").Println("No more entries.")
	}(entries)

	nodomain := strings.TrimSuffix(host, ".local")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = resolver.Browse(ctx, nodomain, "local.", entries)
	if err != nil {
		return nil, e.Push(err, "failed to browse")
	}

	<-ctx.Done()

	if len(addrs) == 0 {
		return nil, e.New("can't resolve %v", host)
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
	if len(addrs) == 0 {
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
