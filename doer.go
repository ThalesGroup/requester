package requester

import "net/http"

// Doer executes http requests.  It is implemented by *http.Client.  You can
// wrap *http.Client with layers of Doers to form a stack of client-side
// middleware.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// DoerFunc adapts a function to implement Doer
type DoerFunc func(req *http.Request) (*http.Response, error)

// Do implements the Doer interface
func (f DoerFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}
