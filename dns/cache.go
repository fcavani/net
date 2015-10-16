// Copyright 2015 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dns

import (
	"sync"
	"time"

	"github.com/fcavani/e"
)

// Number of entries in the cache.
var CacheSize = 100

// Time between the cleanup of the cache. Old entries are removed. In seconds.
var Sleep int = 3600

// Time of life of one entry, in seconds. Is query is a hit this time is reseted.
var DefaultExpire = 10 * 24 * 60 * 60

type host struct {
	Addrs  []string
	Expire time.Time
}

var cache map[string]*host
var mutex *sync.RWMutex
var hit uint64
var miss uint64

func init() {
	cache = make(map[string]*host, CacheSize)
	mutex = new(sync.RWMutex)
	go func() {
		for {
			time.Sleep(time.Duration(Sleep) * time.Second)
			mutex.Lock()
			for hostname, entry := range cache {
				if entry.Expire.Before(time.Now()) {
					delete(cache, hostname)
				}
			}
			mutex.Unlock()
		}
	}()
}

// Hists retuns the number of hits.
func Hits() uint64 {
	return hit
}

// Miss returns the number of cache miss.
func Miss() uint64 {
	return miss
}

const ErrHostFound = "host found"

func addToCache(h string, addrs []string) {
	mutex.Lock()
	defer mutex.Unlock()
	cache[h] = &host{
		Addrs:  addrs,
		Expire: time.Now().Add(time.Duration(DefaultExpire) * time.Second),
	}
}

func lookupInCache(h string) []string {
	mutex.RLock()
	defer mutex.RUnlock()
	entry, found := cache[h]
	if !found {
		miss++
		return nil
	}
	hit++
	entry.Expire = time.Now().Add(time.Duration(DefaultExpire) * time.Second)
	return entry.Addrs
}

// LookupHostCache do a simple lookup but if the lookup fail, its returns
// the hosts in cache. If look is ok its store the hosts in cache or reset
// the expire time for that host.
func LookupHostCache(host string) ([]string, error) {
	addrs, err := LookupHost(host)
	if err != nil {
		a := lookupInCache(host)
		if a == nil {
			return nil, e.Forward(err)
		}
		return a, nil
	}
	addToCache(host, addrs)
	return addrs, nil
}
