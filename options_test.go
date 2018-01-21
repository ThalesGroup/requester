package requests

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/url"
	"testing"
)

func TestRequests_With(t *testing.T) {
	b, err := New(Method("red"))
	require.NoError(t, err)
	b2, err := b.With(Method("green"))
	require.NoError(t, err)
	// should clone first, then apply
	require.Equal(t, "green", b2.Method)
	require.Equal(t, "red", b.Method)

	t.Run("errors", func(t *testing.T) {
		b, err := New(Method("green"))
		require.NoError(t, err)
		b2, err := b.With(Method("red"), RelativeURL("cache_object:foo/bar"))
		require.Error(t, err)
		require.Nil(t, b2)
		require.Equal(t, "green", b.Method)
	})
}

func TestRequests_Apply(t *testing.T) {
	b, err := New(Method("red"))
	require.NoError(t, err)
	err = b.Apply(Method("green"))
	require.NoError(t, err)
	// applies in place
	require.Equal(t, "green", b.Method)

	t.Run("errors", func(t *testing.T) {
		err := b.Apply(URL("cache_object:foo/bar"))
		require.Error(t, err)
		require.Nil(t, b.URL)
	})
}

func TestURL(t *testing.T) {
	cases := []string{"http://a.io/", "http://b.io", "/relPath", "relPath", ""}
	for _, base := range cases {
		t.Run("", func(t *testing.T) {
			b, errFromNew := New(URL(base))
			u, err := url.Parse(base)
			if err == nil {
				require.Equal(t, u, b.URL)
			} else {
				require.EqualError(t, errFromNew, err.Error())
			}
		})
	}

	t.Run("errors", func(t *testing.T) {
		b, err := New(URL("cache_object:foo/bar"))
		require.Error(t, err)
		require.Nil(t, b)
	})
}

func TestRelativeURL(t *testing.T) {
	cases := []struct {
		base     string
		relPath  string
		expected string
	}{
		{"http://a.io/", "foo", "http://a.io/foo"},
		{"http://a.io/", "/foo", "http://a.io/foo"},
		{"http://a.io", "foo", "http://a.io/foo"},
		{"http://a.io", "/foo", "http://a.io/foo"},
		{"http://a.io/foo/", "bar", "http://a.io/foo/bar"},
		// base should end in trailing slash if it is to be URL extended
		{"http://a.io/foo", "bar", "http://a.io/bar"},
		{"http://a.io/foo", "/bar", "http://a.io/bar"},
		// relPath extension is absolute
		{"http://a.io", "http://b.io/", "http://b.io/"},
		{"http://a.io/", "http://b.io/", "http://b.io/"},
		{"http://a.io", "http://b.io", "http://b.io"},
		{"http://a.io/", "http://b.io", "http://b.io"},
		// empty base, empty relPath
		{"", "http://b.io", "http://b.io"},
		{"http://a.io", "", "http://a.io"},
		{"", "", ""},
		{"/red", "", "/red"},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			b, err := New()
			require.NoError(t, err)
			if c.base != "" {
				err := b.Apply(URL(c.base))
				require.NoError(t, err)
			}
			err = b.Apply(RelativeURL(c.relPath))
			require.NoError(t, err)
			require.Equal(t, c.expected, b.URL.String())
		})
	}

	t.Run("errors", func(t *testing.T) {
		b, err := New(URL("http://test.com/red"))
		require.NoError(t, err)
		err = b.Apply(RelativeURL("cache_object:foo/bar"))
		require.Error(t, err)
		require.Equal(t, "http://test.com/red", b.URL.String())
	})
}

func TestMethod(t *testing.T) {
	cases := []struct {
		options        []Option
		expectedMethod string
	}{
		{[]Option{Method("red")}, "red"},
		{[]Option{Head()}, "HEAD"},
		{[]Option{Get()}, "GET"},
		{[]Option{Post()}, "POST"},
		{[]Option{Put()}, "PUT"},
		{[]Option{Patch()}, "PATCH"},
		{[]Option{Delete()}, "DELETE"},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			b, err := New(c.options...)
			require.NoError(t, err)
			require.Equal(t, c.expectedMethod, b.Method)
		})
	}
}

func TestAddHeader(t *testing.T) {
	cases := []struct {
		options        []Option
		expectedHeader http.Header
	}{
		{[]Option{AddHeader("authorization", "OAuth key=\"value\"")}, http.Header{"Authorization": {"OAuth key=\"value\""}}},
		// header keys should be canonicalized
		{[]Option{AddHeader("content-tYPE", "application/json"), AddHeader("User-AGENT", "requests")}, http.Header{"Content-Type": {"application/json"}, "User-Agent": {"requests"}}},
		// values for existing keys should be appended
		{[]Option{AddHeader("A", "B"), AddHeader("a", "c")}, http.Header{"A": {"B", "c"}}},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			b, err := New(c.options...)
			require.NoError(t, err)
			require.Equal(t, c.expectedHeader, b.Header)
		})
	}
}

func TestHeader(t *testing.T) {
	cases := []struct {
		options        []Option
		expectedHeader http.Header
	}{
		// should replace existing values associated with key
		{[]Option{AddHeader("A", "B"), Header("a", "c")}, http.Header{"A": []string{"c"}}},
		{[]Option{Header("content-type", "A"), Header("Content-Type", "B")}, http.Header{"Content-Type": []string{"B"}}},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			b, err := New(c.options...)
			require.NoError(t, err)
			// type conversion from Header to alias'd map for deep equality comparison
			require.Equal(t, c.expectedHeader, b.Header)
		})
	}
}

func TestBasicAuth(t *testing.T) {
	cases := []struct {
		options      []Option
		expectedAuth []string
	}{
		// basic auth: username & password
		{[]Option{BasicAuth("Aladdin", "open sesame")}, []string{"Aladdin", "open sesame"}},
		// empty username
		{[]Option{BasicAuth("", "secret")}, []string{"", "secret"}},
		// empty password
		{[]Option{BasicAuth("admin", "")}, []string{"admin", ""}},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			b, err := New(c.options...)
			require.NoError(t, err)
			req, err := b.RequestContext(context.Background())
			require.NoError(t, err)
			username, password, ok := req.BasicAuth()
			require.True(t, ok, "basic auth missing when expected")
			auth := []string{username, password}
			require.Equal(t, c.expectedAuth, auth)
		})
	}
}

func TestBearerAuth(t *testing.T) {
	cases := []string{
		"red",
		"",
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			b, err := New(BearerAuth(c))
			require.NoError(t, err)
			if c == "" {
				require.Empty(t, b.Header.Get("Authorization"))
			} else {
				require.Equal(t, "Bearer "+c, b.Header.Get("Authorization"))
			}
		})
	}

	t.Run("clearing", func(t *testing.T) {
		b, err := New(BearerAuth("green"))
		require.NoError(t, err)
		err = b.Apply(BearerAuth(""))
		require.NoError(t, err)
		_, ok := b.Header["Authorization"]
		require.False(t, ok, "should have removed Authorization header, instead was %s", b.Header.Get("Authorization"))
	})
}

func TestQueryParams(t *testing.T) {
	cases := []struct {
		options        []Option
		expectedParams url.Values
	}{
		{nil, nil},
		{[]Option{QueryParams(nil)}, url.Values{}},
		{[]Option{QueryParams(paramsA)}, url.Values{"limit": []string{"30"}}},
		{[]Option{QueryParams(paramsA), QueryParams(paramsA)}, url.Values{"limit": []string{"30", "30"}}},
		{[]Option{QueryParams(paramsA), QueryParams(paramsB)}, url.Values{"limit": []string{"30"}, "kind_name": []string{"recent"}, "count": []string{"25"}}},
		{[]Option{QueryParams(paramsA, paramsB)}, url.Values{"limit": []string{"30"}, "kind_name": []string{"recent"}, "count": []string{"25"}}},
		{[]Option{QueryParams(url.Values{"red": []string{"green"}})}, url.Values{"red": []string{"green"}}},
		{[]Option{QueryParams(map[string][]string{"red": []string{"green"}})}, url.Values{"red": []string{"green"}}},
		{[]Option{QueryParams(map[string]string{"red": "green"})}, url.Values{"red": []string{"green"}}},
	}

	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			b, err := New(c.options...)
			require.NoError(t, err)
			require.Equal(t, c.expectedParams, b.QueryParams)
		})
	}
}

func TestQueryParam(t *testing.T) {
	reqs, err := New(QueryParam("color", "red"))
	require.NoError(t, err)

	expected := url.Values{}
	expected.Add("color", "red")
	require.Equal(t, expected, reqs.QueryParams)

	err = reqs.Apply(QueryParam("color", "blue"))
	require.NoError(t, err)

	expected.Add("color", "blue")
	require.Equal(t, expected, reqs.QueryParams)
}

func TestBody(t *testing.T) {
	b, err := New(Body("hey"))
	require.NoError(t, err)
	require.Equal(t, "hey", b.Body)
}

type testMarshaler struct{}

func (*testMarshaler) Unmarshal(data []byte, contentType string, v interface{}) error {
	panic("implement me")
}

func (*testMarshaler) Marshal(v interface{}) (data []byte, contentType string, err error) {
	panic("implement me")
}

func TestMarshaler(t *testing.T) {
	m := &testMarshaler{}
	b, err := New(Marshaler(m))
	require.NoError(t, err)
	require.Equal(t, m, b.Marshaler)
}

func TestUnmarshaler(t *testing.T) {
	m := &testMarshaler{}
	b, err := New(Unmarshaler(m))
	require.NoError(t, err)
	require.Equal(t, m, b.Unmarshaler)
}

func TestJSON(t *testing.T) {
	b, err := New(JSON(false))
	require.NoError(t, err)
	if assert.IsType(t, &JSONMarshaler{}, b.Marshaler) {
		assert.False(t, b.Marshaler.(*JSONMarshaler).Indent)
	}

	err = b.Apply(JSON(true))
	require.NoError(t, err)
	if assert.IsType(t, &JSONMarshaler{}, b.Marshaler) {
		assert.True(t, b.Marshaler.(*JSONMarshaler).Indent)
	}
}

func TestXML(t *testing.T) {
	b, err := New(XML(false))
	require.NoError(t, err)
	if assert.IsType(t, &XMLMarshaler{}, b.Marshaler) {
		assert.False(t, b.Marshaler.(*XMLMarshaler).Indent)
	}

	err = b.Apply(XML(true))
	require.NoError(t, err)
	if assert.IsType(t, &XMLMarshaler{}, b.Marshaler) {
		assert.True(t, b.Marshaler.(*XMLMarshaler).Indent)
	}
}

func TestForm(t *testing.T) {
	b, err := New(Form())
	require.NoError(t, err)
	assert.IsType(t, &FormMarshaler{}, b.Marshaler)
}
