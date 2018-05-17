// Package httpclient is a set of utilities for creating and
// configuring instances of http.Client.
//
// Clients are created with the New() function, which takes a
// set of Options to configure the client.  Options implement common
// configuration recipes, like disabling server TLS verification, or
// setting a simple proxy.
//
// Example:
//
//     c, err := httpclient.New(httpclient.SkipVerify(), httpclient.Timeout(10 * time.Seconds))
//
package httpclient

import (
	"crypto/tls"
	"github.com/ansel1/merry"
	"net"
	"net/http"
	"time"
)

// New builds a new *http.Client.  With no arguments, the client
// will be configured identically to the http.DefaultClient and
// http.DefaultTransport, but will be different instances (so they
// can be further modified without having a global effect).
//
// The client can be further configured by passing Options.
func New(opts ...Option) (*http.Client, error) {
	// fyi: first iteration of this made a shallow copy
	// of http.DefaultTransport, but `go vet` complains that
	// we're making a copy of mutex lock in Transport (legit).
	// So we're just copying the init code.  Need to keep an eye
	// on this in future golang releases

	c := &http.Client{}
	return c, Apply(c, opts...)
}

// Apply applies options to an existing client.
func Apply(c *http.Client, opts ...Option) error {
	for _, opt := range opts {
		err := opt.Apply(c)
		if err != nil {
			return err
		}
	}
	return nil
}

func newDefaultTransport() *http.Transport {
	return &http.Transport{
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
}

// Option is a configuration option for building an http.Client.
type Option interface {

	// Apply is called when constructing an http.Client.  Apply
	// should make some configuration change to the argument.
	//
	// The client argument will not be nil.
	Apply(*http.Client) error
}

// OptionFunc adapts a function to the Option interface.
type OptionFunc func(*http.Client) error

// Apply implements Option.
func (f OptionFunc) Apply(c *http.Client) error {
	return f(c)
}

// A TransportOption configures the client's transport.
//
// The argument will never be nil.  TransportOption will
// create a default http.Transport (configured identically
// to the http.DefaultTransport) if necessary.
//
// If the client's transport is not a *http.Transport, an
// error is returned.
type TransportOption func(transport *http.Transport) error

// Apply implements Option.
func (f TransportOption) Apply(c *http.Client) error {
	var transport *http.Transport
	rt := c.Transport
	switch t := rt.(type) {
	case nil:
		transport = newDefaultTransport()
		c.Transport = transport
	case *http.Transport:
		transport = t
	default:
		return merry.Errorf("client.Transport is not a *http.Transport.  It's a %T", c.Transport)
	}

	return f(transport)
}

// A TLSOption is a type of Option which configures the
// TLS configuration of the client.
//
// The argument will never be nil.  A new, default config will be
// created if necessary.
//
// See SkipVerify for an example implementation.
type TLSOption func(c *tls.Config) error

// Apply implements Option.
func (f TLSOption) Apply(c *http.Client) error {
	return TransportOption(func(t *http.Transport) error {
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{}
		}
		return f(t.TLSClientConfig)
	}).Apply(c)
}
