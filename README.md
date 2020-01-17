# Requester
[![GoDoc](https://godoc.org/github.com/gemalto/requester?status.png)](https://godoc.org/github.com/gemalto/requester) [![Go Report Card](https://goreportcard.com/badge/github.com/gemalto/requester)](https://goreportcard.com/report/github.com/gemalto/requester) [![Build](https://github.com/gemalto/requester/workflows/Build/badge.svg)](https://github.com/gemalto/requester/actions?query=branch%3Amaster+workflow%3ABuild+)

A.K.A "Yet Another Golang Requests Package"

Requester makes it a bit simpler to use Go's `http` package as a client.  As an example, take
a simple request, with the `http` package:

```go
bodyBytes, err := json.Marshal(reqBodyStruct)
if err != nil { return err }

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
```
    
`requester` uses functional options to configure the request, and folds request building, 
execution, and response handling into a single call:

```go
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
```

# Features

- Functional option pattern supports an ergonomic API
- Options for configuring http.Client options
- Tools for writing unit tests, like Inspector{}, MockDoer(), MockHandler(), and the httptestutil package
- Embeddable
- Client-side middleware (see Middleware and Doer)
- `context.Context` support
    
# Core API

The core functions are available on the package, or on instances of Requester{}.

```go
// just build a request
Request(...Option) (*http.Request, error)
RequestContext(context.Context, ...Option) (*http.Request, error)

// build a request and execute it
Send(...Option) (*http.Response, error)
SendContext(context.Context, ...Option) (*http.Response, error)

// build and send a request, and handle the response
Receive(interface{}, ...Option) (*http.Response, []byte, error)
ReceiveContext(context.Context, interface{}, ...Option) (*http.Response, []byte, error)
```
    
`Receive/ReceiveContext` reads and closes the response body, returns it as a byte slice, 
and also attempts to unmarshal it into a target value, if one is provided. 

Each of these accept variadic functional options, which alter the request,
the http.Client, or the control how to process the response. `Option` is defined as:

```go
type Option interface {
   Apply(*Requester) error
}
```

# FAQ

- Why, when there are like, 50 other packages that do the exact same thing?

Yeah, good question.  This library started as a few tweaks to `https://github.com/dghubble/sling`.  Then
it became more of a fork, then a complete rewrite, inspired by a bunch of other similar libraries.

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
   
`Requester` swaps a fluent API for the
 [functional option pattern](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis).
This hopefully keeps a fluent-like coding style, while being more idiomatic Go.  Since Options are just
a simple interface, it's easy to bring your own options, or contribute new options back
to this library.

Also, making the options into objects improved ergonomics in a few places, like mirroring 
the main functions (`Request()`, `Send()`, `Receive()`) on the struct and the package.  Options can be passed
around as arguments or accumulated in slices.
