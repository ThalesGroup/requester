package requester

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func ExampleExpectSuccessCode() {
	mock := DoerFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 400,
			Body:       ioutil.NopCloser(strings.NewReader("bad format")),
		}, nil
	})

	resp, body, err := Receive(
		Get("/profile"),
		WithDoer(mock),
		ExpectSuccessCode(),
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
	mock := DoerFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 400,
			Body:       ioutil.NopCloser(strings.NewReader("bad format")),
		}, nil
	})

	resp, body, err := Receive(
		Get("/profile"),
		WithDoer(mock),
		ExpectCode(201),
	)

	fmt.Println(resp.StatusCode)
	fmt.Println(string(body))
	fmt.Println(err.Error())
	// Output:
	// 400
	// bad format
	// server returned unexpected status code.  expected: 201, received: 400
}
