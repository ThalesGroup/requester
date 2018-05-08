// Package clientserver is a utility for writing HTTP tests.
//
// A ClientServer embeds an httptest.Server and
// a requester.Requester client.  The client is preconfigured
// to talk to the server.
package clientserver

import (
	"github.com/gemalto/requester"
	"net/http"
	"net/http/httptest"
)

// A ClientServer is wrapper around httptest.Server.  It adds a few
// convenience functions for configuring the Handler to a HandlerFunc
// or a ServerMux.
//
// It also has convenience methods for returning a pre-configured Requester
// for sending requests to the server, and methods for installing
// client-side and server-side Inspectors.
//
// When testing HTTP client code, use ClientServer to quickly
// configure mock handlers and using the server-side Inspector to see
// what the client is sending.
//
// When testing HTTP handlers, use ClientServer to launch an HTTP server,
// construct test requests with Requester, and use the Inspectors to assert
// values on the client or server side.
//
// The Inspectors are created lazily, on the first call to InspectServer() or
// InspectClient().  Capturing of exchanges will not start until these methods
// are called the first time.
//
// Should be closed at the end of the test:
//
//     cs := NewServer(nil)
//     defer cs.Close()
//
type ClientServer struct {
	*httptest.Server

	Handler http.Handler

	requester *requester.Requester

	// This inspector is installed in the Requester
	clientInspector *requester.Inspector

	// This inspector is installed in the server
	serverInspector *Inspector
}

// NewServer creates and starts a new ClientServer.
func NewServer(handler http.Handler) *ClientServer {
	return newServer(httptest.NewServer(handler))
}

// NewTLSServer creates and starts a new ClientServer in TLS mode.
// Requester() will return a client already configured with the server's
// certificates.
func NewTLSServer(handler http.Handler) *ClientServer {
	return newServer(httptest.NewTLSServer(handler))
}

// NewUnstartedServer creates a new ClientServer which hasn't
// been started yet.
//
//     cs := NewUnstartedServer(nil)
//     cs.Start()
//     defer cs.Close()
func NewUnstartedServer(handler http.Handler) *ClientServer {
	return newServer(httptest.NewUnstartedServer(handler))
}

func newServer(s *httptest.Server) *ClientServer {
	t := &ClientServer{
		Server: s,
	}

	// intercept the standard handler
	t.Handler = t.Server.Config.Handler
	t.Server.Config.Handler = t.handler()

	return t
}

// Clear clears server and client inspectors.
func (t *ClientServer) Clear() {
	t.serverInspector.Clear()
	t.clientInspector.Clear()
}

// Mux returns a ServeMux.  If the current Handler is a ServeMux, that
// is returned.  Otherwise, a new ServerMux is created and installed as
// the handler.  This will override whatever the prior Handler was.
func (t *ClientServer) Mux() *http.ServeMux {
	if m, ok := t.Config.Handler.(*http.ServeMux); ok {
		return m
	}
	m := http.NewServeMux()
	t.Handler = m
	return m
}

// HandlerFunc is a convience method for installing a HandlerFunc as the
// handler.
func (t *ClientServer) HandlerFunc(hf http.HandlerFunc) {
	t.Handler = hf
}

// Requester returns a Requester which is preconfigured to talk to the
// server.  The Requester's base URL will be set to the server's URL, and in TLS
// mode, it will be pre-configured to trust the server's certificate.
func (t *ClientServer) Requester() *requester.Requester {
	if t.requester == nil {
		t.requester = requester.MustNew(
			requester.WithDoer(t.Client()),
			requester.URL(t.URL),
		)
	}
	return t.requester
}

// InspectClient activates client side inspection.  A requester.Inspector
// is installed in the Requester which captures outgoing requests and
// incoming responses.
//
// Inspection is not activated until InspectClient is called the
// first time.
//
// Indempotent: Subsequent calls return the same Inspector.
func (t *ClientServer) InspectClient() *requester.Inspector {
	if t.clientInspector == nil {
		t.clientInspector = &requester.Inspector{}
		t.Requester().MustApply(t.clientInspector)
	}
	return t.clientInspector
}

// InspectServer activates server side inspection.  An Inspector
// is installed around the server Handler which captures incoming
// requests and outgoing responses.
//
// Inspection is not activated until InspectServer is called the first
// time.
//
// Indempotent: Subsequent calls return the same Inspector.
func (t *ClientServer) InspectServer() *Inspector {
	if t.serverInspector == nil {
		t.serverInspector = NewInspector(0)
	}
	return t.serverInspector
}

func (t *ClientServer) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if h := t.Handler; h != nil {
			// inject the server inspector, if active
			if t.serverInspector != nil {
				h = t.serverInspector.MiddlewareFunc(h)
			}
			h.ServeHTTP(w, req)
		}
	})
}
