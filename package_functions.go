package requests

import (
	"context"
	"net/http"
)

// DefaultRequests is the singleton used by the package-level Request/Do/Receive functions.
var DefaultRequests = Requests{}

// Request does the same as Requests.Request(), using the DefaultRequests.
func Request(opts ...Option) (*http.Request, error) {
	return DefaultRequests.Request(opts...)
}

// RequestContext does the same as Requests.RequestContext(), using the DefaultRequests.
func RequestContext(ctx context.Context, opts ...Option) (*http.Request, error) {
	return DefaultRequests.RequestContext(ctx, opts...)
}

// Do does the same as Requests.Do(), using the DefaultRequests.
func Do(opts ...Option) (*http.Response, error) {
	return DefaultRequests.Do(opts...)
}

// DoContext does the same as Requests.DoContext(), using the DefaultRequests.
func DoContext(ctx context.Context, opts ...Option) (*http.Response, error) {
	return DefaultRequests.DoContext(ctx, opts...)
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
