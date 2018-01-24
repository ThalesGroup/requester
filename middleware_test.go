package requester_test

import (
	"bytes"
	. "github.com/gemalto/requester"
	"github.com/gemalto/requester/clientserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestDump(t *testing.T) {
	cs := clientserver.New(nil)
	defer cs.Close()
	cs.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"color":"red"}`))
	})

	b := &bytes.Buffer{}

	cs.Receive(nil, Dump(b))

	t.Log(b)

	assert.Contains(t, b.String(), "GET / HTTP/1.1")
	assert.Contains(t, b.String(), "HTTP/1.1 200 OK")
	assert.Contains(t, b.String(), `{"color":"red"}`)
}

func TestNon2XXResponseAsError(t *testing.T) {
	cs := clientserver.New(nil)
	defer cs.Close()

	cs.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(407)
		w.Write([]byte("boom!"))
	})

	// without the middleware
	resp, body, err := cs.Receive(nil)
	require.NoError(t, err)
	require.Equal(t, 407, resp.StatusCode)
	require.Equal(t, "boom!", string(body))

	// with the middleware
	resp, _, err = cs.Receive(nil, Non2XXResponseAsError())

	require.Error(t, err)
	assert.Equal(t, 407, resp.StatusCode)

	t.Log(err)

}
