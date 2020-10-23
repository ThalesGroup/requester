package requester_test

import (
	"fmt"
	. "github.com/ThalesGroup/requester"
	"github.com/ThalesGroup/requester/httpclient"
	"github.com/ThalesGroup/requester/httptestutil"
	"net/http"
	"net/http/httptest"
	"os"
	"time"
)

func Example() {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"color":"red"}`))
	}))
	defer s.Close()

	resp, body, _ := Receive(
		Get(s.URL),
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

	r := struct {
		Color string `json:"color"`
	}{}

	Receive(&r, Get(s.URL))

	fmt.Println(r.Color)
	// Output: red
}

func Example_everything() {
	type Resource struct {
		ID    string `json:"id"`
		Color string `json:"color"`
	}

	s := httptest.NewServer(MockHandler(201,
		JSON(true),
		Body(&Resource{Color: "red", ID: "123"}),
	))
	defer s.Close()

	r := httptestutil.Requester(s,
		Post("/resources?size=big"),
		BearerAuth("atoken"),
		JSON(true),
		Body(&Resource{Color: "red"}),
		ExpectCode(201),
		Header("X-Request-Id", "5"),
		QueryParam("flavor", "vanilla"),
		QueryParams(&struct {
			Type string `url:"type"`
		}{Type: "upload"}),
		Client(
			httpclient.SkipVerify(true),
			httpclient.Timeout(5*time.Second),
			httpclient.MaxRedirects(3),
		),
	)

	r.MustApply(DumpToStderr())
	httptestutil.Dump(s, os.Stderr)

	serverInspector := httptestutil.Inspect(s)
	clientInspector := Inspect(r)

	var resource Resource

	resp, body, err := r.Receive(&resource)
	if err != nil {
		panic(err)
	}

	fmt.Println("client-side request url path:", clientInspector.Request.URL.Path)
	fmt.Println("client-side request query:", clientInspector.Request.URL.RawQuery)
	fmt.Println("client-side request body:", clientInspector.RequestBody.String())

	ex := serverInspector.LastExchange()
	fmt.Println("server-side request authorization header:", ex.Request.Header.Get("Authorization"))
	fmt.Println("server-side request request body:", ex.RequestBody.String())
	fmt.Println("server-side request response body:", ex.ResponseBody.String())

	fmt.Println("client-side response body:", clientInspector.ResponseBody.String())

	fmt.Println("response status code:", resp.StatusCode)
	fmt.Println("raw response body:", string(body))
	fmt.Println("unmarshaled response body:", resource)

	// Output:
	// client-side request url path: /resources
	// client-side request query: flavor=vanilla&size=big&type=upload
	// client-side request body: {
	//   "id": "",
	//   "color": "red"
	// }
	// server-side request authorization header: Bearer atoken
	// server-side request request body: {
	//   "id": "",
	//   "color": "red"
	// }
	// server-side request response body: {
	//   "id": "123",
	//   "color": "red"
	// }
	// client-side response body: {
	//   "id": "123",
	//   "color": "red"
	// }
	// response status code: 201
	// raw response body: {
	//   "id": "123",
	//   "color": "red"
	// }
	// unmarshaled response body: {123 red}

}
