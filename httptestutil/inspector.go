package httptestutil

import (
	"bytes"
	"github.com/felixge/httpsnoop"
	"io"
	"io/ioutil"
	"net/http"
)

// Exchange is a snapshot of one request/response exchange with
// the server.
type Exchange struct {
	Request     *http.Request
	RequestBody *bytes.Buffer

	StatusCode   int
	Header       http.Header
	ResponseBody *bytes.Buffer
}

// Inspector is server-side middleware which captures server exchanges in a buffer.
// Exchanges are captured in a buffered channel.  If the channel buffer fills,
// subsequent server exchanges are not captured.
//
// Exchanges can be received directly from the channel, or you can use the NextExchange()
// and LastExchange() convenience methods.
type Inspector struct {
	Exchanges chan Exchange
}

// NewInspector creates a new Inspector with the requested channel buffer size.  If 0,
// the buffer size defaults to 50.
func NewInspector(size int) *Inspector {
	if size == 0 {
		size = 50
	}
	return &Inspector{
		Exchanges: make(chan Exchange, size),
	}
}

// NextExchange receives the next exchange from the channel, or returns nil if no
// exchange is ready.  It is non-blocking.
func (b *Inspector) NextExchange() *Exchange {
	select {
	case e := <-b.Exchanges:
		return &e
	default:
		return nil
	}
}

// LastExchange receives the most recent exchange from channel.  This also has
// the side effect of draining the channel completely.  If no exchange
// is ready, nil is returned.  It is non-blocking.
func (b *Inspector) LastExchange() *Exchange {
	var e *Exchange

	for {
		select {
		case ex := <-b.Exchanges:
			e = &ex
		default:
			return e
		}
	}
}

// Clear drains the channel.
func (b *Inspector) Clear() {
	if b == nil {
		return
	}
	b.LastExchange()
}

// MiddlewareFunc installs the inspector in an HTTP server by wrapping
// the server's Handler.
func (b *Inspector) MiddlewareFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ex := Exchange{}
		ex.Request = r
		if r.Body != nil && r.Body != http.NoBody {
			ex.RequestBody = &bytes.Buffer{}
			if _, err := ex.RequestBody.ReadFrom(r.Body); err != nil {
				panic(err)
			}
			if err := r.Body.Close(); err != nil {
				panic(err)
			}

			r.Body = ioutil.NopCloser(bytes.NewReader(ex.RequestBody.Bytes()))
		} else {
			ex.RequestBody = nil
		}

		ex.ResponseBody = &bytes.Buffer{}

		w = httpsnoop.Wrap(w, httpsnoop.Hooks{
			Write: func(next httpsnoop.WriteFunc) httpsnoop.WriteFunc {
				return func(b []byte) (int, error) {
					ex.ResponseBody.Write(b)
					return next(b)
				}
			},
			WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
				return func(code int) {
					ex.StatusCode = code
					ex.Header = make(http.Header, len(w.Header()))
					for key, value := range w.Header() {
						ex.Header[key] = value
					}
					next(code)
				}
			},
			ReadFrom: func(next httpsnoop.ReadFromFunc) httpsnoop.ReadFromFunc {
				return func(src io.Reader) (int64, error) {
					_, err := ex.ResponseBody.ReadFrom(src)
					if err != nil {
						return 0, err
					}
					return next(bytes.NewReader(ex.ResponseBody.Bytes()))
				}
			},
		})

		next.ServeHTTP(w, r)

		select {
		case b.Exchanges <- ex:
		default:
			// don't block if channel is full, just drop
		}
	})
}
