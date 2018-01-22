package requests_test

import (
	"bytes"
	. "github.com/gemalto/requests"
	"github.com/gemalto/requests/clientserver"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestDump(t *testing.T) {
	cs := clientserver.New(nil)
	defer cs.Close()
	cs.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"color":"red"}`))
	})

	b := &bytes.Buffer{}

	cs.Receive(nil, Dump(b))

	t.Log(b)

	assert.Contains(t, b.String(), "GET / HTTP/1.1")
	assert.Contains(t, b.String(), "HTTP/1.1 200 OK")
	assert.Contains(t, b.String(), `{"color":"red"}`)
}

//func NewClientServer(s *httptest.Server, options ...Option) *ClientServer {
//	if s == nil {
//		s = httptest.NewServer(nil)
//	}
//	t := &ClientServer{
//		Server: s,
//		Requests: &Requests{
//			Doer: s.Client(),
//		},
//	}
//	s.Config.Handler = t
//
//
//	err := t.Apply(URL(s.URL), optionsSlice(options), Use(t.captureClientReqResp))
//	if err != nil {
//		panic(err)
//	}
//
//	return t
//}
//
//type optionsSlice []Option
//
//func (o optionsSlice) Apply(r *Requests) error {
//	return r.Apply(o...)
//}
//
//type ClientServer struct {
//	*httptest.Server
//	*Requests
//	Handler http.Handler
//
//	LastSrvReq     *http.Request
//	LastClientReq  *http.Request
//	LastClientResp *http.Response
//}
//
//func (t *ClientServer) Close() {
//	t.Server.Close()
//}
//
//func (t *ClientServer) Clear() {
//	t.LastClientReq = nil
//	t.LastClientResp = nil
//	t.LastSrvReq = nil
//}
//
//func (t *ClientServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
//	t.LastSrvReq = req
//	if t.Handler != nil {
//		t.Handler.ServeHTTP(w, req)
//	}
//}
//
//func (t *ClientServer) captureClientReqResp(next Doer) Doer {
//	return DoerFunc(func(req *http.Request) (*http.Response, error) {
//		t.LastClientReq = req
//		resp, err := next.Do(req)
//		t.LastClientResp = resp
//		return resp, err
//	})
//}
//
//func (t *ClientServer) Mux() *http.ServeMux {
//	if m, ok := t.Config.Handler.(*http.ServeMux); ok {
//		return m
//	}
//	m := http.NewServeMux()
//	t.Config.Handler = m
//	return m
//}
//
//func (t *ClientServer) HandlerFunc(hf http.HandlerFunc) {
//	t.Handler = hf
//}
