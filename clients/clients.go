// Package clients is a set of utilities for creating and
// configuring instances of http.Client.
//
// Clients are created with the NewClient() function, which takes a
// set of Options to configure the client.  Options implement common
// configuration recipes, like disabling server TLS verification, or
// setting a simple proxy.
//
// Example:
//
//     c, err := clients.NewClient(clients.SkipVerify(), clients.Timeout(10 * time.Seconds))
//
package clients

import (
	"net"
	"net/http"
	"time"
)

// NewClient builds a new *http.Client.  With no arguments, the client
// will be configured identically to the http.DefaultClient and
// http.DefaultTransport, but will be different instances (so they
// can be further modified without having a global effect).
//
// The client can be further configured by passing Options.
func NewClient(opts ...Option) (*http.Client, error) {
	// fyi: first iteration of this made a shallow copy
	// of http.DefaultTransport, but `go vet` complains that
	// we're making a copy of mutex lock in Transport (legit).
	// So we're just copying the init code.  Need to keep an eye
	// on this in future golang releases

	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	c := &http.Client{}

	for _, opt := range opts {
		err := opt.Apply(c, t)
		if err != nil {
			return nil, err
		}
	}

	// if one of the options explicitly sets the transport, that
	// overrides our transport
	if c.Transport != nil {
		c.Transport = t
	}
	return c, nil
}

// Option is a configuration option for building an http.Client.
type Option interface {

	// Apply is called when constructing an http.Client.  Apply
	// should make some configuration change to the arguments.
	//
	// The client and transport arguments will not be nil, and it
	// should be assumed that after being called, the transport
	// argument will be installed into the client as its RoundTripper.
	//
	// If Apply directly sets the client's RoundTripper, then the
	// transport argument will be ignored (it will not be installed
	// in the client after all the options have been invoked).
	Apply(*http.Client, *http.Transport) error
}

// ClientOptionFunc adapts a function to the Option interface.
type ClientOptionFunc func(*http.Client, *http.Transport) error

// Apply implements Option.
func (f ClientOptionFunc) Apply(c *http.Client, t *http.Transport) error {
	return f(c, t)
}
