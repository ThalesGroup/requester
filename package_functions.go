package requester

import (
	"context"
	"net/http"
)

// DefaultRequester is the singleton used by the package-level Request/Send/Receive functions.
// nolint:gochecknoglobals
var DefaultRequester = Requester{}

// Request uses the DefaultRequester to create a request.
//
// See Requester.Request() for more details.
func Request(opts ...Option) (*http.Request, error) {
	return DefaultRequester.Request(opts...)
}

// RequestContext does the same as Request(), but attaches a Context to the request.
func RequestContext(ctx context.Context, opts ...Option) (*http.Request, error) {
	return DefaultRequester.RequestContext(ctx, opts...)
}

// Send uses the DefaultRequester to create a request and execute it.
// The body will not be read or closed.
//
// See Requester.Send() for more details.
func Send(opts ...Option) (*http.Response, error) {
	return DefaultRequester.Send(opts...)
}

// SendContext does the same as Send(), but attaches a Context to the request.
func SendContext(ctx context.Context, opts ...Option) (*http.Response, error) {
	return DefaultRequester.SendContext(ctx, opts...)
}

// ReceiveContext does the same as Receive(), but attaches a Context to
// the request.
//
// The second argument may be nil, an Option, or a value to unmarshal the
// response body into.
func ReceiveContext(ctx context.Context, into interface{}, opts ...Option) (*http.Response, []byte, error) {
	return DefaultRequester.ReceiveContext(ctx, into, opts...)
}

// Receive uses the DefaultRequester to create a request, execute it, and read the response.
// The response body will be fully read and closed.
//
// See Requester.Receive() for more details.
//
// The first argument may be nil, an Option, or a value to unmarshal the
// response body into.
func Receive(into interface{}, opts ...Option) (*http.Response, []byte, error) {
	return DefaultRequester.Receive(into, opts...)
}
