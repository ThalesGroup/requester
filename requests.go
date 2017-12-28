package requests

import (
	"bytes"
	"context"
	"github.com/ansel1/merry"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// Requests is an HTTP Request builder and sender.
//
// A Requests object can be used to construct *http.Requests,
// send requests via a configurable HTTP client, and unmarshal
// the response.  A Requests object is configured by setting
// its members, which in most cases mirror the members of
// *http.Request, or by applying Options, using the Apply()
// and With() methods.
//
// Once configured, you can use Requests solely as a *http.Request
// factory, by calling Request() or RequestContext().
//
// Or you can use the Requests
// to construct and send requests (via a configurable Doer) and
// get back the raw *http.Response, with the Do() and
// DoContext() methods.
//
// Or you can have Requests also read the response body and
// unmarshal it into a struct, with Receive(), ReceiveContext(),
// ReceiveFull(), and ReceiveFullContext().
//
// A Requests object can be constructed as a literal:
//
//     r := requests.Requests{
//              URL:    u,
//              Method: "POST",
//              Body:   b,
//          }
//
// ...or via the New() constructor:
//
//     reqs, err := requests.New(requests.Post("http://test.com/red"), requests.Body(b))
//
// Additional options can be applied with Apply():
//
//     err := reqs.Apply(Accept("application/json"))
//
// Requests can be cloned, to create an identically configured Requests object, which can
// then be further configured without affecting the parent:
//
//     reqs2 := reqs.Clone()
//     err := reqs2.Apply(Header("X-Frame","1"))
//
// With() is equivalent to Clone() and Apply():
//
//     reqs2, err := reqs.With(Header("X-Frame","1"))
//
// The remaining methods of Requests are for creating HTTP requests, sending them, and handling
// the responses: Request, Do, Receive, and ReceiveFull.
//
//     req, err        := reqs.Request()           // create a requests
//     resp, err       := reqs.Do()                // create and send a request
//
//     var m Resource
//     resp, body, err := reqs.Receive(&m)         // create and send request, read and unmarshal response
//
//     var e ErrorResponse
//     resp, body, err := reqs.ReceiveFull(&m, &e) // create and send request, read response, unmarshal 2XX responses
//                                                 // into m, and other responses in e
//
// Request, Do, Receive, and ReceiveFull all accept a varargs of Options, which will be applied
// only to a single request, not to the Requests object.
//
//     req, err 	   := reqs.Request(
//                        	Put("users/bob"),
//                          Body(bob),
//                        )
//
// RequestContext, DoContext, ReceiveContext, and ReceiveFullContext variants accept a context, which is
// attached to the constructed request:
//
//     req, err        := reqs.RequestContext(ctx)
//
type Requests struct {
	////////////////////////////////////////////////////////////////
	//                                                            //
	//  Attributes affecting the construction of http.Requests.   //
	//                                                            //
	////////////////////////////////////////////////////////////////

	// Method defaults to "GET".
	Method string
	URL    *url.URL

	// Header supplies the request headers.  If the Content-Type header
	// is explicitly set here, it will override the Content-Type header
	// supplied by the Marshaler.
	Header http.Header

	// advanced options, not typically used.  If not sure, leave them
	// blank.
	// Most of these settings are set automatically by the http package.
	// Setting them here will override the automatic values.
	GetBody          func() (io.ReadCloser, error)
	ContentLength    int64
	TransferEncoding []string
	Close            bool
	Host             string
	Trailer          http.Header

	// QueryParams are added to the request, in addition to any
	// query params already encoded in the URL
	QueryParams url.Values

	// Body can be set to a string, []byte, io.Reader, or a struct.
	// If set to a string, []byte, or io.Reader,
	// the value will be used as the body of the request.
	// If set to a struct, the Marshaler
	// will be used to marshal the value into the request body.
	Body interface{}

	// Marshaler will be used to marshal the Body value into the body
	// of requests.  It is only used if Body is a struct value.
	// Defaults to the DefaultMarshaler, which marshals to JSON.
	//
	// If no Content-Type header has been explicitly set in Requests.Header, the
	// Marshaler will supply an appropriate one.
	Marshaler BodyMarshaler

	//////////////////////////////////////////////////////////////
	//
	//  Attributes related to sending requests and handling
	//  responses.
	//
	////////////////////////////////////////////////////////////////

	// Doer holds the HTTP client for used to execute requests.
	// Defaults to http.DefaultClient.
	Doer Doer

	// Middleware wraps the Doer.  Middleware will be invoked in the order
	// it is in this slice.
	Middleware []Middleware

	// Unmarshaler will be used by the Receive methods to unmarshal
	// the response body.  Defaults to DefaultUnmarshaler, which unmarshals
	// multiple content types based on the Content-Type response header.
	Unmarshaler BodyUnmarshaler
}

// New returns a new Requests.
func New(options ...Option) (*Requests, error) {
	b := &Requests{}
	err := b.Apply(options...)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	return b, nil
}

func cloneURL(url *url.URL) *url.URL {
	if url == nil {
		return nil
	}
	urlCopy := *url
	return &urlCopy
}

func cloneValues(v url.Values) url.Values {
	if v == nil {
		return nil
	}
	v2 := make(url.Values, len(v))
	for key, value := range v {
		v2[key] = value
	}
	return v2
}

func cloneHeader(h http.Header) http.Header {
	if h == nil {
		return nil
	}
	h2 := make(http.Header)
	for key, value := range h {
		h2[key] = value
	}
	return h2
}

// Clone returns a deep copy of a Requests.  Useful inheriting and adding settings from
// a parent Requests without modifying the parent.  For example,
//
//     parent, _ := requests.New(Get("https://api.io/"))
//     foo := parent.Clone()
//     foo.Apply(Get("foo/"))
// 	   bar := parent.Clone()
//     bar.Apply(Post("bar/"))
//
// foo and bar will both use the same client, but send requests to
// https://api.io/foo/ and https://api.io/bar/ respectively.
func (r *Requests) Clone() *Requests {
	s2 := *r
	s2.Header = cloneHeader(r.Header)
	s2.Trailer = cloneHeader(r.Trailer)
	s2.URL = cloneURL(r.URL)
	s2.QueryParams = cloneValues(r.QueryParams)
	return &s2
}

// Request returns a new http.Request.
//
// If Options are passed, they will only by applied to this single request.
//
// If r.Body is a struct, it will be marshaled into the request body using
// r.Marshaler.  The Marshaler will also set the Content-Type header, unless
// this header is already explicitly set in r.Header.
//
// If r.Body is an io.Reader, string, or []byte, it is set as the request
// body directly, and no default Content-Type is set.
//
func (r *Requests) Request(opts ...Option) (*http.Request, error) {
	return r.RequestContext(context.Background(), opts...)
}

// RequestContext does the same as Request, but requires a context.  Use this
// to set a request timeout:
//
//     req, err := r.RequestContext(context.WithTimeout(context.Background(), 10 * time.Seconds))
//
func (r *Requests) RequestContext(ctx context.Context, opts ...Option) (*http.Request, error) {
	reqs := r
	if len(opts) > 0 {
		var err error
		reqs, err = reqs.With(opts...)
		if err != nil {
			return nil, err
		}
	}
	// marshal body, if applicable
	bodyData, ct, err := reqs.getRequestBody()
	if err != nil {
		return nil, err
	}

	urlS := ""
	if reqs.URL != nil {
		urlS = reqs.URL.String()
	}

	req, err := http.NewRequest(reqs.Method, urlS, bodyData)
	if err != nil {
		return nil, err
	}

	// if we marshaled the body, use our content type
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}

	if reqs.ContentLength != 0 {
		req.ContentLength = reqs.ContentLength
	}

	if reqs.GetBody != nil {
		req.GetBody = reqs.GetBody
	}

	// copy the host
	if reqs.Host != "" {
		req.Host = reqs.Host
	}

	req.TransferEncoding = reqs.TransferEncoding
	req.Close = reqs.Close
	req.Trailer = reqs.Trailer

	// copy Headers pairs into new Header map
	for k, v := range reqs.Header {
		req.Header[k] = v
	}

	if len(reqs.QueryParams) > 0 {
		if req.URL.RawQuery != "" {
			req.URL.RawQuery += "&" + reqs.QueryParams.Encode()
		} else {
			req.URL.RawQuery = reqs.QueryParams.Encode()
		}
	}

	return req.WithContext(ctx), nil
}

// getRequestBody returns the io.Reader which should be used as the body
// of new Requests.
func (r *Requests) getRequestBody() (body io.Reader, contentType string, err error) {
	switch v := r.Body.(type) {
	case nil:
		return nil, "", nil
	case io.Reader:
		return v, "", nil
	case string:
		return strings.NewReader(v), "", nil
	case []byte:
		return bytes.NewReader(v), "", nil
	default:
		marshaler := r.Marshaler
		if marshaler == nil {
			marshaler = DefaultMarshaler
		}
		b, ct, err := marshaler.Marshal(r.Body)
		if err != nil {
			return nil, "", err
		}
		return bytes.NewReader(b), ct, err
	}
}

// Do executes a request with the Doer.  The response body is not closed:
// it is the caller's responsibility to close the response body.
// If the caller prefers the body as a byte slice, or prefers the body
// unmarshaled into a struct, see the Receive methods below.
//
// Additional options arguments can be passed.  They will be applied to this request only.
func (r *Requests) Do(opts ...Option) (*http.Response, error) {
	return r.DoContext(context.Background(), opts...)
}

// DoContext does the same as Request, but requires a context.
func (r *Requests) DoContext(ctx context.Context, opts ...Option) (*http.Response, error) {
	// if there are request options, apply them now, rather than passing them
	// to RequestContext().  Options may modify the Middleware or the Doer, and
	// we want to honor those options as well as the ones which affect the request.
	reqs := r
	if len(opts) > 0 {
		var err error
		reqs, err = reqs.With(opts...)
		if err != nil {
			return nil, err
		}
	}
	req, err := reqs.RequestContext(ctx)
	if err != nil {
		return nil, err
	}
	doer := reqs.Doer
	if doer == nil {
		doer = http.DefaultClient
	}
	return Wrap(doer, reqs.Middleware...).Do(req)
}

// Receive creates a new HTTP request and returns the response. Success
// responses (2XX) are unmarshaled into successV.
// Any error creating the request, sending it, or decoding a 2XX response
// is returned.
//
// If option arguments are passed, they are applied to this single request only.
func (r *Requests) Receive(successV interface{}, opts ...Option) (resp *http.Response, body string, err error) {
	return r.ReceiveFullContext(context.Background(), successV, nil, opts...)
}

// ReceiveContext does the same as Receive, but requires a context.
func (r *Requests) ReceiveContext(ctx context.Context, successV interface{}, opts ...Option) (resp *http.Response, body string, err error) {
	return r.ReceiveFullContext(ctx, successV, nil, opts...)
}

// ReceiveFull creates a new HTTP request and returns the response. Success
// responses (2XX) are unmarshaled into successV and
// other responses are unmarshaled into failureV.
// Any error creating the request, sending it, or decoding the response is
// returned.
func (r *Requests) ReceiveFull(successV, failureV interface{}, opts ...Option) (resp *http.Response, body string, err error) {
	return r.ReceiveFullContext(context.Background(), successV, failureV, opts...)

}

// ReceiveFullContext does the same as ReceiveFull
func (r *Requests) ReceiveFullContext(ctx context.Context, successV, failureV interface{}, opts ...Option) (resp *http.Response, body string, err error) {
	resp, err = r.DoContext(ctx, opts...)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return
	}

	bodyS, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	body = string(bodyS)

	var unmarshalInto interface{}
	if code := resp.StatusCode; 200 <= code && code <= 299 {
		unmarshalInto = successV
	} else {
		unmarshalInto = failureV
	}

	if unmarshalInto != nil {
		unmarshaler := r.Unmarshaler
		if unmarshaler == nil {
			unmarshaler = DefaultUnmarshaler
		}

		err = unmarshaler.Unmarshal(bodyS, resp.Header.Get("Content-Type"), unmarshalInto)
	}
	return
}
