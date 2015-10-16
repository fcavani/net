// Copyright 2015 Felipe A. Cavani. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// Start date:		2012-03-08
// Last modification:	2012-

// Support for self-signed certificate in SendMail function
package smtp

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/smtp"
	"reflect"
	"strings"
	"time"

	"github.com/fcavani/e"
	utilNet "github.com/fcavani/util/net"
	"github.com/fcavani/util/net/dns"
)

// Command object executes conn.SetDeadline before every function calls.
type Command struct {
	Timeout time.Duration
	Conn    net.Conn
}

type Return func(args ...interface{})

// SetError finds a type that implements a error interface and set it to err.
func setError(retvals []reflect.Value, err error) {
	if err == nil {
		return
	}
	var i int
	var retval reflect.Value
	for i, retval = range retvals {
		if reflect.TypeOf(err).AssignableTo(retval.Type()) {
			if retval.CanSet() && retval.IsNil() {
				retval.Set(reflect.ValueOf(err))
			}
		}
	}
	if i == len(retvals) {
		panic("can't assign an error to none of the returned values")
	}
}

// ExecTimeout executes f with arguments args and return a function of type Return.
// Return must be called with the pointer to the return values.
func (c *Command) ExecTimeout(timeout time.Duration, f interface{}, args ...interface{}) Return {
	var retvals chan []reflect.Value = make(chan []reflect.Value)
	val := reflect.ValueOf(f)
	if val.Kind() != reflect.Func {
		panic("f is not a function")
	}
	t := val.Type()
	if t.NumIn() != len(args) {
		panic("invalid number of arguments")
	}
	a := make([]reflect.Value, len(args))
	for i, arg := range args {
		a[i] = reflect.ValueOf(arg)
		if !a[i].Type().AssignableTo(t.In(i)) {
			panic(fmt.Sprintf("invalid argument type, argument %v must be assignable to %v", a[i].Type(), t.In(i)))
		}
	}
	go func() {
		var err error
		if timeout != 0 {
			err = c.Conn.SetDeadline(time.Now().Add(timeout))
			if err != nil {
				log.Println("SetDeadline failed with error:", e.Trace(e.Forward(err)))
			}
		} else {
			err = c.Conn.SetDeadline(time.Time{})
			if err != nil {
				log.Println("SetDeadline failed with error:", e.Trace(e.Forward(err)))
			}
		}
		values := val.Call(a)
		setError(values, err)
		retvals <- values
	}()

	return func(args ...interface{}) {
		retvals := <-retvals
		if len(retvals) != len(args) {
			panic(fmt.Sprintf("the number of arguments (%v) in Returns must be equal to the number of return values in the function (%v)", len(args), len(retvals)))
		}
		for i, retval := range retvals {
			val := reflect.ValueOf(args[i])
			if val.Kind() != reflect.Ptr {
				panic("Returns arguments must be pointers")
			}
			if retval.Type() != val.Elem().Type() {
				panic("diferent kind")
			}
			val.Elem().Set(retval)
		}
	}
}

// Exec executes f with arguments args and return a function of type Return.
// Return must be called with the pointer to the return values.
func (c *Command) Exec(f interface{}, args ...interface{}) Return {
	return c.ExecTimeout(c.Timeout, f, args...)
}

// Generate a comma separated list of e-mails from a array of e-mails
func EmailsToString(mails []string) (s string) {
	for i, mail := range mails {
		if i > 0 {
			s += ", "
		}
		s += mail
	}
	return
}

// SendMail send a message to specific destination (to) using smtp server in addrs
// and a auth.
func SendMail(addr string, a smtp.Auth, from string, to []string, hello string, msg []byte, timeout time.Duration, insecureSkipVerify bool) error {
	serverName := addr
	port := ""
	serverName, port, err := net.SplitHostPort(addr)
	if err != nil {
		return e.Push(err, "invalid adderess")
	}

	if serverName == "" || port == "" {
		return e.New("addrs is invalid")
	}

	hosts, err := dns.LookupHostCache(serverName)
	if err != nil {
		return e.Forward(err)
	}
	if len(hosts) == 0 {
		return e.New("can't resolve the addr")
	}

	var conn net.Conn
	for _, host := range hosts {
		addr, err = utilNet.IpPort(host, port)
		if err != nil {
			err = e.Forward(err)
			continue
		}

		conn, err = net.DialTimeout("tcp", addr, timeout)
		if err != nil {
			err = e.Forward(err)
			continue
		}
	}
	if conn == nil {
		return err
	}

	command := &Command{
		Timeout: timeout,
		Conn:    conn,
	}

	var c *smtp.Client
	r := command.Exec(smtp.NewClient, conn, serverName)
	r(&c, &err)
	if err != nil {
		return e.Forward(err)
	}
	defer c.Close()

	if hello != "" {
		r = command.Exec(c.Hello, hello)
		r(&err)
		if err != nil {
			return e.Forward(err)
		}
	}

	if ok, _ := c.Extension("STARTTLS"); ok {
		r = command.Exec(c.StartTLS, &tls.Config{
			ServerName:         serverName,
			InsecureSkipVerify: insecureSkipVerify,
		})
		r(&err)
		if err != nil {
			return e.Forward(err)
		}
	}

	if a != nil {
		found, _ := c.Extension("AUTH")
		if found {
			r = command.Exec(c.Auth, a)
			r(&err)
			if err != nil {
				return e.Forward(err)
			}
		}
	}

	r = command.Exec(c.Mail, from)
	r(&err)
	if err != nil {
		return e.Forward(err)
	}

	for _, addr := range to {
		r = command.Exec(c.Rcpt, addr)
		r(&err)
		if err != nil {
			return e.New(err)
		}
	}

	var w io.WriteCloser
	r = command.ExecTimeout(0, c.Data)
	r(&w, &err)
	if err != nil {
		return e.New(err)
	}
	_, err = w.Write(msg)
	if err != nil {
		return e.New(err)
	}
	err = w.Close()
	if err != nil {
		return e.New(err)
	}

	r = command.Exec(c.Quit)
	r(&err)
	if err != nil {
		return e.New(err)
	}
	return nil
}

// TestSMTP tests if can connect with the server and send some commands.
func TestSMTP(addr string, a smtp.Auth, hello string, timeout time.Duration, insecureSkipVerify bool) error {
	serverName := addr
	port := ""
	s := strings.SplitN(addr, ":", 2)
	if len(s) >= 2 {
		serverName = s[0]
		port = s[1]
	}

	if serverName == "" || port == "" {
		return e.New("addrs is invalid")
	}

	hosts, err := dns.LookupHostCache(serverName)
	if err != nil {
		return e.Forward(err)
	}
	if len(hosts) == 0 {
		return e.New("can't resolve the addr")
	}

	conn, err := net.DialTimeout("tcp", hosts[0]+":"+port, timeout)
	if err != nil {
		return e.Forward(err)
	}

	command := &Command{
		Timeout: timeout,
		Conn:    conn,
	}

	var c *smtp.Client
	r := command.Exec(smtp.NewClient, conn, serverName)
	r(&c, &err)
	if err != nil {
		return e.Forward(err)
	}
	defer c.Close()

	if hello != "" {
		r = command.Exec(c.Hello, hello)
		r(&err)
		if err != nil {
			return e.Forward(err)
		}
	}

	if ok, _ := c.Extension("STARTTLS"); ok {
		r = command.Exec(c.StartTLS, &tls.Config{
			ServerName:         serverName,
			InsecureSkipVerify: insecureSkipVerify,
		})
		r(&err)
		if err != nil {
			return e.Forward(err)
		}
	}

	if a != nil {
		found, _ := c.Extension("AUTH")
		if found {
			r = command.Exec(c.Auth, a)
			r(&err)
			if err != nil {
				return e.Forward(err)
			}
		}
	}

	r = command.Exec(c.Reset)
	r(&err)
	if err != nil {
		return e.New(err)
	}

	r = command.Exec(c.Quit)
	r(&err)
	if err != nil {
		return e.New(err)
	}

	return nil
}
