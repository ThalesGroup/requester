package requester_test

import (
	"fmt"
	"github.com/gemalto/requester"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
)

func Example() {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"color":"red"}`))
	}))
	defer s.Close()

	resp, body, _ := requester.Receive(nil,
		requester.Get(s.URL),
	)

	fmt.Println(resp.StatusCode)
	fmt.Println(string(body))
	// Output:
	// 200
	// {"color":"red"}

}

func Example_receive() {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"color":"red"}`))
	}))
	defer s.Close()

	respStruct := struct {
		Color string
	}{}

	requester.Receive(&respStruct,
		requester.Get(s.URL),
	)

	fmt.Println(respStruct.Color)
	// Output: red
}

func ExampleExpectSuccessCode() {

	resp, body, err := requester.Receive(
		requester.Get("/profile"),
		requester.MockDoer(400, requester.Body("bad format")),
		requester.ExpectSuccessCode(),
	)

	fmt.Println(resp.StatusCode)
	fmt.Println(string(body))
	fmt.Println(err.Error())
	// Output:
	// 400
	// bad format
	// server returned an unsuccessful status code: 400
}

func ExampleExpectCode() {
	var mockDoer requester.DoerFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 400,
			Body:       ioutil.NopCloser(strings.NewReader("bad format")),
		}, nil
	}

	resp, body, err := requester.Receive(
		requester.Get("/profile"),
		mockDoer,
		requester.ExpectCode(201),
	)

	fmt.Println(resp.StatusCode)
	fmt.Println(string(body))
	fmt.Println(err.Error())
	// Output:
	// 400
	// bad format
	// server returned unexpected status code.  expected: 201, received: 400
}

// Inspector is an Option which captures requests and responses and their bodies.  It's
// a tool for writing tests.
func ExampleInspector() {
	var mockDoer requester.DoerFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 201,
			Body:       ioutil.NopCloser(strings.NewReader("pong")),
		}, nil
	}

	i := requester.Inspector{}

	requester.Receive(
		mockDoer,
		requester.Header(requester.HeaderAccept, requester.MediaTypeTextPlain),
		requester.Body("ping"),
		&i,
	)

	fmt.Println(i.Request.Header.Get(requester.HeaderAccept))
	fmt.Println(i.RequestBody.String())
	fmt.Println(i.Response.StatusCode)
	fmt.Println(i.ResponseBody.String())

	// Output:
	// text/plain
	// ping
	// 201
	// pong
}
