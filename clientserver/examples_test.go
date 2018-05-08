package clientserver_test

import (
	"fmt"
	"github.com/gemalto/requester"
	"github.com/gemalto/requester/clientserver"
	"net/http"
	"strconv"
)

func ExampleClientServer() {

	// NewServer creates an http test server and starts it (which is why it needs to be closed)
	cs := clientserver.NewServer(nil)
	defer cs.Close()

	// ClientServer has convenience functions for registering a handler, handler func, or using
	// a mux
	cs.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(201)
		writer.Write([]byte("Done!"))
	})

	// ClientServer can create a Requester, which is pre-wired to talk to itself
	resp, body, _ := cs.Requester().Receive(nil)

	fmt.Println(resp.StatusCode)
	fmt.Println(string(body))
	// Output:
	// 201
	// Done!
}

// Mux returns a non-nil *http.ServeMux, which is installed as the server's Handler.
func ExampleClientServer_Mux() {

	cs := clientserver.NewServer(nil)
	defer cs.Close()

	cs.Mux().HandleFunc("/test", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(204)
	})

	resp, _, _ := cs.Requester().Receive(requester.Get("/test"))

	fmt.Println(resp.StatusCode)

	// Output: 204

}

// ClientServer includes a requester.Inspector, which is automatically installed
// into the embedded Requester.  This allows tests to inspect the last request and
// response.  ClientServer also captures the last request received by the server
func ExampleClientServer_inspection() {

	cs := clientserver.NewServer(nil)
	defer cs.Close()

	cs.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(201)
		writer.Write([]byte("pong"))
	})

	ic := cs.InspectClient()
	is := cs.InspectServer()

	cs.Requester().Receive(
		requester.Header("Accept", "application/json"),
		requester.Body("ping"),
	)

	fmt.Println("client sent: " + ic.Request.Header.Get("Accept"))
	fmt.Println("client sent: " + ic.RequestBody.String())
	ex := is.LastExchange()
	fmt.Println("server received: " + ex.Request.Header.Get("Accept"))
	fmt.Println("server received: " + ex.RequestBody.String())
	fmt.Println("server sent: " + strconv.Itoa(ex.StatusCode))
	fmt.Println("server sent: " + ex.ResponseBody.String())
	fmt.Println("client received: " + strconv.Itoa(ic.Response.StatusCode))
	fmt.Println("client received: " + ic.ResponseBody.String())

	// If re-using the ClientServer between test cases, these capture elements can be cleared
	cs.Clear()

	// Output:
	// client sent: application/json
	// client sent: ping
	// server received: application/json
	// server received: ping
	// server sent: 201
	// server sent: pong
	// client received: 201
	// client received: pong

}
