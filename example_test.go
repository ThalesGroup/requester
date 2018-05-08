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

	resp, body, _ := requester.Receive(
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
