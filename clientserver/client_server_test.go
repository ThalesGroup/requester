package clientserver

import (
	"github.com/gemalto/requester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestClientServer_InspectServer(t *testing.T) {

	cs := NewServer(nil)
	defer cs.Close()

	cs.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(201)
		writer.Write([]byte("pong"))
	})

	_, _, err := cs.Requester().Receive(requester.Body("ping"))
	require.NoError(t, err)

	is := cs.InspectServer()

	// no capturing should be happening before the first call to InspectServer
	assert.Empty(t, is.Exchanges)

	_, _, err = cs.Requester().Receive(requester.Body("ping"))
	require.NoError(t, err)

	ex := is.LastExchange()
	require.NotNil(t, ex)
	assert.NotNil(t, ex.Request)
	assert.Equal(t, "ping", ex.RequestBody.String())
	assert.Equal(t, "pong", ex.ResponseBody.String())
}

func TestClientServer_InspectClient(t *testing.T) {
	cs := NewServer(nil)
	defer cs.Close()

	cs.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(201)
		writer.Write([]byte("pong"))
	})

	_, _, err := cs.Requester().Receive(requester.Body("ping"))
	require.NoError(t, err)

	ic := cs.InspectClient()

	// no capturing should be happening before the first call to InspectClient
	assert.Nil(t, ic.Request)

	_, _, err = cs.Requester().Receive(requester.Body("ping"))
	require.NoError(t, err)

	assert.Equal(t, "pong", ic.ResponseBody.String())

}

func TestClientServer_Clear(t *testing.T) {

	cs := NewServer(nil)
	defer cs.Close()

	cs.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(201)
		writer.Write([]byte("pong"))
	})

	is := cs.InspectServer()
	ic := cs.InspectClient()

	cs.Requester().Receive(requester.Get("/test"), requester.Body(`ping`))

	require.Len(t, is.Exchanges, 1)
	require.NotNil(t, ic.Request)
	require.NotNil(t, ic.RequestBody)
	require.NotNil(t, ic.Response)
	require.NotNil(t, ic.ResponseBody)

	cs.Clear()

	assert.Len(t, is.Exchanges, 0)
	assert.Nil(t, ic.Request)
	assert.Nil(t, ic.RequestBody)
	assert.Nil(t, ic.Response)
	assert.Nil(t, ic.ResponseBody)
}

func TestNewTLSServer(t *testing.T) {

	cs := NewTLSServer(nil)
	defer cs.Close()

	cs.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(201)
		writer.Write([]byte("pong"))
	})

	is := cs.InspectServer()

	_, _, err := cs.Requester().Receive(nil)
	require.NoError(t, err)

	ex := is.LastExchange()
	require.NotNil(t, ex)

	assert.NotNil(t, ex.Request.TLS)
}

func TestNewUnstartedServer(t *testing.T) {
	cs := NewUnstartedServer(nil)
	defer cs.Close()

	cs.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(201)
		writer.Write([]byte("pong"))
	})

	cs.Start()

	_, _, err := cs.Requester().Receive(nil)
	require.NoError(t, err)
}
