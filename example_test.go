package requester_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gemalto/requester"
	"github.com/go-errors/errors"
	"io/ioutil"
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

var requestBody struct {
	Color string `json:"color"`
}

type Resource struct {
	ID    string `json:"id"`
	Color string `json:"color"`
}

var ctx context.Context

func Example_native() {
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("POST", "http://api.com/resources/", bytes.NewReader(bodyBytes))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != 201 {
		panic(errors.New("expected code 201"))
	}

	respBody, _ := ioutil.ReadAll(resp.Body)
	var r Resource
	if err := json.Unmarshal(respBody, &r); err != nil {
		panic(err)
	}

	fmt.Printf("%d %s %v", resp.StatusCode, string(respBody), r)
}

func Example_demo() {
	var r Resource

	resp, body, err := requester.ReceiveContext(ctx, &r,
		requester.JSON(false),
		requester.Body(requestBody),
		requester.Post("http://api.com/resources/"),
		requester.ExpectCode(201),
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%d %s %v", resp.StatusCode, string(body), r)
}
