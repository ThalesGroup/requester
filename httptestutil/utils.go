// Package httptestutil contains utilities for use in HTTP tests, particular when using
// httptest.Server.
//
// Inspect() can be used to intercept and inspect the traffic to and from an httptest.Server.
package httptestutil

import (
	"github.com/ThalesGroup/requester"
	"net/http/httptest"
)

// Requester creates a Requester instance which is pre-configured to send requests to
// the test server.  The Requester is configured with the server's base URL, and
// the server's TLS certs (if using a TLS server).
func Requester(ts *httptest.Server, options ...requester.Option) *requester.Requester {
	r := requester.MustNew(requester.URL(ts.URL), requester.WithDoer(ts.Client()))
	r.MustApply(options...)
	return r
}

// Inspect installs and returns an Inspector.  The Inspector captures exchanges with the
// test server.  It's useful in tests to inspect the incoming requests and request bodies
// and the outgoing responses and response bodies.
//
// Inspect wraps and replaces the server's Handler.  It should be called after the real
// Handler has been installed.
func Inspect(ts *httptest.Server) *Inspector {

	i := NewInspector(0)
	ts.Config.Handler = i.Wrap(ts.Config.Handler)

	return i
}
