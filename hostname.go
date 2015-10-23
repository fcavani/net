// Copyright 2015 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"encoding/json"
	"net/http"

	"github.com/fcavani/e"
)

type Addr struct {
	Ip      string   `json:"ip"`
	Name    string   `json:"name,omitempty"`
	Proxies []string `json:"proxies,omitempty"`
}

func HostnameFqdn() (hostname string, err error) {
	resp, err := http.Get("https://fcavani.com/ip")
	if err != nil {
		return "", e.Forward(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", e.New("protocol fail")
	}
	dec := json.NewDecoder(resp.Body)
	var addr Addr
	err = dec.Decode(&addr)
	if err != nil {
		return "", e.Forward(err)
	}
	return addr.Name, nil
}
