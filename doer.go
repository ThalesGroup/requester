package requests

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

// Middleware can be used to wrap Doers with additional functionality:
//
//     loggingMiddleware := func(next Doer) Doer {
//         return func(req *http.Request) (*http.Response, error) {
//             logRequest(req)
//             return next(req)
//         }
//     }
//
type Middleware func(Doer) Doer

// Wrap applies a set of middleware to a Doer.  The returned Doer will invoke
// the middleware in the order of the arguments.
func Wrap(d Doer, m ...Middleware) Doer {
	for i := len(m) - 1; i > -1; i-- {
		d = m[i](d)
	}
	return d
}
