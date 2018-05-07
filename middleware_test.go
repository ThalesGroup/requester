package requester

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
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

	//cs := clientserver.New(nil)
	//defer cs.Close()

	//cs.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	//	w.Header().Set("Content-Type", "application/json")
	//	w.Write([]byte(`{"color":"red"}`))
	//})

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

func TestExpectCode(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(407)
		w.Write([]byte("boom!"))
	}))
	defer ts.Close()

	//cs := clientserver.New(nil)
	//defer cs.Close()

	//cs.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//	w.WriteHeader(407)
	//	w.Write([]byte("boom!"))
	//})

	// without middleware
	resp, body, err := Receive(Get(ts.URL))
	require.NoError(t, err)
	require.Equal(t, 407, resp.StatusCode)
	require.Equal(t, "boom!", string(body))

	resp, body, err = Receive(Get(ts.URL), ExpectCode(203))
	// body and response should still be returned
	assert.Equal(t, 407, resp.StatusCode)
	assert.Equal(t, "boom!", string(body))
	// but an error should be returned too
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected: 203")
	assert.Contains(t, err.Error(), "received: 407")

}

func TestExpectSuccessCode(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(407)
		w.Write([]byte("boom!"))
	}))
	defer ts.Close()

	//cs := clientserver.New(nil)
	//defer cs.Close()

	//cs.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//	w.WriteHeader(407)
	//	w.Write([]byte("boom!"))
	//})

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
}
