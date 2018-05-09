// Package httptestutil contains utilities for use in HTTP tests, particular when using
// httptest.Server.
//
// Inspect() can be used to intercept and inspect the traffic to and from an httptest.Server.
package httptestutil

import (
	"github.com/gemalto/requester"
	"io"
	"net/http/httptest"
	"os"
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

// Dump writes requests and responses to the writer.
func Dump(ts *httptest.Server, to io.Writer) {
	ts.Config.Handler = DumpTo(ts.Config.Handler, to)
}

// DumpToStdout writes requests and responses to os.Stdout.
func DumpToStdout(ts *httptest.Server) {
	Dump(ts, os.Stdout)
}

type logFunc func(a ...interface{})

// Write implements io.Writer.
func (f logFunc) Write(p []byte) (n int, err error) {
	f(string(p))
	return len(p), nil
}

// DumpToLog writes requests and responses to a logging function.  The function
// signature is the same as testing.T.Log, so it can be used to pipe
// traffic to the test log:
//
//     func TestHandler(t *testing.T) {
//         ...
//         DumpToLog(ts, t.Log)
//
func DumpToLog(ts *httptest.Server, logf func(a ...interface{})) {
	Dump(ts, logFunc(logf))
}
