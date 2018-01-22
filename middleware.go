package requests

import (
	"github.com/ansel1/merry"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
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

// Non2XXResponseAsError converts error responses from the server into
// an `error`.  For simple code, this removes the need to check both
// `err` and `resp.StatusCode`.
//
// The body of the response is dumped into the error message, for example:
//
//     fmt.Println(err)
//
// ...might output:
//
//     server returned non-2XX status code:
//     HTTP/1.1 407 Proxy Authentication Required
//	   Content-Length: 5
//	   Content-Type: text/plain; charset=utf-8
//	   Date: Mon, 22 Jan 2018 18:55:18 GMT
//
//	   boom!
//
// This probably isn't appropriate for production code, where the full response might be
// sensitive, or too long, or binary, but it should be OK for tests or sample code.
//
// For production code, consider using this as an example for your own error handler.
//
func Non2XXResponseAsError() Middleware {
	return func(next Doer) Doer {
		return DoerFunc(func(req *http.Request) (*http.Response, error) {
			resp, err := next.Do(req)
			if err == nil && (resp.StatusCode < 200 || resp.StatusCode > 299) {
				// it's an error.  capture response body as text
				defer resp.Body.Close()

				d, _ := httputil.DumpResponse(resp, true)

				msg := "server returned non-2XX status code"
				if len(d) > 0 {
					msg += ":\n"
					msg += string(d)
				} else {
					msg += ": " + strconv.Itoa(resp.StatusCode)
				}
				err := merry.New(msg)
				err = err.WithHTTPCode(resp.StatusCode)
				return resp, err
			}
			return resp, err
		})
	}
}
