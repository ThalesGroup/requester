package requests

import (
	"io"
	"net/http"
	"net/http/httputil"
	"os"
)

// Middleware can be used to wrap Doers with additional functionality:
//
//     loggingMiddleware := func(next Doer) Doer {
//         return func(req *http.Request) (*http.Response, error) {
//             logRequest(req)
//             return next(req)
//         }
//     }
//
// Middleware can be applied to a Requests object with the Use() option:
//
//     reqs.Apply(requests.Use(loggingMiddleware))
//
// Middleware itself is an Option, so it can also be applied directly:
//
//     reqs.Apply(Middleware(loggingMiddleware))
//
type Middleware func(Doer) Doer

// Apply implements Option
func (m Middleware) Apply(r *Requests) error {
	r.Middleware = append(r.Middleware, m)
	return nil
}

// Wrap applies a set of middleware to a Doer.  The returned Doer will invoke
// the middleware in the order of the arguments.
func Wrap(d Doer, m ...Middleware) Doer {
	for i := len(m) - 1; i > -1; i-- {
		d = m[i](d)
	}
	return d
}

// Dump dumps requests and responses to a writer.  Just intended for debugging.
func Dump(w io.Writer) Middleware {
	return func(next Doer) Doer {
		return DoerFunc(func(req *http.Request) (*http.Response, error) {
			dump, dumperr := httputil.DumpRequest(req, true)
			// Write the entire request and response out as a single Write() call
			// So if this is being redirected to a logger, it's all sent in a single
			// package
			if dumperr != nil {
				io.WriteString(w, "Error dumping request: "+dumperr.Error()+"\n")
			} else {
				io.WriteString(w, string(dump)+"\n")
			}
			resp, err := next.Do(req)
			if resp != nil {
				dump, dumperr = httputil.DumpResponse(resp, true)
				if dumperr != nil {
					io.WriteString(w, "Error dumping response: "+dumperr.Error()+"\n")
				} else {
					io.WriteString(w, string(dump)+"\n")
				}
			}
			return resp, err
		})
	}
}

// DumpToStandardOut dumps requests to os.Stdout.
func DumpToStandardOut() Middleware {
	return Dump(os.Stdout)
}
