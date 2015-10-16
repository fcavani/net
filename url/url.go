// Copyright 2015 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Url package have util functions to deal with url.
package url

import (
	"math"
	"net/url"
	"regexp"
	"strings"

	"github.com/fcavani/e"
	utilNet "github.com/fcavani/util/net"
)

// OnlyHost returns the FQDN or the ip of the host in the url.
func OnlyHost(rawurl string) (string, error) {
	parsed, err := url.Parse(rawurl)
	if err != nil {
		return "", e.Forward(err)
	}
	host, _, err := utilNet.SplitHostPort(parsed.Host)
	if err != nil && !e.Equal(err, utilNet.ErrCantFindPort) {
		return "", e.Forward(err)
	}
	return host, nil
}

func Copy(in *url.URL) (out *url.URL) {
	out = new(url.URL)
	out.Scheme = in.Scheme
	out.Opaque = in.Opaque
	if in.User != nil {
		usr := in.User.Username()
		pass, ok := in.User.Password()
		if ok {
			out.User = url.UserPassword(usr, pass)
		} else {
			out.User = url.User(usr)
		}
	}
	out.Host = in.Host
	out.Path = in.Path
	out.RawQuery = in.RawQuery
	out.Fragment = in.Fragment
	return
}

var regSocket *regexp.Regexp
var regUnix *regexp.Regexp

func init() {
	regSocket = regexp.MustCompile(`([A-Za-z0-9]*)\((.*)\)`)
	regUnix = regexp.MustCompile(`.*\(.*\)`)
}

//"mysql://root:pass@unix(/var/run/mysql.socket)/db"

// Socket split in socket type and path the socket write
// in the same notation of github.com/go-sql-driver/mysql.
// Like this: unix(/var/run/mysql.socket)
func Socket(host string) (string, string, error) {
	sub := regSocket.FindStringSubmatch(host)
	if len(sub) != 3 {
		return "", "", e.New("path not found")
	}
	return sub[1], sub[2], nil
}

func pathInHost(u *url.URL) {
	if u.Host == "" && u.Path != "" {
		u.Host = u.Path
		u.Path = ""
	}
	if u.Host[len(u.Host)-1] == ':' && u.Path != "" {
		u.Host += u.Path
		u.Path = ""
	}
}

// ParseWithSocket parses url like this: mysql://root:pass@unix(/var/run/mysql.socket)/db
// and normal urls.
func ParseWithSocket(url_ string) (*url.URL, error) {
	u := new(url.URL)
	s := strings.SplitN(url_, "://", 2)
	if len(s) != 2 {
		return nil, e.New("invalid url")
	}
	u.Scheme = s[0]
	rest := ""
	s = strings.SplitN(s[1], "@", 2)
	if len(s) == 1 {
		rest = s[0]
	} else if len(s) == 2 {
		userpass := strings.SplitN(s[0], ":", 2)
		if len(userpass) == 1 {
			u.User = url.User(userpass[0])
		} else if len(userpass) == 2 {
			pass, err := url.QueryUnescape(userpass[1])
			if err != nil {
				return nil, e.New(err)
			}
			u.User = url.UserPassword(userpass[0], pass)
		} else {
			return nil, e.New("invalid user password")
		}
		rest = s[1]
	} else {
		return nil, e.New("invalid user string")
	}

	unix := regUnix.FindAllString(rest, 1)
	if len(unix) == 1 {
		u.Host = unix[0]
		rest = strings.TrimSpace(regUnix.ReplaceAllLiteralString(rest, ""))
		q := strings.Index(rest, "?")
		f := strings.Index(rest, "#")
		pend := f
		if q > f {
			pend = q
		}
		i := strings.Index(rest, "/")
		if i != -1 && pend != -1 {
			u.Path = rest[i:pend]
			rest = rest[pend:]
		} else if i == -1 && pend != -1 {
			rest = rest[pend:]
		} else if i != -1 && pend == -1 {
			u.Path = rest[i:]
			pathInHost(u)
			return u, nil
		} else if i == -1 && pend == -1 {
			pathInHost(u)
			return u, nil
		}
	} else if len(unix) == 0 {
		q := strings.Index(rest, "?")
		f := strings.Index(rest, "#")
		pend := f
		ff := f
		if f == -1 {
			ff = math.MaxInt64
		}
		if q < ff && q >= 0 {
			pend = q
		}
		i := strings.Index(rest, "/")
		if i != -1 && pend != -1 {
			u.Host = rest[:i]
			u.Path = rest[i:pend]
			rest = rest[pend:]
		} else if i == -1 && pend != -1 {
			u.Host = rest[:pend]
			rest = rest[pend:]
		} else if i != -1 && pend == -1 {
			u.Host = rest[:i]
			u.Path = rest[i:]
			pathInHost(u)
			return u, nil
		} else if i == -1 && pend == -1 {
			u.Host = rest
			pathInHost(u)
			return u, nil
		}
	} else {
		return nil, e.New("socket address is invalid")
	}

	pathInHost(u)

	q := strings.Index(rest, "?")
	f := strings.Index(rest, "#")

	if q+1 >= len(rest) {
		return nil, e.New("error parsing query")
	}
	if f+1 >= len(rest) {
		return nil, e.New("error parsing fragment")
	}

	if q != -1 && f != -1 && q <= f {
		u.RawQuery = rest[q+1 : f]
		u.Fragment = rest[f+1:]
	} else if q != -1 && f == -1 {
		u.RawQuery = rest[q+1:]
	} else if q == -1 && f != -1 {
		u.Fragment = rest[f+1:]
	} else if q == -1 && f == -1 {
		return u, nil
	} else {
		return nil, e.New("error parsing query and fragment %v, %v", q, f)
	}
	return u, nil
}
