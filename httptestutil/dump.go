package httptestutil

import (
	"bytes"
	"fmt"
	"github.com/felixge/httpsnoop"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
)

// DumpTo wraps an http.Handler.  It dumps requests and responses to
// a writer, using the httputil.DumpRequest and httputil.DumpResponse functions.
func DumpTo(handler http.Handler, writer io.Writer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dump, err := httputil.DumpRequest(r, true)
		if err != nil {
			fmt.Fprintf(writer, "error dumping request: %#v", err)
		} else {
			writer.Write(append(dump, []byte("\r\n")...))
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
			fmt.Fprintf(writer, "error dumping response: %#v", err)
		} else {
			writer.Write(append(d, []byte("\r\n")...))
		}
	})
}
