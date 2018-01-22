package requester_test

import (
	"fmt"
	"github.com/gemalto/requester"
	"net/http"
	"net/http/httptest"
)

func Example() {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"color":"red"}`))
	}))
	defer s.Close()

	resp, body, err := requester.Receive(nil,
		requester.Get(s.URL),
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%d %s", resp.StatusCode, body)
	// Output: 200 {"color":"red"}

}

func Example_receiveResource() {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"color":"red"}`))
	}))
	defer s.Close()

	respStruct := struct {
		Color string `json:"color"`
	}{}

	resp, body, err := requester.Receive(&respStruct,
		requester.Get(s.URL),
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%d %s %v", resp.StatusCode, body, respStruct)
	// Output: 200 {"color":"red"} {red}
}
