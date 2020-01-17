package httptestutil

import (
	"bytes"
	"fmt"
	"github.com/felixge/httpsnoop"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
)

// DumpTo wraps an http.Handler in a new handler.  The new handler dumps requests and responses to
// a writer, using the httputil.DumpRequest and httputil.DumpResponse functions.
func DumpTo(handler http.Handler, writer io.Writer) http.Handler {

	// use the same default as http.Server
	if handler == nil {
		handler = http.DefaultServeMux
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dump, err := httputil.DumpRequest(r, true)
		if err != nil {
			_, _ = fmt.Fprintf(writer, "error dumping request: %#v", err)
		} else {
			_, _ = writer.Write(append(dump, []byte("\r\n")...))
		}

		ex := Exchange{}

		w = httpsnoop.Wrap(w, hooks(&ex))

		handler.ServeHTTP(w, r)

		// create a dummy response to dump
		resp := http.Response{
			Proto:         r.Proto,
			ProtoMajor:    r.ProtoMajor,
			ProtoMinor:    r.ProtoMinor,
			StatusCode:    ex.StatusCode,
			Header:        w.Header(),
			Body:          ioutil.NopCloser(bytes.NewReader(ex.ResponseBody.Bytes())),
			ContentLength: int64(ex.ResponseBody.Len()),
		}

		d, err := httputil.DumpResponse(&resp, true)
		if err != nil {
			_, _ = fmt.Fprintf(writer, "error dumping response: %#v", err)
		} else {
			_, _ = writer.Write(append(d, []byte("\r\n")...))
		}
	})
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
