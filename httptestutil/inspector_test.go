package httptestutil

import (
	"fmt"
	"github.com/gemalto/requester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestNewInspector(t *testing.T) {
	i := NewInspector(0)

	ts := httptest.NewServer(requester.MockHandler(201, requester.Body("pong")))
	defer ts.Close()

	origHandler := ts.Config.Handler

	ts.Config.Handler = i.Wrap(origHandler)

	Requester(ts).Receive(requester.Get("/test"))
	Requester(ts).Receive(requester.Get("/test"))

	assert.Len(t, i.Exchanges, 2)

	i = NewInspector(5)

	ts.Config.Handler = i.Wrap(origHandler)

	// run ten requests
	for i := 0; i < 10; i++ {
		Requester(ts).Receive(requester.Get("/test"))
	}

	// channel should only have buffered 5
	assert.Len(t, i.Exchanges, 5)
}

func TestInspector(t *testing.T) {

	ts := httptest.NewServer(requester.MockHandler(201, requester.Body("pong")))
	defer ts.Close()

	is := Inspect(ts)

	resp, body, err := Requester(ts).Receive(requester.Get("/test"), requester.Body("ping"))
	require.NoError(t, err)

	assert.Equal(t, 201, resp.StatusCode)
	assert.Equal(t, "pong", string(body))

	ex := is.LastExchange()
	require.NotNil(t, ex)
	assert.Equal(t, "/test", ex.Request.URL.Path)
	assert.Equal(t, "ping", ex.RequestBody.String())
	assert.Equal(t, 201, ex.StatusCode)
	assert.Equal(t, "pong", ex.ResponseBody.String())
}

func TestInspector_NextExchange(t *testing.T) {

	var count int

	ts := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(201)
		writer.Write([]byte("pong" + strconv.Itoa(count)))
		count++
	}))
	defer ts.Close()

	is := Inspect(ts)

	Requester(ts).Receive(requester.Get("/test"))
	Requester(ts).Receive(requester.Get("/test"))
	Requester(ts).Receive(requester.Get("/test"))

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
	ts := httptest.NewServer(nil)
	defer ts.Close()

	var count int
	ts.Config.Handler = http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(201)
		writer.Write([]byte("pong" + strconv.Itoa(count)))
		count++
	})

	is := Inspect(ts)

	Requester(ts).Receive(requester.Get("/test"))
	Requester(ts).Receive(requester.Get("/test"))
	Requester(ts).Receive(requester.Get("/test"))

	ex := is.LastExchange()

	require.NotNil(t, ex)
	assert.Equal(t, "pong2", ex.ResponseBody.String())

	require.Nil(t, is.LastExchange())
}

func TestInspector_Drain(t *testing.T) {
	ts := httptest.NewServer(nil)
	defer ts.Close()

	var count int
	ts.Config.Handler = http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(201)
		writer.Write([]byte("pong" + strconv.Itoa(count)))
		count++
	})

	is := Inspect(ts)

	Requester(ts).Receive(requester.Get("/test"))
	Requester(ts).Receive(requester.Get("/test"))
	Requester(ts).Receive(requester.Get("/test"))

	drain := is.Drain()

	require.Len(t, drain, 3)
	assert.Equal(t, "pong1", drain[1].ResponseBody.String())
	require.Nil(t, is.LastExchange())
}

func TestInspector_Clear(t *testing.T) {
	ts := httptest.NewServer(nil)
	defer ts.Close()

	var count int
	ts.Config.Handler = http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(201)
		writer.Write([]byte("pong" + strconv.Itoa(count)))
		count++
	})

	is := Inspect(ts)

	Requester(ts).Receive(requester.Get("/test"))
	Requester(ts).Receive(requester.Get("/test"))
	Requester(ts).Receive(requester.Get("/test"))

	require.Len(t, is.Exchanges, 3)

	is.Clear()

	require.Empty(t, is.Exchanges)

	t.Run("nil", func(t *testing.T) {
		var i *Inspector
		assert.NotPanics(t, func() {
			i.Clear()
		})

	})
}

func TestInspector_readFrom(t *testing.T) {
	// fixed a bug in the hook func's ReadFrom hook.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(requester.HeaderContentType, requester.MediaTypeJSON)
		w.WriteHeader(201)
		readerFrom := w.(io.ReaderFrom)
		readerFrom.ReadFrom(strings.NewReader("pong"))
		readerFrom.ReadFrom(strings.NewReader("kilroy"))
	}))
	defer ts.Close()

	i := Inspect(ts)

	_, body, _ := Requester(ts).Receive(requester.Get("/test"), requester.Body("ping"))
	assert.Equal(t, "pongkilroy", string(body))
	assert.Equal(t, "pongkilroy", i.LastExchange().ResponseBody.String())
}

func TestInspect_nilhandler(t *testing.T) {

	ts := httptest.NewServer(nil)
	defer ts.Close()

	i := Inspect(ts)

	_, _, err := Requester(ts).Receive(nil)
	require.NoError(t, err)

	require.NotNil(t, i.LastExchange())
}

func ExampleInspector_Wrap() {
	mux := http.NewServeMux()
	// configure mux...

	i := NewInspector(0)

	ts := httptest.NewServer(i.Wrap(mux))
	defer ts.Close()
}

func ExampleInspector_NextExchange() {
	i := NewInspector(0)

	var h http.Handler = http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Write([]byte(`pong`))
	})

	h = i.Wrap(h)

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

	var h http.Handler = http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Write([]byte(`pong`))
	})

	h = i.Wrap(h)

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
