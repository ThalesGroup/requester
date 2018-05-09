package requester

import (
	"github.com/ansel1/merry"
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
// Middleware can be applied to a Requester object with the Use() option:
//
//     reqs.Apply(requester.Use(loggingMiddleware))
//
// Middleware itself is an Option, so it can also be applied directly:
//
//     reqs.Apply(Middleware(loggingMiddleware))
//
type Middleware func(Doer) Doer

// Apply implements Option
func (m Middleware) Apply(r *Requester) error {
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
			dump, dumperr := httputil.DumpRequestOut(req, true)
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

// DumpToStout dumps requests to os.Stdout.
func DumpToStout() Middleware {
	return Dump(os.Stdout)
}

type logFunc func(a ...interface{})

func (f logFunc) Write(p []byte) (n int, err error) {
	f(string(p))
	return len(p), nil
}

// DumpToLog dumps the request and response to a logging function.
// logf is compatible with fmt.Print(), testing.T.Log, or log.XXX()
// functions.
//
// Request and response will be logged separately.  Though logf
// takes a variadic arg, it will only be called with one string
// arg at a time.
func DumpToLog(logf func(a ...interface{})) Middleware {
	return Dump(logFunc(logf))
}

// ExpectCode is middleware which generates an error if the response's status code does not match
// the expected code.
func ExpectCode(code int) Middleware {
	return func(next Doer) Doer {
		return DoerFunc(func(req *http.Request) (*http.Response, error) {
			resp, err := next.Do(req)
			if err == nil && resp != nil && resp.StatusCode != code {
				return resp, merry.Errorf("server returned unexpected status code.  expected: %d, received: %d", code, resp.StatusCode)
			}

			return resp, err
		})
	}
}

// ExpectSuccessCode is middleware which generates an error if the response's status code is not between 200 and
// 299.
func ExpectSuccessCode() Middleware {
	return func(next Doer) Doer {
		return DoerFunc(func(req *http.Request) (*http.Response, error) {
			resp, err := next.Do(req)
			if err == nil && resp != nil && (resp.StatusCode < 200 || resp.StatusCode > 299) {
				return resp, merry.Errorf("server returned an unsuccessful status code: %d", resp.StatusCode)
			}

			return resp, err
		})
	}
}
