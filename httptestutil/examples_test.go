package httptestutil_test

import (
	"fmt"
	"github.com/gemalto/requester"
	"github.com/gemalto/requester/httptestutil"
	"net/http"
	"net/http/httptest"
	"strconv"
)

func Example() {

	mux := http.NewServeMux()
	mux.Handle("/echo", requester.MockHandler(201, requester.Body("pong")))

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// inspect server traffic
	is := httptestutil.Inspect(ts)

	// construct a pre-configured Requester
	r := httptestutil.Requester(ts)
	resp, body, _ := r.Receive(requester.Get("/echo"), requester.Body("ping"))

	ex := is.LastExchange()
	fmt.Println("server received: " + ex.RequestBody.String())
	fmt.Println("server sent: " + strconv.Itoa(ex.StatusCode))
	fmt.Println("server sent: " + ex.ResponseBody.String())
	fmt.Println("client received: " + strconv.Itoa(resp.StatusCode))
	fmt.Println("client received: " + string(body))

	// Output:
	// server received: ping
	// server sent: 201
	// server sent: pong
	// client received: 201
	// client received: pong
}
