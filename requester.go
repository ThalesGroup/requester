package requester

import (
	"bytes"
	"context"
	"github.com/ansel1/merry"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Requester is an HTTP request builder and HTTP client.
//
// Requester can be used to construct requests,
// send requests via a configurable HTTP client, and unmarshal
// the response.  A Requester is configured by setting
// its members, which in most cases mirror the members of
// *http.Request.  A Requester can also be configured by applying
// functional Options, which simply modify Requester's members.
//
// A Requester can be constructed as a literal:
//
//     r := requester.Requester{
//              URL:    u,
//              Method: "POST",
//              Body:   b,
//          }
//
// ...or via the New() and MustNew() constructors, which take Options:
//
//     reqs, err := requester.New(requester.Post("http://test.com/red"), requester.Body(b))
//
// Additional options can be applied with Apply() and MustApply():
//
//     err := reqs.Apply(requester.Accept("application/json"))
//
// Requesters can be cloned.  The clone can
// then be further configured without affecting the parent:
//
//     reqs2 := reqs.Clone()
//     err := reqs2.Apply(Header("X-Frame","1"))
//
// With()/MustWith() is equivalent to Clone() and Apply()/MustApply():
//
//     reqs2, err := reqs.With(requester.Header("X-Frame","1"))
//
// The remaining methods of Requester are for creating HTTP requests, sending them, and handling
// the responses: Request(), Send(), and Receive().
//
//     req, err := reqs.Request()          // create a requests
//     resp, err := reqs.Send()            // create and send a request
//
//     var m Resource
//     resp, body, err := reqs.Receive(&m) // create and send request, read and unmarshal response
//
// Request(), Send(), and Receive() all accept a varargs of Options, which will be applied
// only to a single request, not to the Requester.
//
//     req, err := reqs.Request(
//                          requester.Put("users/bob"),
//                          requester.Body(bob),
//                        )
//
// RequestContext(), SendContext(), and ReceiveContext() variants accept a context, which is
// attached to the constructed request:
//
//     req, err        := reqs.RequestContext(ctx)
//
type Requester struct {
	////////////////////////////////////////////////////////////////
	//                                                            //
	//  Attributes affecting the construction of http.Requester.   //
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
	// of requester.  It is only used if Body is a struct value.
	// Defaults to the DefaultMarshaler, which marshals to JSON.
	//
	// If no Content-Type header has been explicitly set in Requester.Header, the
	// Marshaler will supply an appropriate one.
	Marshaler Marshaler

	//////////////////////////////////////////////////////////////
	//
	//  Attributes related to sending requester and handling
	//  responses.
	//
	////////////////////////////////////////////////////////////////

	// Doer holds the HTTP client for used to execute requester.
	// Defaults to http.DefaultClient.
	Doer Doer

	// Middleware wraps the Doer.  Middleware will be invoked in the order
	// it is in this slice.
	Middleware []Middleware

	// Unmarshaler will be used by the Receive methods to unmarshal
	// the response body.  Defaults to DefaultUnmarshaler, which unmarshals
	// multiple content types based on the Content-Type response header.
	Unmarshaler Unmarshaler
}

// New returns a new Requester, applying all options.
func New(options ...Option) (*Requester, error) {
	b := &Requester{}
	err := b.Apply(options...)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	return b, nil
}

// MustNew creates a new Requester, applying all options.  If
// an error occurs applying options, this will panic.
func MustNew(options ...Option) *Requester {
	b := &Requester{}
	b.MustApply(options...)
	return b
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

// Clone returns a deep copy of a Requester.
func (r *Requester) Clone() *Requester {
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
func (r *Requester) Request(opts ...Option) (*http.Request, error) {
	return r.RequestContext(context.Background(), opts...)
}

// RequestContext does the same as Request, but requires a context.  Use this
// to set a request timeout:
//
//     req, err := r.RequestContext(context.WithTimeout(context.Background(), 10 * time.Seconds))
//
func (r *Requester) RequestContext(ctx context.Context, opts ...Option) (*http.Request, error) {

	reqs, err := r.withOpts(opts...)
	if err != nil {
		return nil, err
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
			existingValues := req.URL.Query()
			for key, value := range reqs.QueryParams {
				for _, v := range value {
					existingValues.Add(key, v)
				}
			}
			req.URL.RawQuery = existingValues.Encode()
		} else {
			req.URL.RawQuery = reqs.QueryParams.Encode()
		}

	}

	return req.WithContext(ctx), nil
}

// getRequestBody returns the io.Reader which should be used as the body
// of new Requester.
func (r *Requester) getRequestBody() (body io.Reader, contentType string, err error) {
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

// Send executes a request with the Doer.  The response body is not closed:
// it is the caller's responsibility to close the response body.
// If the caller prefers the body as a byte slice, or prefers the body
// unmarshaled into a struct, see the Receive methods below.
//
// Additional options arguments can be passed.  They will be applied to this request only.
func (r *Requester) Send(opts ...Option) (*http.Response, error) {
	return r.SendContext(context.Background(), opts...)
}

// withOpts is like With(), but skips the clone if there are no options to apply.
func (r *Requester) withOpts(opts ...Option) (*Requester, error) {
	if len(opts) > 0 {
		return r.With(opts...)
	}
	return r, nil
}

// SendContext does the same as Request, but requires a context.
func (r *Requester) SendContext(ctx context.Context, opts ...Option) (*http.Response, error) {

	// if there are request options, apply them now, rather than passing them
	// to RequestContext().  Options may modify the Middleware or the Doer, and
	// we want to honor those options as well as the ones which affect the request.
	reqs, err := r.withOpts(opts...)
	if err != nil {
		return nil, err
	}

	req, err := reqs.RequestContext(ctx)
	if err != nil {
		return nil, err
	}
	return reqs.Do(req)
}

// Do implements Doer.  Executes the request using the configured
// Doer and Middleware.
func (r *Requester) Do(req *http.Request) (*http.Response, error) {
	doer := r.Doer
	if doer == nil {
		doer = http.DefaultClient
	}
	return Wrap(doer, r.Middleware...).Do(req)
}

// Receive creates a new HTTP request and returns the response.
// Any error creating the request, sending it, or decoding a 2XX response
// is returned.
//
// The second argument may be nil, an Option, or a value to unmarshal the
// response body into.
//
// If option arguments are passed, they are applied to this single request only.
func (r *Requester) Receive(into interface{}, opts ...Option) (resp *http.Response, body []byte, err error) {
	return r.ReceiveContext(context.Background(), into, opts...)
}

// ReceiveContext does the same as Receive, but requires a context.
//
// The second argument may be nil, an Option, or a value to unmarshal the
// response body into.
func (r *Requester) ReceiveContext(ctx context.Context, into interface{}, opts ...Option) (resp *http.Response, body []byte, err error) {

	// if into is really an option, treat it like an option
	if opt, ok := into.(Option); ok {
		opts = append(opts, nil)
		copy(opts[1:], opts)
		opts[0] = opt
		into = nil
	}

	r, err = r.withOpts(opts...)
	if err != nil {
		return nil, nil, err
	}

	resp, err = r.SendContext(ctx)

	// Due to middleware, there are cases where both a response *and* and error
	// are returned.  We need to make sure we handle the body, if present, even when
	// an error was returned.
	body, bodyReadError := readBody(resp)

	if err != nil {
		return resp, body, err
	}

	if bodyReadError != nil {
		return resp, body, bodyReadError
	}

	if into != nil {
		unmarshaler := r.Unmarshaler
		if unmarshaler == nil {
			unmarshaler = DefaultUnmarshaler
		}

		err = unmarshaler.Unmarshal(body, resp.Header.Get("Content-Type"), into)
	}
	return resp, body, err
}

func readBody(resp *http.Response) ([]byte, error) {

	if resp == nil || resp.Body == nil {
		return nil, nil
	}

	defer resp.Body.Close()

	// check if we have a content length hint.  Pre-sizing
	// the buffer saves time
	cls := resp.Header.Get("Content-Length")
	var cl int64

	if cls != "" {
		cl, _ = strconv.ParseInt(cls, 10, 0)
	}

	if cl == 0 {
		body, err := ioutil.ReadAll(resp.Body)
		return body, merry.Prepend(err, "reading response body")
	}

	buf := bytes.Buffer{}
	buf.Grow(int(cl))
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		err = merry.Wrap(err)
		return nil, merry.Prepend(err, "reading response body")
	}
	return buf.Bytes(), nil
}

// Params returns the QueryParams, initializing them if necessary.  Never returns nil.
func (r *Requester) Params() url.Values {
	if r.QueryParams == nil {
		r.QueryParams = url.Values{}
	}
	return r.QueryParams
}

// Headers returns the Header, initializing it if necessary.  Never returns nil.
func (r *Requester) Headers() http.Header {
	if r.Header == nil {
		r.Header = http.Header{}
	}
	return r.Header
}

// Trailers returns the Trailer, initializing it if necessary.  Never returns nil.
func (r *Requester) Trailers() http.Header {
	if r.Trailer == nil {
		r.Trailer = http.Header{}
	}
	return r.Trailer
}
