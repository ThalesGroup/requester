/*
Package requester is a Go library for building and sending HTTP requests.  It's a thin wrapper around the
http package, which makes it a little easier to create and send requests, and to handle responses.

Requester revolves around the use of functional Options.  Options can be be used to configure aspects
of the request, like headers, query params, the URL, etc.  Options can also configure the client used to
send requests.

The central functions are Request(), Send(), and Receive():

	// Create a request
	req, err := requester.Request(
		requester.JSON(),
		requester.Get("http://api.com/users/bob"),
	)

	// Create and send a request
	resp, err := requester.Send(
		requester.JSON(),
		requester.Non2XXResponseAsError(),
		requester.Get("http://api.com/users/bob"),
	)

	// Create and send a request, and handle the response
	var user User
	resp, body, err := requester.Receive(&user,
		requester.JSON(),
		requester.Non2XXResponseAsError(),
		requester.BasicAuth("user", "password"),
		requester.Client(httpclient.NoRedirects()),
		requester.DumpToStandardOut(),
		requester.Get("http://api.com/users/bob"),
	)

	// Create a reusable Requester instance
	reqr, err := requester.New(
		requester.JSON(),
		requester.Non2XXResponseAsError(),
		requester.BasicAuth("user", "password"),
		requester.Client(httpclient.NoRedirects()),
		requester.DumpToStandardOut(),
	)

	// Requester instances have the same main methods as the package
	req, err := reqr.Request(
		requester.Get("http://api.com/users/bob")
	)

	resp, err := reqr.Send(
		requester.Get("http://api.com/users/bob")
	)

	resp, body, err := reqr.Receive(&user,
		requester.Get("http://api.com/users/bob")
	)

There are also *Context() variants as well:

	ctx = context.WithTimeout(ctx, 10 * time.Second)
	requester.RequestContext(ctx, requester.Get("http://api.com/users/bob"))
	requester.SendContext(ctx, requester.Get("http://api.com/users/bob"))
	requester.ReceiveContext(ctx, &into, requester.Get("http://api.com/users/bob"))

The attributes of the Requester control how it creates and sends requests, and how it
handles responses:

	type Requester struct {

		// These are copied directly into requests, if they contain
		// a non-zero value
		Method string
		URL    *url.URL
		Header http.Header
		GetBody          func() (io.ReadCloser, error)
		ContentLength    int64
		TransferEncoding []string
		Close            bool
		Host             string
		Trailer          http.Header

		// these are handled specially, see below
		QueryParams url.Values
		Body interface{}
		Marshaler Marshaler

		// these configure how to send requests
		Doer Doer
		Middleware []Middleware

		// this configures response handling
		Unmarshaler Unmarshaler
	}

These attributes can be modified directly, by assignment, or by applying Options.  Options
are simply functions which modify these attributes. For example:

	reqr, err := requester.New(
		requester.Method("POST),
	)

is equivalent to

	reqr := &requester.Requester{
		Method: "POST",
	}

New Requesters can be constructed with New(...Options):

	reqs, err := requester.New(
		requester.Get("http://api.server/resources/1"),
		requester.JSON(),
		requester.Accept(requester.MediaTypeJSON)
	)

Additional options can be applied with Apply():

	err := reqs.Apply(
		requester.Method("POST"),
		requester.Body(bodyStruct),
	)

...or the Requester's members can just be set directly:

	reqs.Method = "POST"
	reqs.Body = bodyStruct

Requesters can be cloned, creating a copy which can be further configured:

	base, err := requester.New(
		requester.URL("https://api.com"),
		requester.JSON(),
		requester.Accept(requester.MediaTypeJSON),
		requester.BearerAuth(token),
	)

	clone = base.Clone()

	err = clone.Apply(
		requester.Get("resources/1"),
	)

With(...Option) combines Clone() and Apply(...Option):

	clone, err := base.With(
		requester.Get("resources/1"),
	)

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

Query Params

Requester.QueryParams will be merged into any query parameters encoded into the
URL.  For example:

	reqs, _ := requester.New(
		requester.URL("http://test.com?color=red"),
	)

	reqs.QueryParams = url.Values("flavor":[]string{"vanilla"})

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

Note the default marshaler is JSON, and sets the Content-Type header.

Receive

Receive() handles the response as well:

	type Resource struct {
		Color string `json:"color"`
	}

	var res Resource

	resp, body, err := requester.Receive(&res,
		requester.Get("http://api.com/resources/1",
	)

	fmt.Println(body)     // {"color":"red"}

The body of the response is returned.  Even in cases where an error is returned, the body
and the response will be returned as well, if available.  This is helpful when middleware
which validates aspects of the response generates an error, but the calling code still needs
to inspect the contents of the body (e.g. for an error message).

If the first argument is not nil, the body will also be unmarshaled into that value.  By default, the unmarshaler
will use the response's Content-Type header to determine how to unmarshal
the response body into a struct.  This can be customized by setting Requester.Unmarshaler:

	reqs.Unmarshaler = &requester.XMLMarshaler(Indent:true)                  // via assignment
	reqs.Apply(requester.WithUnmarshaler(&requester.XMLMarshaler(Indent:true)))   // or via an Option

Doer and Middleware

Requester uses a Doer to execute requests, which is an interface.  By default, http.DefaultClient is used,
but this can be replaced by a customize client, or a mock Doer:

	reqs.Doer = requester.DoerFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{}
	})

Requester itself is a Doer, so it can be nested in other Requester or composed with other packages
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
	reqs.Middleware = append(reqs.Middleware, mw)   // via assignment
	reqs.Apply(requester.Use(mw))                    // or via option
*/
package requester
