package requester

import "net/http"

// These are tools for writing tests.

// MockDoer creates a mock Doer which returns a mocked response.
// By default, the mocked response will contain the status code,
// and typical default values for some standard response fields, like
// the ProtoXXX fields.
//
// Options can be passed in, which are used to construct a template
// http.Request.  The fields of the template request are copied into
// the mocked responses (http.Request and http.Response share most fields,
// so we're leverage the rich set of requester.Options to build the response).
func MockDoer(statusCode int, options ...Option) DoerFunc {
	return func(req *http.Request) (*http.Response, error) {
		resp := MockResponse(statusCode, options...)
		resp.Request = req
		return resp, nil
	}
}

// ChannelDoer returns a DoerFunc and a channel.  The DoerFunc will return the responses
// send on the channel.
func ChannelDoer() (chan<- *http.Response, DoerFunc) {
	input := make(chan *http.Response, 1)

	return input, func(req *http.Request) (*http.Response, error) {
		resp := <-input
		resp.Request = req
		return resp, nil
	}
}

// MockResponse creates an *http.Response from the Options.  Requests and Responses share most of the
// same fields, so we use the options to build a Request, then copy the values as appropriate
// into a Response.  Useful for created mocked responses for tests.
func MockResponse(statusCode int, options ...Option) *http.Response {
	r, err := Request(options...)
	if err != nil {
		panic(err)
	}

	resp := &http.Response{
		// TODO: Status
		StatusCode:       statusCode,
		Proto:            r.Proto,
		ProtoMajor:       r.ProtoMajor,
		ProtoMinor:       r.ProtoMinor,
		Header:           r.Header,
		Body:             r.Body,
		ContentLength:    r.ContentLength,
		TransferEncoding: r.TransferEncoding,
		// TODO: Close,
		Trailer: r.Trailer,
	}
	return resp
}
