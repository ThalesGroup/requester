package requester

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	sdklog "log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"testing"

	"github.com/ansel1/merry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDump(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"color":"red"}`))
	}))
	defer ts.Close()

	b := &bytes.Buffer{}

	Receive(Get(ts.URL), Dump(b))

	t.Log(b)

	assert.Contains(t, b.String(), "GET / HTTP/1.1")
	assert.Contains(t, b.String(), "HTTP/1.1 200 OK")
	assert.Contains(t, b.String(), `{"color":"red"}`)
}

func TestDumpToLog(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"color":"red"}`))
	}))
	defer ts.Close()

	var args []interface{}

	Receive(Get(ts.URL), DumpToLog(func(a ...interface{}) {
		args = append(args, a...)
	}))

	assert.Len(t, args, 2)

	reqLog := args[0].(string)
	respLog := args[1].(string)

	assert.Contains(t, reqLog, "GET / HTTP/1.1")
	assert.Contains(t, respLog, "HTTP/1.1 200 OK")
	assert.Contains(t, respLog, `{"color":"red"}`)
}

func TestDumpToStout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"color":"red"}`))
	}))
	defer ts.Close()

	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	Receive(Get(ts.URL), DumpToStout())

	// back to normal state
	os.Stdout = old // restoring the real stdout
	w.Close()
	out := <-outC

	assert.Contains(t, out, "GET / HTTP/1.1")
	assert.Contains(t, out, "HTTP/1.1 200 OK")
	assert.Contains(t, out, `{"color":"red"}`)
}

func TestDumpToSterr(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"color":"red"}`))
	}))
	defer ts.Close()

	old := os.Stderr // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = old }()

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	Receive(Get(ts.URL), DumpToStderr())

	// back to normal state
	os.Stderr = old // restoring the real stdout
	w.Close()
	out := <-outC

	assert.Contains(t, out, "GET / HTTP/1.1")
	assert.Contains(t, out, "HTTP/1.1 200 OK")
	assert.Contains(t, out, `{"color":"red"}`)
}

func TestExpectCode(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(407)
		w.Write([]byte("boom!"))
	}))
	defer ts.Close()

	r, err := New(Get(ts.URL))
	require.NoError(t, err)

	// without middleware
	resp, body, err := r.Receive(nil)
	require.NoError(t, err)
	require.Equal(t, 407, resp.StatusCode)
	require.Equal(t, "boom!", string(body))

	// add expect option
	r, err = r.With(ExpectCode(203))
	require.NoError(t, err)

	resp, body, err = r.Receive(nil)
	// body and response should still be returned
	assert.Equal(t, 407, resp.StatusCode)
	assert.Equal(t, "boom!", string(body))
	// but an error should be returned too
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected: 203")
	assert.Contains(t, err.Error(), "received: 407")
	assert.Equal(t, 407, merry.HTTPCode(err))

	// Using the option twice: latest option should win
	_, _, err = r.Receive(ExpectCode(407))
	require.NoError(t, err)

	// original requester's expect option should be unmodified
	_, _, err = r.Receive(nil)
	// but an error should be returned too
	require.Error(t, err)
	require.Equal(t, 407, merry.HTTPCode(err))

}

func TestExpectSuccessCode(t *testing.T) {

	codeToReturn := 407
	bodyToReturn := "boom!"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(codeToReturn)
		_, _ = w.Write([]byte(bodyToReturn))
	}))
	defer ts.Close()

	// without middleware
	resp, body, err := Receive(Get(ts.URL))
	require.NoError(t, err)
	require.Equal(t, 407, resp.StatusCode)
	require.Equal(t, "boom!", string(body))

	resp, body, err = Receive(Get(ts.URL), ExpectSuccessCode())
	// body and response should still be returned
	assert.Equal(t, 407, resp.StatusCode)
	assert.Equal(t, "boom!", string(body))
	// but an error should be returned too
	require.Error(t, err)
	assert.Contains(t, err.Error(), "code: 407")
	assert.Equal(t, 407, merry.HTTPCode(err))

	// test positive path: if success code is returned, then no error should be returned
	successCodes := []int{200, 201, 204, 278}
	for _, code := range successCodes {
		codeToReturn = code
		_, _, err := Receive(Get(ts.URL), ExpectSuccessCode())
		require.NoError(t, err, "should not have received an error for code %v", code)
	}
}

func TestGUnzip(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(200)
		gz := gzip.NewWriter(w)
		defer gz.Close()
		_, err := gz.Write([]byte(`{"color":"green","count":25}`))
		require.NoError(t, err)
	}))
	t.Cleanup(ts.Close)

	t.Run("without middleware", func(t *testing.T) {
		resp, body, err := ReceiveContext(context.Background(),
			Header("Accept-Encoding", "gzip"),
			Get(ts.URL, "/"),
		)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, resp.Header.Get("Content-Encoding"), "gzip")
		assert.Greater(t, resp.ContentLength, int64(0))
		assert.NotEmpty(t, resp.Header.Get("Content-Length"))
		assert.NotEqual(t, `{"color":"green","count":25}`, string(body))
		assert.False(t, resp.Uncompressed)
	})

	t.Run("with middleware", func(t *testing.T) {
		resp, body, err := ReceiveContext(context.Background(),
			Decompress(),
			Header("Accept-Encoding", "gzip"),
			Get(ts.URL, "/"),
		)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Empty(t, resp.Header.Get("Content-Encoding"))
		assert.Equal(t, resp.ContentLength, int64(-1))
		assert.Empty(t, resp.Header.Get("Content-Length"))
		assert.Equal(t, `{"color":"green","count":25}`, string(body))
		assert.True(t, resp.Uncompressed)
	})

}

func ExampleMiddleware() {
	var m Middleware = func(next Doer) Doer {
		return DoerFunc(func(req *http.Request) (*http.Response, error) {
			d, _ := httputil.DumpRequest(req, true)
			fmt.Println(string(d))
			return next.Do(req)
		})
	}

	// Middleware implements Option, so it can be passed directly to functions which accept
	// options
	Send(m)

	// ...or applied with the Use option
	Send(Use(m))

	// ...or applied directly
	_ = Requester{
		Middleware: []Middleware{m},
	}

}

func ExampleDumpToLog() {

	Send(DumpToLog(func(a ...interface{}) {
		fmt.Println(a...)
	}))

	// compatible with the log package's functions
	Send(DumpToLog(sdklog.Println))

	// compatible with the testing package's function
	var t *testing.T
	Send(DumpToLog(t.Log))

}

func ExampleExpectSuccessCode() {

	_, _, err := Receive(
		MockDoer(400),
		ExpectSuccessCode(),
	)

	fmt.Println(err.Error())

	// Output: server returned an unsuccessful status code: 400
}

func ExampleExpectCode() {

	_, _, err := Receive(
		MockDoer(400),
		ExpectCode(201),
	)

	fmt.Println(err.Error())

	// Output: server returned unexpected status code.  expected: 201, received: 400
}
