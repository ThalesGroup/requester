package clientserver

import (
	"fmt"
	"github.com/gemalto/requester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestNewInspector(t *testing.T) {
	i := NewInspector(0)

	cs := NewServer(nil)
	defer cs.Close()

	cs.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(201)
		writer.Write([]byte("pong"))
	})

	origHandler := cs.Handler

	cs.Handler = i.MiddlewareFunc(origHandler)

	cs.Requester().Receive(requester.Get("/test"))
	cs.Requester().Receive(requester.Get("/test"))

	assert.Len(t, i.Exchanges, 2)

	i = NewInspector(5)

	cs.Handler = i.MiddlewareFunc(origHandler)

	// run ten requests
	for i := 0; i < 10; i++ {
		cs.Requester().Receive(requester.Get("/test"))
	}

	// channel should only have buffered 5
	assert.Len(t, i.Exchanges, 5)
}

func TestInspector(t *testing.T) {

	cs := NewServer(nil)
	defer cs.Close()

	cs.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(201)
		writer.Write([]byte("pong"))
	})

	is := cs.InspectServer()

	cs.Requester().Receive(requester.Get("/test"), requester.Body("ping"))

	ex := is.LastExchange()
	require.NotNil(t, ex)
	assert.Equal(t, "/test", ex.Request.URL.Path)
	assert.Equal(t, "ping", ex.RequestBody.String())
	assert.Equal(t, 201, ex.StatusCode)
	assert.Equal(t, "pong", ex.ResponseBody.String())
}

func TestInspector_NextExchange(t *testing.T) {
	cs := NewServer(nil)
	defer cs.Close()

	var count int
	cs.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(201)
		writer.Write([]byte("pong" + strconv.Itoa(count)))
		count++
	})

	is := cs.InspectServer()

	cs.Requester().Receive(requester.Get("/test"))
	cs.Requester().Receive(requester.Get("/test"))
	cs.Requester().Receive(requester.Get("/test"))

	var exchanges []*Exchange

	for {
		ex := is.NextExchange()
		if ex == nil {
			break
		}
		exchanges = append(exchanges, ex)
	}

	assert.Len(t, exchanges, 3)
	assert.Equal(t, "pong0", exchanges[0].ResponseBody.String())
	assert.Equal(t, "pong1", exchanges[1].ResponseBody.String())
	assert.Equal(t, "pong2", exchanges[2].ResponseBody.String())
}

func TestInspector_LastExchange(t *testing.T) {
	cs := NewServer(nil)
	defer cs.Close()

	var count int
	cs.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(201)
		writer.Write([]byte("pong" + strconv.Itoa(count)))
		count++
	})

	is := cs.InspectServer()

	cs.Requester().Receive(requester.Get("/test"))
	cs.Requester().Receive(requester.Get("/test"))
	cs.Requester().Receive(requester.Get("/test"))

	ex := is.LastExchange()

	require.NotNil(t, ex)
	assert.Equal(t, "pong2", ex.ResponseBody.String())

	require.Nil(t, is.LastExchange())
}

func TestInspector_Clear(t *testing.T) {
	cs := NewServer(nil)
	defer cs.Close()

	var count int
	cs.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(201)
		writer.Write([]byte("pong" + strconv.Itoa(count)))
		count++
	})

	is := cs.InspectServer()

	cs.Requester().Receive(requester.Get("/test"))
	cs.Requester().Receive(requester.Get("/test"))
	cs.Requester().Receive(requester.Get("/test"))

	require.Len(t, is.Exchanges, 3)

	is.Clear()

	require.Empty(t, is.Exchanges)
}

func ExampleInspector_MiddlewareFunc() {
	mux := http.NewServeMux()
	// configure mux...

	i := NewInspector(0)

	ts := httptest.NewServer(i.MiddlewareFunc(mux))
	defer ts.Close()
}

func ExampleInspector_NextExchange() {
	i := NewInspector(0)

	var h http.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(`pong`))
	})

	h = i.MiddlewareFunc(h)

	ts := httptest.NewServer(h)
	defer ts.Close()

	requester.Receive(requester.Get(ts.URL), requester.Body("ping1"))
	requester.Receive(requester.Get(ts.URL), requester.Body("ping2"))

	fmt.Println(i.NextExchange().RequestBody.String())
	fmt.Println(i.NextExchange().RequestBody.String())
	fmt.Println(i.NextExchange())

	// Output:
	// ping1
	// ping2
	// <nil>
}

func ExampleInspector_LastExchange() {
	i := NewInspector(0)

	var h http.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(`pong`))
	})

	h = i.MiddlewareFunc(h)

	ts := httptest.NewServer(h)
	defer ts.Close()

	requester.Receive(requester.Get(ts.URL), requester.Body("ping1"))
	requester.Receive(requester.Get(ts.URL), requester.Body("ping2"))

	fmt.Println(i.LastExchange().RequestBody.String())
	fmt.Println(i.LastExchange())

	// Output:
	// ping2
	// <nil>
}
