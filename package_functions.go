package requester

import (
	"context"
	"net/http"
)

// DefaultRequester is the singleton used by the package-level Request/Send/Receive functions.
var DefaultRequester = Requester{}

// Request does the same as Requester.Request(), using the DefaultRequester.
func Request(opts ...Option) (*http.Request, error) {
	return DefaultRequester.Request(opts...)
}

// RequestContext does the same as Requester.RequestContext(), using the DefaultRequester.
func RequestContext(ctx context.Context, opts ...Option) (*http.Request, error) {
	return DefaultRequester.RequestContext(ctx, opts...)
}

// Send does the same as Requester.Send(), using the DefaultRequester.
func Send(opts ...Option) (*http.Response, error) {
	return DefaultRequester.Send(opts...)
}

// SendContext does the same as Requester.SendContext(), using the DefaultRequester.
func SendContext(ctx context.Context, opts ...Option) (*http.Response, error) {
	return DefaultRequester.SendContext(ctx, opts...)
}

// ReceiveContext does the same as Requester.ReceiveContext(), using the DefaultRequester.
func ReceiveContext(ctx context.Context, successV interface{}, opts ...Option) (*http.Response, string, error) {
	return DefaultRequester.ReceiveContext(ctx, successV, opts...)
}

// Receive does the same as Requester.Receive(), using the DefaultRequester.
func Receive(successV interface{}, opts ...Option) (*http.Response, string, error) {
	return DefaultRequester.Receive(successV, opts...)
}
