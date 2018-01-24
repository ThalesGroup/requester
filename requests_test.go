package requester_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	. "github.com/gemalto/requester"
	"github.com/gemalto/requester/clientserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// Url-tagged query struct
var paramsA = struct {
	Limit int `url:"limit"`
}{
	30,
}
var paramsB = FakeParams{KindName: "recent", Count: 25}

// Json-tagged model struct
type FakeModel struct {
	Text          string  `json:"text,omitempty"`
	FavoriteCount int64   `json:"favorite_count,omitempty"`
	Temperature   float64 `json:"temperature,omitempty"`
}

var modelA = FakeModel{Text: "note", FavoriteCount: 12}

func TestNew(t *testing.T) {
	reqs, err := New()
	require.NoError(t, err)
	require.NotNil(t, reqs)
}

func TestRequester_Clone(t *testing.T) {
	cases := [][]Option{
		{Get(), URL("http: //example.com")},
		{URL("http://example.com")},
		{QueryParams(url.Values{})},
		{QueryParams(paramsA)},
		{QueryParams(paramsA, paramsB)},
		{Body(&FakeModel{Text: "a"})},
		{Body(FakeModel{Text: "a"})},
		{Header("Content-Type", "application/json")},
		{AddHeader("A", "B"), AddHeader("a", "c")},
	}

	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			reqs, err := New(c...)
			require.NoError(t, err)

			child := reqs.Clone()
			require.Equal(t, reqs.Doer, child.Doer)
			require.Equal(t, reqs.Method, child.Method)
			require.Equal(t, reqs.URL, child.URL)
			// Header should be a copy of parent Requester header. For example, calling
			// baseSling.AddHeader("k","v") should not mutate previously created child Slings
			assert.EqualValues(t, reqs.Header, child.Header)
			if reqs.Header != nil {
				// struct literal cases don't init Header in usual way, skip header check
				assert.EqualValues(t, reqs.Header, child.Header)
				reqs.Header.Add("K", "V")
				assert.Empty(t, child.Header.Get("K"), "child.header was a reference to original map, should be copy")
			} else {
				assert.Nil(t, child.Header)
			}
			// queryStruct slice should be a new slice with a copy of the contents
			assert.EqualValues(t, reqs.QueryParams, child.QueryParams)
			if len(reqs.QueryParams) > 0 {
				// mutating one slice should not mutate the other
				child.QueryParams.Set("color", "red")
				assert.Empty(t, reqs.QueryParams.Get("color"), "child.QueryParams should be a copy")
			}
			// bodyJSON should be copied
			assert.Equal(t, reqs.Body, child.Body)
		})
	}
}

func TestRequester_Request_URLAndMethod(t *testing.T) {
	cases := []struct {
		options        []Option
		expectedMethod string
		expectedURL    string
	}{
		{[]Option{URL("http://a.io")}, "GET", "http://a.io"},
		{[]Option{RelativeURL("http://a.io")}, "GET", "http://a.io"},
		{[]Option{Get("http://a.io")}, "GET", "http://a.io"},
		{[]Option{Put("http://a.io")}, "PUT", "http://a.io"},
		{[]Option{URL("http://a.io/"), RelativeURL("foo")}, "GET", "http://a.io/foo"},
		{[]Option{URL("http://a.io/"), Post("foo")}, "POST", "http://a.io/foo"},
		// if relative relPath is an absolute url, base is ignored
		{[]Option{URL("http://a.io"), RelativeURL("http://b.io")}, "GET", "http://b.io"},
		{[]Option{RelativeURL("http://a.io"), RelativeURL("http://b.io")}, "GET", "http://b.io"},
		// last method setter takes priority
		{[]Option{Get("http://b.io"), Post("http://a.io")}, "POST", "http://a.io"},
		{[]Option{Post("http://a.io/"), Put("foo/"), Delete("bar")}, "DELETE", "http://a.io/foo/bar"},
		// last Base setter takes priority
		{[]Option{URL("http://a.io"), URL("http://b.io")}, "GET", "http://b.io"},
		// URL setters are additive
		{[]Option{URL("http://a.io/"), RelativeURL("foo/"), RelativeURL("bar")}, "GET", "http://a.io/foo/bar"},
		{[]Option{RelativeURL("http://a.io/"), RelativeURL("foo/"), RelativeURL("bar")}, "GET", "http://a.io/foo/bar"},
		// removes extra '/' between base and ref url
		{[]Option{URL("http://a.io/"), Get("/foo")}, "GET", "http://a.io/foo"},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			reqs, err := New(c.options...)
			require.NoError(t, err)
			req, err := reqs.RequestContext(context.Background())
			require.NoError(t, err)
			assert.Equal(t, c.expectedURL, req.URL.String())
			assert.Equal(t, c.expectedMethod, req.Method)
		})
	}

	t.Run("invalidmethod", func(t *testing.T) {
		b, err := New(Method("@"))
		require.NoError(t, err)
		req, err := b.RequestContext(context.Background())
		require.Error(t, err)
		require.Nil(t, req)
	})

}

func TestRequester_Request_QueryParams(t *testing.T) {
	cases := []struct {
		options     []Option
		expectedURL string
	}{
		{[]Option{URL("http://a.io"), QueryParams(paramsA)}, "http://a.io?limit=30"},
		{[]Option{URL("http://a.io/?color=red"), QueryParams(paramsA)}, "http://a.io/?color=red&limit=30"},
		{[]Option{URL("http://a.io"), QueryParams(paramsA), QueryParams(paramsB)}, "http://a.io?count=25&kind_name=recent&limit=30"},
		{[]Option{URL("http://a.io/"), RelativeURL("foo?relPath=yes"), QueryParams(paramsA)}, "http://a.io/foo?relPath=yes&limit=30"},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			reqs, err := New(c.options...)
			require.NoError(t, err)
			req, _ := reqs.RequestContext(context.Background())
			require.Equal(t, c.expectedURL, req.URL.String())
		})
	}
}

func TestRequester_Request_Body(t *testing.T) {
	cases := []struct {
		options             []Option
		expectedBody        string // expected Body io.Reader as a string
		expectedContentType string
	}{
		// Body (json)
		{[]Option{Body(modelA)}, `{"text":"note","favorite_count":12}`, ContentTypeJSON},
		{[]Option{Body(&modelA)}, `{"text":"note","favorite_count":12}`, ContentTypeJSON},
		{[]Option{Body(&FakeModel{})}, `{}`, ContentTypeJSON},
		{[]Option{Body(FakeModel{})}, `{}`, ContentTypeJSON},
		// BodyForm
		//{[]Option{Body(paramsA)}, "limit=30", formContentType},
		//{[]Option{Body(paramsB)}, "count=25&kind_name=recent", formContentType},
		//{[]Option{Body(&paramsB)}, "count=25&kind_name=recent", formContentType},
		// Raw bodies, skips marshaler
		{[]Option{Body(strings.NewReader("this-is-a-test"))}, "this-is-a-test", ""},
		{[]Option{Body("this-is-a-test")}, "this-is-a-test", ""},
		{[]Option{Body([]byte("this-is-a-test"))}, "this-is-a-test", ""},
		// no body
		{nil, "", ""},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			reqs, err := New(c.options...)
			require.NoError(t, err)
			req, err := reqs.RequestContext(context.Background())
			require.NoError(t, err)
			if reqs.Body != nil {
				buf := new(bytes.Buffer)
				buf.ReadFrom(req.Body)
				// req.Body should have contained the expectedBody string
				assert.Equal(t, c.expectedBody, buf.String())
				// Header Content-Type should be expectedContentType ("" means no contentType expected)

			} else {
				assert.Nil(t, req.Body)
			}
			assert.Equal(t, c.expectedContentType, req.Header.Get(HeaderContentType))
		})
	}
}

func TestRequester_Request_Marshaler(t *testing.T) {
	var capturedV interface{}
	reqs := Requester{
		Body: []string{"blue"},
		Marshaler: MarshalFunc(func(v interface{}) ([]byte, string, error) {
			capturedV = v
			return []byte("red"), "orange", nil
		}),
	}

	req, err := reqs.RequestContext(context.Background())
	require.NoError(t, err)

	require.Equal(t, []string{"blue"}, capturedV)
	by, err := ioutil.ReadAll(req.Body)
	require.NoError(t, err)
	require.Equal(t, "red", string(by))
	require.Equal(t, "orange", req.Header.Get("Content-Type"))

	t.Run("errors", func(t *testing.T) {
		reqs.Marshaler = MarshalFunc(func(v interface{}) ([]byte, string, error) {
			return nil, "", errors.New("boom")
		})
		_, err := reqs.RequestContext(context.Background())
		require.Error(t, err, "boom")
	})
}

func TestRequester_Request_ContentLength(t *testing.T) {
	reqs, err := New(Body("1234"))
	require.NoError(t, err)
	req, err := reqs.RequestContext(context.Background())
	require.NoError(t, err)
	// content length should be set automatically
	require.EqualValues(t, 4, req.ContentLength)

	// I should be able to override it
	reqs.ContentLength = 10
	req, err = reqs.RequestContext(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, 10, req.ContentLength)
}

func TestRequester_Request_GetBody(t *testing.T) {
	reqs, err := New(Body("1234"))
	require.NoError(t, err)
	req, err := reqs.RequestContext(context.Background())
	require.NoError(t, err)
	// GetBody should be populated automatically
	rdr, err := req.GetBody()
	require.NoError(t, err)
	bts, err := ioutil.ReadAll(rdr)
	require.NoError(t, err)
	require.Equal(t, "1234", string(bts))

	// I should be able to override it
	reqs.GetBody = func() (io.ReadCloser, error) {
		return ioutil.NopCloser(strings.NewReader("5678")), nil
	}
	req, err = reqs.RequestContext(context.Background())
	require.NoError(t, err)
	rdr, err = req.GetBody()
	require.NoError(t, err)
	bts, err = ioutil.ReadAll(rdr)
	require.NoError(t, err)
	require.Equal(t, "5678", string(bts))
}

func TestRequester_Request_Host(t *testing.T) {
	reqs, err := New(URL("http://test.com/red"))
	require.NoError(t, err)
	req, err := reqs.RequestContext(context.Background())
	require.NoError(t, err)
	// Host should be set automatically
	require.Equal(t, "test.com", req.Host)

	// but I can override it
	reqs.Host = "test2.com"
	req, err = reqs.RequestContext(context.Background())
	require.NoError(t, err)
	require.Equal(t, "test2.com", req.Host)
}

func TestRequester_Request_TransferEncoding(t *testing.T) {
	reqs := Requester{}
	req, err := reqs.RequestContext(context.Background())
	require.NoError(t, err)
	// should be empty by default
	require.Nil(t, req.TransferEncoding)

	// but I can set it
	reqs.TransferEncoding = []string{"red"}
	req, err = reqs.RequestContext(context.Background())
	require.NoError(t, err)
	require.Equal(t, reqs.TransferEncoding, req.TransferEncoding)
}

func TestRequester_Request_Close(t *testing.T) {
	reqs := Requester{}
	req, err := reqs.RequestContext(context.Background())
	require.NoError(t, err)
	// should be false by default
	require.False(t, req.Close)

	// but I can set it
	reqs.Close = true
	req, err = reqs.RequestContext(context.Background())
	require.NoError(t, err)
	require.True(t, req.Close)
}

func TestRequester_Request_Trailer(t *testing.T) {
	reqs := Requester{}
	req, err := reqs.RequestContext(context.Background())
	require.NoError(t, err)
	// should be empty by default
	require.Nil(t, req.Trailer)

	// but I can set it
	reqs.Trailer = http.Header{"color": []string{"red"}}
	req, err = reqs.RequestContext(context.Background())
	require.NoError(t, err)
	require.Equal(t, reqs.Trailer, req.Trailer)
}

func TestRequester_Request_Header(t *testing.T) {
	reqs := Requester{}
	req, err := reqs.RequestContext(context.Background())
	require.NoError(t, err)
	// should be empty by default
	require.Empty(t, req.Header)

	// but I can set it
	reqs.Header = http.Header{"color": []string{"red"}}
	req, err = reqs.RequestContext(context.Background())
	require.NoError(t, err)
	require.Equal(t, reqs.Header, req.Header)
}

func TestRequester_Request_Context(t *testing.T) {
	reqs := Requester{}
	req, err := reqs.RequestContext(context.WithValue(context.Background(), colorContextKey, "red"))
	require.NoError(t, err)
	require.Equal(t, "red", req.Context().Value(colorContextKey))
}

func TestRequester_Request(t *testing.T) {
	reqs := Requester{}
	req, err := reqs.Request()
	require.NoError(t, err)
	require.NotNil(t, req)
}

func TestRequester_Request_options(t *testing.T) {
	reqs := Requester{}
	req, err := reqs.Request(Get("http://test.com/blue"))
	require.NoError(t, err)
	assert.Equal(t, "http://test.com/blue", req.URL.String())
}

func TestRequester_SendContext(t *testing.T) {
	cs := clientserver.New(nil)
	defer cs.Close()

	// SendContext() just creates a request and sends it to the Doer.  That's all we're confirming here
	cs.Mux().HandleFunc("/server", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})

	resp, err := cs.SendContext(
		context.WithValue(context.Background(), colorContextKey, "purple"),
		Post("/server"),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	// confirm the request went through
	require.NotNil(t, resp)
	assert.Equal(t, 204, resp.StatusCode)
	assert.Equal(t, "purple", cs.LastClientReq.Context().Value(colorContextKey), "context should be passed through")
	assert.Equal(t, "", cs.Method, "option arguments should have only affected that request, should not have leaked back into the Requester objects")

	t.Run("Send", func(t *testing.T) {
		// same as SendContext, just without the context.
		resp, err := cs.Send(Get("/server"))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 204, resp.StatusCode)
	})
}

func TestRequester_ReceiveFullContext(t *testing.T) {

	cs := clientserver.New(nil, Get("/model.json"))
	defer cs.Close()

	cs.Mux().HandleFunc("/model.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(206)
		w.Write([]byte(`{"color":"green","count":25}`))
	})

	cs.Mux().HandleFunc("/err", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write([]byte(`{"color":"red","count":30}`))
	})

	t.Run("success", func(t *testing.T) {
		cases := []struct {
			succV, failV interface{}
		}{
			{&testModel{}, &testModel{}},
			{&testModel{}, nil},
			{nil, &testModel{}},
			{nil, nil},
		}
		for _, c := range cases {
			t.Run(fmt.Sprintf("succV=%v,failV=%v", c.succV, c.failV), func(t *testing.T) {
				cs.Clear()

				resp, body, err := cs.ReceiveFullContext(
					context.WithValue(context.Background(), colorContextKey, "purple"),
					c.succV,
					c.failV,
				)
				require.NoError(t, err)
				assert.Equal(t, 206, resp.StatusCode)
				assert.Equal(t, `{"color":"green","count":25}`, body)
				assert.Equal(t, "purple", cs.LastClientReq.Context().Value(colorContextKey), "context should be passed through")
				if c.succV != nil {
					assert.Equal(t, &testModel{"green", 25}, c.succV)
				}
				if c.failV != nil {
					assert.Equal(t, &testModel{}, c.failV)
				}
			})
		}
	})

	t.Run("failure", func(t *testing.T) {
		cases := []struct {
			succV, failV interface{}
		}{
			{&testModel{}, &testModel{}},
			{&testModel{}, nil},
			{nil, &testModel{}},
			{nil, nil},
		}
		for _, c := range cases {
			t.Run(fmt.Sprintf("succV=%v,failV=%v", c.succV, c.failV), func(t *testing.T) {
				urlBefore := cs.Requester.URL.String()
				resp, body, err := cs.ReceiveFullContext(
					context.Background(),
					c.succV,
					c.failV,
					Get("/err"),
				)
				require.NoError(t, err)
				assert.Equal(t, 500, resp.StatusCode)
				assert.Equal(t, `{"color":"red","count":30}`, body)
				if c.succV != nil {
					assert.Equal(t, &testModel{}, c.succV)
				}
				if c.failV != nil {
					assert.Equal(t, &testModel{"red", 30}, c.failV)
				}
				assert.Equal(t, urlBefore, cs.Requester.URL.String(), "the Get option should only affect the single request, it should not leak back into the Requester object")
			})
		}
	})

	// convenience functions which just wrap ReceiveFullContext
	t.Run("ReceiveFull", func(t *testing.T) {
		var mSucc, mFail testModel
		resp, body, err := cs.ReceiveFull(&mSucc, mFail)
		require.NoError(t, err)
		assert.Equal(t, 206, resp.StatusCode)
		assert.Equal(t, `{"color":"green","count":25}`, body)
		assert.Equal(t, "green", mSucc.Color)

		resp, body, err = cs.ReceiveFull(&mSucc, &mFail, Get("/err"))
		require.NoError(t, err)
		assert.Equal(t, 500, resp.StatusCode)
		assert.Equal(t, `{"color":"red","count":30}`, body)
		assert.Equal(t, "red", mFail.Color)
	})

	t.Run("ReceiveContext", func(t *testing.T) {
		cs.Clear()
		var m testModel
		resp, body, err := cs.ReceiveContext(context.WithValue(context.Background(), colorContextKey, "purple"), &m)
		require.NoError(t, err)
		assert.Equal(t, 206, resp.StatusCode)
		assert.Equal(t, `{"color":"green","count":25}`, body)
		assert.Equal(t, "green", m.Color)
		assert.Equal(t, "purple", cs.LastClientReq.Context().Value(colorContextKey), "context should be passed through")
	})

	t.Run("Receive", func(t *testing.T) {
		var m testModel
		resp, body, err := cs.Receive(&m)
		require.NoError(t, err)
		assert.Equal(t, 206, resp.StatusCode)
		assert.Equal(t, `{"color":"green","count":25}`, body)
		assert.Equal(t, "green", m.Color)
	})

	t.Run("acceptoptionsforintoargs", func(t *testing.T) {

		var method string
		cs.Mux().HandleFunc("/blue", func(writer http.ResponseWriter, request *http.Request) {
			method = request.Method
			writer.WriteHeader(208)
		})

		// Receive will Options to be passed as the "into" arguments
		resp, _, _ := cs.Receive(Get("/blue"))
		assert.Equal(t, 208, resp.StatusCode)

		// Options should be applied in the order of the arguments
		resp, _, _ = cs.Receive(Get("/red"), Get("/blue"))
		assert.Equal(t, 208, resp.StatusCode)

		// variants

		ctx := context.Background()
		resp, _, _ = cs.ReceiveContext(ctx, Get("/blue"))
		assert.Equal(t, 208, resp.StatusCode)

		resp, _, _ = cs.ReceiveFull(Get("/blue"), nil)
		assert.Equal(t, 208, resp.StatusCode)

		resp, _, _ = cs.ReceiveFull(nil, Get("/blue"))
		assert.Equal(t, 208, resp.StatusCode)

		resp, _, _ = cs.ReceiveFull(Get("/blue"), Post())
		assert.Equal(t, 208, resp.StatusCode)
		assert.Equal(t, "POST", method)

		resp, _, _ = cs.ReceiveFullContext(ctx, Get("/blue"), nil)
		assert.Equal(t, 208, resp.StatusCode)
		assert.Equal(t, "GET", method)

		resp, _, _ = cs.ReceiveFullContext(ctx, nil, Get("/blue"))
		assert.Equal(t, 208, resp.StatusCode)

		resp, _, _ = cs.ReceiveFullContext(ctx, Get("/blue"), Post())
		assert.Equal(t, 208, resp.StatusCode)
		assert.Equal(t, "POST", method)
	})
}
