package requester

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

// Inspect installs and returns an Inspector.  The Inspector captures the last
// request, request body, response and response body.  Useful in tests for inspecting
// traffic.
func Inspect(r *Requester) *Inspector {
	i := Inspector{}
	r.MustApply(&i)
	return &i
}

// Inspector is a Requester Option which captures requests and responses.
// It's useful for inspecting the contents of exchanges in tests.
//
// It not an efficient way to capture bodies, and keeps requests
// and responses around longer than their intended lifespan, so it
// should not be used in production code or benchmarks.
type Inspector struct {

	// The last request sent by the client.
	Request *http.Request

	// The last response received by the client.
	Response *http.Response

	// The last client request body
	RequestBody *bytes.Buffer

	// The last client response body
	ResponseBody *bytes.Buffer
}

// Clear clears the inspector's fields.
func (i *Inspector) Clear() {
	if i == nil {
		return
	}
	i.RequestBody = nil
	i.ResponseBody = nil
	i.Request = nil
	i.Response = nil
}

// Apply implements Option
func (i *Inspector) Apply(r *Requester) error {
	return r.Apply(Middleware(i.Wrap))
}

// Wrap implements Middleware
func (i *Inspector) Wrap(next Doer) Doer {
	return DoerFunc(func(req *http.Request) (*http.Response, error) {
		i.Request = req
		// capture the body
		if req.Body != nil {
			reqBody, _ := ioutil.ReadAll(req.Body)
			req.Body.Close()
			req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))
			i.RequestBody = bytes.NewBuffer(reqBody)
		}
		resp, err := next.Do(req)
		i.Response = resp
		if resp != nil && resp.Body != nil {
			respBody, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			resp.Body = ioutil.NopCloser(bytes.NewReader(respBody))
			i.ResponseBody = bytes.NewBuffer(respBody)
		}
		return resp, err
	})
}
