package requester

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

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
	i.RequestBody = nil
	i.ResponseBody = nil
	i.Request = nil
	i.Response = nil
}

// Apply implements Option
func (i *Inspector) Apply(r *Requester) error {
	return r.Apply(Middleware(i.MiddlewareFunc))
}

// MiddlewareFunc implements Middleware
func (i *Inspector) MiddlewareFunc(next Doer) Doer {
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
