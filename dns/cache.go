// Copyright 2015 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dns

import (
	"sync"
	"time"

	"github.com/fcavani/e"
	log "github.com/fcavani/slog"
)

// Time between the cleanup of the cache. Old entries are removed. In seconds.
var Sleep = 3600 * time.Second

// Time of life of one entry, in seconds. Is query is a hit this time is reseted.
var DefaultExpire = 24 * 60 * 60 * time.Second

const ErrNotFound = "entry not found"
const ErrDupEntry = "duplicated entry"
const ErrIterStop = "iter stop"
const ErrServFail = "serv fail"

type Host struct {
	Addrs    []string
	ServFail bool
	Expire   time.Time
}

func (h *Host) ReturnPtr() (string, error) {
	if h.ServFail {
		return "", e.New(ErrServFail)
	}
	return h.Addrs[0], nil
}

func (h *Host) ReturnAddrs() ([]string, error) {
	if h.ServFail {
		return nil, e.New(ErrServFail)
	}
	return h.Addrs, nil
}

type Storer interface {
	Get(key string) (*Host, error)
	Put(key string, data *Host) error
	Del(key string) error
	Iter(f func(key string, data *Host) error) error
}

type Mem struct {
	m   map[string]*Host
	lck sync.RWMutex
}

func NewMem() Storer {
	return &Mem{
		m: make(map[string]*Host),
	}
}

func (m *Mem) Get(key string) (*Host, error) {
	m.lck.RLock()
	defer m.lck.RUnlock()
	data, found := m.m[key]
	if !found {
		return nil, e.New(ErrNotFound)
	}
	return data, nil
}

func (m *Mem) Put(key string, data *Host) error {
	m.lck.Lock()
	defer m.lck.Unlock()
	_, found := m.m[key]
	if found {
		return e.New(ErrDupEntry)
	}
	m.m[key] = data
	return nil
}

func (m *Mem) Del(key string) error {
	m.lck.Lock()
	defer m.lck.Unlock()
	_, found := m.m[key]
	if !found {
		return e.New(ErrNotFound)
	}
	delete(m.m, key)
	return nil
}

func (m *Mem) Iter(f func(key string, data *Host) error) error {
	var err error
	for k, v := range m.m {
		err = f(k, v)
		if e.Equal(err, ErrIterStop) {
			return nil
		} else if err != nil {
			return e.Forward(err)
		}
	}
	return nil
}

type Cacher interface {
	Get(key string) *Host
	PutAddrs(key string, ips []string) error
	PutPtr(key, ptr string) error
	PutServFail(key string) error
	Close() error
}

type Cache struct {
	s      Storer
	d      time.Duration
	chstop chan chan struct{}
}

func NewCache(s Storer, d, cleanup time.Duration) Cacher {
	c := &Cache{
		s:      s,
		d:      d,
		chstop: make(chan chan struct{}),
	}
	go func() {
		for {
			select {
			case <-time.After(cleanup):
				err := c.s.Iter(func(key string, data *Host) error {
					if data.Expire.Before(time.Now()) {
						er := c.s.Del(key)
						if er != nil {
							return e.Forward(er)
						}
					}
					return nil
				})
				if err != nil {
					log.DebugLevel().Tag("dns", "cache", "cleanup").Println(err)
				}
			case ch := <-c.chstop:
				ch <- struct{}{}
				return
			}
		}
	}()
	return c
}

func (c *Cache) Close() error {
	ch := make(chan struct{})
	c.chstop <- ch
	<-ch
	return nil
}

func (c *Cache) Get(key string) *Host {
	h, err := c.s.Get(key)
	if err != nil {
		return nil
	}
	return h
}

func (c *Cache) PutAddrs(key string, ips []string) error {
	c.s.Del(key)
	h := &Host{
		Addrs:  ips,
		Expire: time.Now().Add(c.d),
	}
	return e.Forward(c.s.Put(key, h))
}

func (c *Cache) PutPtr(key, ptr string) error {
	c.s.Del(key)
	h := &Host{
		Addrs:  []string{ptr},
		Expire: time.Now().Add(c.d),
	}
	return e.Forward(c.s.Put(key, h))
}

func (c *Cache) PutServFail(key string) error {
	c.s.Del(key)
	h := &Host{
		Addrs:    []string{""},
		ServFail: true,
		Expire:   time.Now().Add(c.d),
	}
	return e.Forward(c.s.Put(key, h))
}
