package requests

import (
	"context"
	"net/http"
)

// DefaultRequests is the singleton used by the package-level Request/Send/Receive functions.
var DefaultRequests = Requests{}

// Request does the same as Requests.Request(), using the DefaultRequests.
func Request(opts ...Option) (*http.Request, error) {
	return DefaultRequests.Request(opts...)
}

// RequestContext does the same as Requests.RequestContext(), using the DefaultRequests.
func RequestContext(ctx context.Context, opts ...Option) (*http.Request, error) {
	return DefaultRequests.RequestContext(ctx, opts...)
}

// Send does the same as Requests.Send(), using the DefaultRequests.
func Send(opts ...Option) (*http.Response, error) {
	return DefaultRequests.Send(opts...)
}

// SendContext does the same as Requests.SendContext(), using the DefaultRequests.
func SendContext(ctx context.Context, opts ...Option) (*http.Response, error) {
	return DefaultRequests.SendContext(ctx, opts...)
}

// ReceiveContext does the same as Requests.ReceiveContext(), using the DefaultRequests.
func ReceiveContext(ctx context.Context, successV interface{}, opts ...Option) (*http.Response, string, error) {
	return DefaultRequests.ReceiveContext(ctx, successV, opts...)
}

// Receive does the same as Requests.Receive(), using the DefaultRequests.
func Receive(successV interface{}, opts ...Option) (*http.Response, string, error) {
	return DefaultRequests.Receive(successV, opts...)
}

// ReceiveFull does the same as Requests.ReceiveFull(), using the DefaultRequests.
func ReceiveFull(successV, failureV interface{}, opts ...Option) (*http.Response, string, error) {
	return DefaultRequests.ReceiveFull(successV, failureV, opts...)
}

// ReceiveFullContext does the same as Requests.ReceiveFullContext(), using the DefaultRequests.
func ReceiveFullContext(ctx context.Context, successV, failureV interface{}, opts ...Option) (*http.Response, string, error) {
	return DefaultRequests.ReceiveFullContext(ctx, successV, failureV, opts...)
}
