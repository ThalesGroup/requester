# Requests [![Build Status](https://travis-ci.org/ansel1/requests.svg?branch=master)](https://travis-ci.org/ansel1/requests) [![GoDoc](https://godoc.org/github.com/ansel1/requests?status.png)](https://godoc.org/github.com/ansel1/requests) [![Go Report Card](https://goreportcard.com/badge/github.com/ansel1/requests)](https://goreportcard.com/report/github.com/ansel1/requests)

A.K.A "Yet Another Golang Requests Package"

EXPERIMENTAL:  Until the version reaches 1.0, the API may change.

Requests makes it a bit simpler to use Go's `http` package as a client.  As an example, take
a simple request, with the `http` package:

```go
req, err := http.NewRequest("GET", "http://www.google.com", nil)
if err != nil { return err }

resp, err := http.DefaultClient.Do(req)
if err != nil { return err }

defer resp.Body.Close()
if resp.StatusCode == 200 {
    respBody, _ := ioutil.ReadAll(resp.Body)
    fmt.Println(string(respBody))
}
```
    
With `requests`:

```go
resp, body, err := requests.Receive(nil, requests.Get("http://www.google.com"))
if err != nil { return err }

fmt.Printf("%d %s", resp.StatusCode, body)
```
    
For a more complex use case, take an API call, which sends and receives JSON.  
With `http`:

```go
bodyBytes, err := json.Marshal(reqBodyStruct)
if err != nil { return err }

req, err := http.NewRequest("POST", "http://api.com/resources/", bytes.NewReader(bodyBytes))
if err != nil { return err }
req.Header.Set("Content-Type", "application/json")
req.Header.Set("Accept", "application/json"

resp, err := http.DefaultClient.Do(req)
if err != nil { return err }

if resp.StatusCode == 201 {
    respBody, _ := ioutil.ReadAll(resp.Body)
    var respStruct Resource
    err := json.Unmarshal(respBody, &respStruct)
    if err != nil { return err }
    
    fmt.Println(respBody)
}
```
    
With `requests`:

```go
var respStruct Resource

resp, body, err := requests.Receive(&respStruct, 
    requests.JSON(),
    requests.Get("http://www.google.com")
)
if err != nil { return err }
    
fmt.Printf("%d %s %v", resp.StatusCode, body, respStruct)
```
    
Requests revolves around the use of `Option`s, which are arguments to the functions which
create and/or send requests.  Options can be be used to set headers, query parameters, compose
the URL, set the method, install middleware, configure or replace the HTTP client used
to send the request, etc.

The package-level `Request(...Option)` function creates an (unsent) `*http.Request`:

```go
req, err := requests.Request(Get("http://www.google.com"))
```
    
The `Do(...Option)` function both creates the request and sends it, using the `http.DefaultClient`:

```go
resp, err := requests.Do(Get("http://www.google.com"))
```
    
A raw `*http.Response` is returned.  It is the
caller's responsibility to close the response body, just as with `http.Client`'s `Do()` method.

The package also has `Receive(interface{}, ...Option)` and `ReceiveFull(interface{},interface{},...Option)` 
functions which handle reading the response and optionally unmarshaling the body into a struct:

```go
var user User
resp, body, err := requests.Receive(&user, requests.Get("http://api.com/users/bob"))
```
    
The `Receive*()` functions read and close the response body for you, return the entire response
body as a string, and optionally unmarshal the response body into a struct.  If you only want
the body back as a string, pass nil as the first argument.  The string body is returned either way.
Requests can handle JSON and XML response bodies, as determined by the response's `Content-Type` 
header.  Other types of bodies can be handled by using a custom `Unmarshaler`.

If you have an API which returns structured non-2XX responses (like an error response JSON
body), you can use the `ReceiveFull()` function to pass an alternate struct value to unmarhal the 
error response body into:

```go
 var user User
 var apiError APIError
 resp, body, err := requests.ReceiveFull(&user, &apiError, requests.Get("http://api.com/users/bob"))
```
     
All these functions have `*Context()` variants, which add a `context.Context` to the request.  This is
particularly useful for setting a request timeout:

```go
ctx = context.WithTimeout(ctx, 10 * time.Second)
requests.RequestContext(ctx, Get("http://www.google.com"))
requests.DoContext(ctx, Get("http://www.google.com"))
requests.ReceiveContext(ctx, &into, Get("http://www.google.com"))
requests.ReceiveFullContext(ctx, &into, &apierr, Get("http://www.google.com"))
```
    
# Requests Instance

The package level functions just delegate to the `DefaultRequests` variable, which holds 
a `Requests` instance.  An instance of `Requests` is useful for building a re-usable, composable
HTTP client.

A new `Requests` can be constructed with `New(...Options)`:

```go
reqs, err := requests.New(
    requests.Get("http://api.server/resources/1"), 
    requests.JSON(), 
    requests.Accept(requests.ContentTypeJSON)
)
```

...or can be created with a literal:

```go
u, err := url.Parse("http://api.server/resources/1")
if err != nil { return err }

reqs := &Requests{
    URL: u,
    Method: "GET",
    Header: http.Header{
        requests.HeaderContentType: []string{requests.ContentTypeJSON),
        requests.HeaderAccept: []string{requests.ContentTypeJSON),
    },
}
```
    
Additional options can be applied with Apply():

```go
err := reqs.Apply(requests.Method("POST"), requests.Body(bodyStruct))
if err != nil { return err }
```
    
...or can just be set directly:

```go
reqs.Method = "POST"
reqs.Body = bodyStruct
```

`Requests` can be cloned, creating a copy which can be further configured:

```go
base, _ := requests.New(
    requests.URL("https://api.com"), 
    requests.JSON(),
    requests.Accept(requests.ContentTypeJSON),
    requests.BearerAuth(token),
)
    
getResource = base.Clone()
getResource.Apply(requests.RelativeURL("resources/1"))
```
    
`With(...Option)` combines `Clone()` and `Apply(...Option)`:

```go
getResource, _ := base.With(requests.RelativeURL("resources/1"))
```
    
Options can also be passed to the Request/Do/Receive* methods.  These Options will only
be applied to the particular request, not the `Requests` instance:

```go
resp, body, err := base.Receive(nil, requests.Get("resources", "1")  // path elements
                                                                     // will be joined.
```
                                                      
# Request Options

The `Requests` struct has attributes mirror counterparts on `*http.Request`:

```go
Method string
URL    *url.URL
Header http.Header
GetBody          func() (io.ReadCloser, error)
ContentLength    int64
TransferEncoding []string
Close            bool
Host             string
Trailer          http.Header
```
    
If not set, the constructed `*http.Request`s will have the normal default values these
attributes have after calling `http.NewRequest()` (some attributes will be initialized,
some remained zeroed).

If set, then the `Requests`' values will overwrite the values of these attributes in the 
`*http.Request`.

Functional `Options` are defined which set most of these attributes.  You can configure
`Requests` either by applying `Option`s, or by simply setting the attributes directly.

# Client Options

The HTTP client used to execute requests can also be customized through options:

```go
requests.Do(requests.Get("https://api.com"), requests.Client(clients.SkipVerify()))
```
    
`github.com/ansel1/requests/clients` is a standalone package for constructing and configuring
`http.Client`s.  The `requests.Client(...clients.Option)` option constructs a new HTTP client
and installs it into `Requests.Doer`.

# Query Params

The `QueryParams` attribute will be merged into any query parameters encoded into the 
URL.  For example:

```go
reqs, _ := requests.New(requests.URL("http://test.com?color=red"))
reqs.QueryParams = url.Values("flavor":[]string{"vanilla"})
r, _ := reqs.Request()
r.URL.String()             // http://test.com?color=red&flavor=vanilla
```
    
The `QueryParams()` option can take a `map[string]interface{}` or `url.Values`, or accepts
a struct value, which is marshaled into a `url.Values` using `github.com/google/go-querystring`:

```go
type Params struct {
    Color string `url:"color"`
}

reqs, _ := requests.New(
    requests.URL("http://test.com"),
    requests.QueryParams(Params{Color:"blue"}),
    requests.QueryParams(map[string][]string{"flavor":[]string{"vanilla"}}),
)
r, _ := reqs.Request()
r.URL.String()             // http://test.com?color=blue,flavor=vanilla
```
    
# Body

If `Requests.Body` is set to a `string`, `[]byte`, or `io.Reader`, the value will
be used directly as the request body:

```go
req, _ := requests.Request(
    requests.Post("http://api.com"),
    requests.ContentType(requests.ContentTypeJSON),
    requests.Body(`{"color":"red"}`),
)
httputil.DumpRequest(req, true)

// POST / HTTP/1.1
// Host: api.com
// Content-Type: application/json
// 
// {"color":"red"}
```
    
If `Body` is any other value, it will be marshaled into the body, using the 
`Marshaler`:

```go
type Resource struct {
    Color string `json:"color"`
}

req, _ := requests.Request(
    requests.Post("http://api.com"),
    requests.Body(Resource{Color:"red"}),
)
httputil.DumpRequest(req, true)

// POST / HTTP/1.1
// Host: api.com
// Content-Type: application/json
// 
// {"color":"red"}
```
    
Note the default marshaler is JSON, and sets the `Content-Type` header.

# Receive

`Receive()` handles the response as well:

```go
type Resource struct {
    Color string `json:"color"`
}

var res Resource

resp, body, err := requests.Receive(&res, requests.Get("http://api.com/resources/1")
if err != nil { return err }

fmt.Println(body)     // {"color":"red"}
```
    
The body of the response is returned as a string.  If the first argument is not nil, the body will
also be unmarshaled into that value.

By default, the unmarshaler will use the response's `Content-Type` header to determine how to unmarshal
the response body into a struct.  This can be customized by setting `Requests.Unmarshaler`:

```go
reqs.Unmarshaler = &requests.XML(true)                  // via assignment
reqs.Apply(requests.Unmarshaler(&requests.XML(true)))   // or via an Option
```
        
# Doer and Middleware

`Requests` uses an implementation of `Doer` to execute requests.  By default, `http.DefaultClient` is used, 
but this can be replaced by a customize client, or a mock `Doer`:

```go
reqs.Doer = requests.DoerFunc(func(req *http.Request) (*http.Response, error) {
    return &http.Response{}
})
```
    
You can also install middleware into `Requests`, which can intercept the request and response:

```go
mw := func(next requests.Doer) requests.Doer {
    return requests.DoerFunc(func(req *http.Request) (*http.Response, error) {
        fmt.Println(httputil.DumpRequest(req, true))
        resp, err := next(req)
        if err == nil { 
            fmt.Println(httputil.DumpResponse(resp, true))
        }
        return resp, err
    })
}
reqs.Middleware = append(reqs.Middleware, mw)   // via assignment
reqs.Apply(requests.Use(mw))                    // or via option
```
    
# FAQ

- Why, when there are like, 50 other packages that do the exact same thing?

Yeah, good question.  This library started as a few tweaks to `https://github.com/dghubble/sling`.  Then
it became more of a fork, then a complete rewrite, inspiration from a bunch of other similar libraries.

A few things bugged me about other libraries:

1. Some didn't offer enough control over the base `http` primitives, like the underlying
   `http.Client`, and all the esoteric attributes of `http.Request`.
2. I wanted more control over marshaling and unmarshaling bodies, without sacrificing access
   to the raw body.
3. Some libraries which offer lots more control or options also seemed to be much more 
   complicated, or less idiomatic.
4. Most libraries don't handle `context.Context`s at all.
5. The main thing: most other libraries use a "fluent" API, where you call methods on a builder
   instance to configure the request, and these methods each return the builder, making it
   simple to call the next method, something like this:
   
        req.Get("http://api.com").Header("Content-Type", "application/json").Body(reqBody)
        
   I used to like fluent APIs in other languages, but they don't feel right in Go.  You typically
   end up deferring errors until later, so the error doesn't surface near the code that caused
   the error.  Its difficult to mix fluent APIs with interfaces, because the concrete types
   tend to have lots of methods, and they all have to return the same concrete type.  For the 
   same reason, it's awkward to embed types with fluent APIs.  Fluent APIs also make it hard to extend
   the library with additional, external options.
   
`Requests` swaps a fluent API for the
 [functional option pattern](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis).
This hopefully keeps a fluent-like coding style, while being more idiomatic Go.  Since Options are just
a simple interface, it's easy to bring your own options, or contribute new options back
to this library.

Also, making the options into objects improved ergonomics in a few places, like mirroring 
the main functions (`Request()`, `Do()`, `Receive()`) on the struct and the package.  Options can be passed
around as arguments or accumulated in slices.


