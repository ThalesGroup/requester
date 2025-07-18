package requester

import (
	"compress/gzip"
	"context"
	"github.com/ansel1/merry"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
)

// Middleware can be used to wrap Doers with additional functionality.
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
				_, _ = io.WriteString(w, "Error dumping request: "+dumperr.Error()+"\n")
			} else {
				_, _ = io.WriteString(w, string(dump)+"\n")
			}
			resp, err := next.Do(req)
			if resp != nil {
				dump, dumperr = httputil.DumpResponse(resp, true)
				if dumperr != nil {
					_, _ = io.WriteString(w, "Error dumping response: "+dumperr.Error()+"\n")
				} else {
					_, _ = io.WriteString(w, string(dump)+"\n")
				}
			}
			return resp, err
		})
	}
}

// DumpToStout dumps requests and responses to os.Stdout
func DumpToStout() Middleware {
	return Dump(os.Stdout)
}

// DumpToStderr dumps requests and responses to os.Stderr
func DumpToStderr() Middleware {
	return Dump(os.Stderr)
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
// logf will be invoked once for the request, and once for the response.
// Each invocation will only have a single argument (the entire request
// or response is logged as a single string value).
func DumpToLog(logf func(a ...interface{})) Middleware {
	return Dump(logFunc(logf))
}

// ExpectCode generates an error if the response's status code does not match
// the expected code.
//
// The response body will still be read and returned.
func ExpectCode(code int) Middleware {
	return func(next Doer) Doer {
		return DoerFunc(func(req *http.Request) (*http.Response, error) {
			r, c := getCodeChecker(req)
			c.code = code
			resp, err := next.Do(r)
			return c.checkCode(resp, err)
		})
	}
}

// ExpectSuccessCode is middleware which generates an error if the response's status code is not between 200 and
// 299.
//
// The response body will still be read and returned.
func ExpectSuccessCode() Middleware {
	return func(next Doer) Doer {
		return DoerFunc(func(req *http.Request) (*http.Response, error) {
			r, c := getCodeChecker(req)
			c.code = expectSuccessCode
			resp, err := next.Do(r)
			return c.checkCode(resp, err)
		})
	}
}

type ctxKey int

const expectCodeCtxKey ctxKey = iota

const expectSuccessCode = -1

type codeChecker struct {
	code int
}

func (c *codeChecker) checkCode(resp *http.Response, err error) (*http.Response, error) {
	switch {
	case err != nil, resp == nil:
	case c.code == expectSuccessCode:
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			err = merry.
				Errorf("server returned an unsuccessful status code: %d", resp.StatusCode).
				WithHTTPCode(resp.StatusCode)
		}
	case c.code != resp.StatusCode:
		err = merry.
			Errorf("server returned unexpected status code.  expected: %d, received: %d", c.code, resp.StatusCode).
			WithHTTPCode(resp.StatusCode)
	}
	return resp, err
}

func getCodeChecker(req *http.Request) (*http.Request, *codeChecker) {
	c, _ := req.Context().Value(expectCodeCtxKey).(*codeChecker)
	if c == nil {
		c = &codeChecker{}
		req = req.WithContext(context.WithValue(req.Context(), expectCodeCtxKey, c))
	}
	return req, c
}

// Decompress middleware will decompress the response body if the response
// Content-Type indicates the body is compressed.
//
// Normally, this is not needed.  Golang's default HTTP transport
// automatically requests compression and automatically decompresses
// the response.  However, the transport will only auto-decompress if
// it originally requested the compression.
//
// Cases where this middleware is needed:
//   - if the Accept-Encoding header is explicitly set to "gzip" by the
//     caller, the transport will not do any automatic compression processing
//   - if the server returns compressed responses even when compression
//     was not requested by the client (i.e. the Accept-Encoding header was
//     not set on the request).  Technically, servers should not use
//     compression unless the client requests it, but some servers are
//     known to violate this rule.
//
// This middleware currently only support gzip compression.
func Decompress() Middleware {
	return func(d Doer) Doer {
		return DoerFunc(func(req *http.Request) (*http.Response, error) {
			resp, err := d.Do(req)
			if err != nil || resp == nil {
				return resp, err
			}
			if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
				gr, err := gzip.NewReader(resp.Body)
				if err != nil {
					resp.Body.Close()
					return nil, err
				}
				// Replace the original Body with the decompressed reader
				resp.Body = struct {
					io.Reader
					io.Closer
				}{
					Reader: gr,
					Closer: resp.Body, // we keep closing the original
				}
				resp.Header.Del("Content-Encoding")
				resp.Header.Del("Content-Length")
				resp.ContentLength = -1
				resp.Uncompressed = true
			}
			return resp, err
		})
	}
}
