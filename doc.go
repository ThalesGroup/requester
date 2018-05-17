/*
Package requester builds and executes HTTP requests.  It's a thin wrapper around the
http package with conveniences for configuring requests and processing responses.

The central, package-level functions are:

 Request(...Option) (*http.Request, error)
 Send(...Option) (*http.Response, error)
 Receive(interface{}, ...Option) (*http.Response, []byte, error)

Context-aware variants are also available.

requester.Requester{} has the same methods.  A Requester instance can be used to repeat a request,
as a template for similar requests across a REST API surface, or embedded in another type, as
the core of a language binding to a REST API.  The exported attributes of requester.Requester{}
control how it constructs requests, what client it uses to execute them, and how the responses
are handled.

Most methods and functions in the package accept Options, which are functions that configure the attributes
of Requesters.  The package provides many options for configuring most attributes Requester.

Receive

Receive() builds a request, executes it, and reads the response body.  If a target value is provided,
Receive will attempt to unmarshal the body into the target value.

	type Resource struct {
		Color string `json:"color"`
	}

	var res Resource

	resp, body, err := requester.Receive(&res,
		requester.Get("http://api.com/resources/1",
	)

	fmt.Println(body)      // {"color":"red"}
	fmt.Println(res.Color) // red

The body of the response, if present, is always returned as a []byte, even when unmarshaling or returning
an error.

By default, Receive uses the response's Content-Type header to determine how to unmarshal
the response body into a struct.  This can be customized by setting Requester.Unmarshaler:

	reqs.Unmarshaler = &requester.XMLMarshaler(Indent:true)

Query Params

Requester.QueryParams will be merged into any query parameters encoded into the
URL.  For example:

	reqs, _ := requester.New(
		requester.URL("http://test.com?color=red"),
	)

	reqs.Params().Set("flavor","vanilla")

	r, _ := reqs.Request()
	r.URL.String()
	// Output: http://test.com?color=red&flavor=vanilla

The QueryParams() option can take a map[string]string, a map[string]interface{}, a url.Values, or
a struct.  Structs are marshaled into url.Values using "github.com/google/go-querystring":

	type Params struct {
		Color string `url:"color"`
	}

	reqs, _ := requester.New(
		requester.URL("http://test.com"),
		requester.QueryParams(
			Params{Color:"blue"},
			map[string][]string{"flavor":[]string{"vanilla"}},
			map[string]string{"temp":"hot"},
			url.Values{"speed":[]string{"fast"}},
		),
		requester.QueryParam("volume","load"),
	)

	r, _ := reqs.Request()
	r.URL.String()
	// Output: http://test.com?color=blue,flavor=vanilla,temp=hot,speed=fast,volume=loud

Body

If Requester.Body is set to a string, []byte, or io.Reader, the value will
be used directly as the request body:

	req, _ := requester.Request(
		requester.Post("http://api.com"),
		requester.ContentType(requester.MediaTypeJSON),
		requester.Body(`{"color":"red"}`),
	)

	httputil.DumpRequest(req, true)

	// POST / HTTP/1.1
	// Host: api.com
	// Content-Type: application/json
	//
	// {"color":"red"}

If Body is any other value, it will be marshaled into the body, using the
Requester.Marshaler:

	type Resource struct {
		Color string `json:"color"`
	}

	req, _ := requester.Request(
		requester.Post("http://api.com"),
		requester.Body(Resource{Color:"red"}),
	)

	httputil.DumpRequest(req, true)

	// POST / HTTP/1.1
	// Host: api.com
	// Content-Type: application/json
	//
	// {"color":"red"}

Note the default Marshaler is JSON, and sets the request's Content-Type header.

HTTP Client Options

The HTTP client used to execute requests can also be customized with Options:

	import "github.com/gemalto/requester/httpclient"

	requester.Send(
		requester.Get("https://api.com"),
		requester.Client(httpclient.SkipVerify()),
	)

"github.com/gemalto/requester/httpclient" is a standalone package for constructing and configuring
http.Clients.  The requester.Client(...httpclient.Option) option constructs a new HTTP client
and installs it into Requester.Doer.

Doer and Middleware

Requester uses a Doer to execute requests, which is an interface.  By default, http.DefaultClient is used,
but this can be replaced by a customized client, or a mock Doer:

	reqs.Doer = requester.DoerFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{}
	})

Requester itself is a Doer, so it can be nested in another Requester or composed with other packages
that support Doers.

You can also install middleware into Requester, which can intercept the request and response:

	mw := func(next requester.Doer) requester.Doer {
		return requester.DoerFunc(func(req *http.Request) (*http.Response, error) {
			fmt.Println(httputil.DumpRequest(req, true))
			resp, err := next(req)
			if err == nil {
				fmt.Println(httputil.DumpResponse(resp, true))
			}
			return resp, err
		})
	}
	reqs.Middleware = append(reqs.Middleware, mw)
*/
package requester
