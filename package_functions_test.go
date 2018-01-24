package requester_test

import (
	"context"
	. "github.com/gemalto/requester"
	"github.com/gemalto/requester/clientserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestRequest(t *testing.T) {
	req, err := Request(Get("http://blue.com/red"))
	require.NoError(t, err)
	require.NotNil(t, req)
	require.Equal(t, "http://blue.com/red", req.URL.String())
}

type testContextKey string

const colorContextKey = testContextKey("color")

func TestRequestContext(t *testing.T) {
	req, err := RequestContext(
		context.WithValue(context.Background(), colorContextKey, "green"),
		Get("http://blue.com/red"),
	)
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, "http://blue.com/red", req.URL.String())
	assert.Equal(t, "green", req.Context().Value(colorContextKey))
}

func TestSend(t *testing.T) {

	cs := clientserver.New(nil)
	defer cs.Close()

	cs.Mux().HandleFunc("/red", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})

	resp, err := Send(Get(cs.Server.URL, "red"))
	require.NoError(t, err)

	assert.Equal(t, 204, resp.StatusCode)
}

func TestSendContext(t *testing.T) {
	cs := clientserver.New(nil)
	defer cs.Close()

	cs.Mux().HandleFunc("/red", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})

	resp, err := SendContext(
		context.WithValue(context.Background(), colorContextKey, "blue"),
		Get(cs.Server.URL, "red"),
		WithDoer(cs),
	)

	require.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)
	assert.Equal(t, "blue", cs.LastClientReq.Context().Value(colorContextKey))
}

type testModel struct {
	Color string `xml:"color" json:"color" url:"color"`
	Count int    `xml:"count" json:"count" url:"count"`
}

func TestReceive(t *testing.T) {
	cs := clientserver.New(nil)
	defer cs.Close()

	cs.Mux().HandleFunc("/red", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderContentType, ContentTypeJSON)
		w.WriteHeader(205)
		w.Write([]byte(`{"count":25}`))

	})

	var m testModel
	resp, body, err := Receive(&m, Get(cs.Server.URL, "red"))
	require.NoError(t, err)

	assert.Equal(t, `{"count":25}`, string(body))
	assert.Equal(t, 205, resp.StatusCode)
	assert.Equal(t, 25, m.Count)

	t.Run("Context", func(t *testing.T) {
		var m testModel

		resp, body, err := ReceiveContext(
			context.WithValue(context.Background(), colorContextKey, "yellow"),
			&m,
			Get(cs.Server.URL, "red"),
			WithDoer(cs),
		)
		require.NoError(t, err)

		assert.Equal(t, `{"count":25}`, string(body))
		assert.Equal(t, 205, resp.StatusCode)
		assert.Equal(t, 25, m.Count)
		assert.Equal(t, "yellow", cs.LastClientReq.Context().Value(colorContextKey))
	})
}
