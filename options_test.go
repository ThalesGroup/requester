package requester

import (
	"context"
	"fmt"
	"github.com/gemalto/requester/httpclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestRequester_With(t *testing.T) {
	reqs, err := New(Method("red"))
	require.NoError(t, err)
	reqs2, err := reqs.With(Method("green"))
	require.NoError(t, err)
	// should clone first, then apply
	require.Equal(t, "green", reqs2.Method)
	require.Equal(t, "red", reqs.Method)

	t.Run("errors", func(t *testing.T) {
		reqs, err := New(Method("green"))
		require.NoError(t, err)
		reqs2, err := reqs.With(Method("red"), RelativeURL("cache_object:foo/bar"))
		require.Error(t, err)
		require.Nil(t, reqs2)
		require.Equal(t, "green", reqs.Method)
	})
}

func TestRequester_MustWith(t *testing.T) {
	reqs := MustNew(Method("red"))

	reqs2 := reqs.MustWith(Method("green"))

	// should clone first, then apply
	require.Equal(t, "green", reqs2.Method)
	require.Equal(t, "red", reqs.Method)

	// panics on error
	require.Panics(t, func() {
		reqs.MustWith(URL("cache_object:foo/bar"))
	})
}

func TestRequester_Apply(t *testing.T) {
	reqs, err := New(Method("red"))
	require.NoError(t, err)
	err = reqs.Apply(Method("green"))
	require.NoError(t, err)
	// applies in place
	require.Equal(t, "green", reqs.Method)

	t.Run("errors", func(t *testing.T) {
		err := reqs.Apply(URL("cache_object:foo/bar"))
		require.Error(t, err)
		require.Nil(t, reqs.URL)
	})
}

func TestRequester_MustApply(t *testing.T) {
	reqs, err := New(Method("red"))
	require.NoError(t, err)

	reqs.MustApply(Method("green"))
	// applies in place
	require.Equal(t, "green", reqs.Method)

	// panics on error
	require.Panics(t, func() {
		reqs.MustApply(URL("cache_object:foo/bar"))
	})
}

func TestURL(t *testing.T) {
	cases := []string{"http://a.io/", "http://b.io", "/relPath", "relPath", ""}
	for _, base := range cases {
		t.Run("", func(t *testing.T) {
			reqs, errFromNew := New(URL(base))
			u, err := url.Parse(base)
			if err == nil {
				require.Equal(t, u, reqs.URL)
			} else {
				require.EqualError(t, errFromNew, err.Error())
			}
		})
	}

	t.Run("errors", func(t *testing.T) {
		reqs, err := New(URL("cache_object:foo/bar"))
		require.Error(t, err)
		require.Nil(t, reqs)
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
			reqs, err := New()
			require.NoError(t, err)
			if c.base != "" {
				err := reqs.Apply(URL(c.base))
				require.NoError(t, err)
			}
			err = reqs.Apply(RelativeURL(c.relPath))
			require.NoError(t, err)
			require.Equal(t, c.expected, reqs.URL.String())
		})
	}

	t.Run("errors", func(t *testing.T) {
		reqs, err := New(URL("http://test.com/red"))
		require.NoError(t, err)
		err = reqs.Apply(RelativeURL("cache_object:foo/bar"))
		require.Error(t, err)
		require.Equal(t, "http://test.com/red", reqs.URL.String())
	})
}

func TestAppendPath(t *testing.T) {

	cases := []struct {
		name   string
		in     *Requester
		out    string
		append []string
	}{
		{
			name:   "basic",
			in:     MustNew(URL("")),
			append: []string{"blue", "green"},
			out:    "/blue/green",
		},
		{
			name:   "append",
			in:     MustNew(URL("/red")),
			append: []string{"blue", "green"},
			out:    "/red/blue/green",
		},
		{
			name:   "nil url",
			in:     &Requester{},
			append: []string{"blue", "green"},
			out:    "/blue/green",
		},
		{
			name:   "strip empty elements",
			in:     &Requester{},
			append: []string{"blue", "//", "/", "", "  ", " / / / ", "green"},
			out:    "/blue/green",
		},
		{
			name:   "strip slashes",
			in:     &Requester{},
			append: []string{"/blue/", "/green"},
			out:    "/blue/green",
		},
		{
			name:   "one",
			in:     &Requester{},
			append: []string{"blue"},
			out:    "/blue",
		},
		{
			name:   "three",
			in:     &Requester{},
			append: []string{"blue", "green", "red"},
			out:    "/blue/green/red",
		},
		{
			name:   "none",
			in:     MustNew(URL("/blue")),
			append: []string{},
			out:    "/blue",
		},
		{
			name:   "preserve trailing slash",
			in:     &Requester{},
			append: []string{"blue", "green/"},
			out:    "/blue/green/",
		},
		{
			name:   "add trailing slash",
			in:     &Requester{},
			append: []string{"blue", "green", "/"},
			out:    "/blue/green/",
		},
		{
			name:   "base trailing slash",
			in:     MustNew(URL("/blue/")),
			append: []string{},
			out:    "/blue/",
		},
		{
			name:   "preserve inner trailing slash",
			in:     MustNew(URL("/blue/")),
			append: []string{"green/red", "yellow"},
			out:    "/blue/green/red/yellow",
		},
		{
			name:   "preserve encoded stuff",
			in:     MustNew(URL("/blue/")),
			append: []string{url.PathEscape("green/red "), "yellow"},
			out:    "/blue/green%2Fred%20/yellow",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.in.MustWith(AppendPath(tc.append...)).URL.String()
			assert.Equal(t, tc.out, s)
		})
	}
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
			reqs, err := New(c.options...)
			require.NoError(t, err)
			require.Equal(t, c.expectedMethod, reqs.Method)
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
		{[]Option{AddHeader("content-tYPE", "application/json"), AddHeader("User-AGENT", "requester")}, http.Header{"Content-Type": {"application/json"}, "User-Agent": {"requester"}}},
		// values for existing keys should be appended
		{[]Option{AddHeader("A", "B"), AddHeader("a", "c")}, http.Header{"A": {"B", "c"}}},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			reqs, err := New(c.options...)
			require.NoError(t, err)
			require.Equal(t, c.expectedHeader, reqs.Header)
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
			reqs, err := New(c.options...)
			require.NoError(t, err)
			// type conversion from Header to alias'd map for deep equality comparison
			require.Equal(t, c.expectedHeader, reqs.Header)
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
			reqs, err := New(c.options...)
			require.NoError(t, err)
			req, err := reqs.RequestContext(context.Background())
			require.NoError(t, err)
			username, password, ok := req.BasicAuth()
			require.True(t, ok, "basic auth missing when expected")
			auth := []string{username, password}
			require.Equal(t, c.expectedAuth, auth)
		})
	}

	t.Run("delete header", func(t *testing.T) {
		r := MustNew(BasicAuth("bob", "red"))

		// assert the authorization header is set
		assert.NotEmpty(t, r.Header.Get(HeaderAuthorization))

		// applying the option with empty values deletes the header
		r.MustApply(BasicAuth("", ""))
		assert.Empty(t, r.Header.Get(HeaderAuthorization))

	})
}

func TestBearerAuth(t *testing.T) {
	cases := []string{
		"red",
		"",
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			reqs, err := New(BearerAuth(c))
			require.NoError(t, err)
			if c == "" {
				require.Empty(t, reqs.Header.Get("Authorization"))
			} else {
				require.Equal(t, "Bearer "+c, reqs.Header.Get("Authorization"))
			}
		})
	}

	t.Run("clearing", func(t *testing.T) {
		reqs, err := New(BearerAuth("green"))
		require.NoError(t, err)
		err = reqs.Apply(BearerAuth(""))
		require.NoError(t, err)
		_, ok := reqs.Header["Authorization"]
		require.False(t, ok, "should have removed Authorization header, instead was %s", reqs.Header.Get("Authorization"))
	})
}

func TestRange(t *testing.T) {
	s := MustNew(Range("bytes:1-2")).Header.Get("Range")
	assert.Equal(t, "bytes:1-2", s)
}

type FakeParams struct {
	KindName string `url:"kind_name"`
	Count    int    `url:"count"`
}

// Url-tagged query struct
var paramsA = struct {
	Limit int `url:"limit"`
}{
	30,
}
var paramsB = FakeParams{KindName: "recent", Count: 25}

var paramsC = struct {
	Color string     `url:"color"`
	Size  string     `url:"-"`
	Child FakeParams `url:"-"`
}{
	Color: "red",
	Size:  "big",
	Child: FakeParams{KindName: "car", Count: 4},
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
		{[]Option{QueryParams(paramsC)}, url.Values{"color": []string{"red"}}},
	}

	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			reqs, err := New(c.options...)
			require.NoError(t, err)
			require.Equal(t, c.expectedParams, reqs.QueryParams)
		})
	}
}

func TestQueryParam(t *testing.T) {
	reqs := MustNew(QueryParam("color", "red"))

	expected := url.Values{}
	expected.Add("color", "red")
	require.Equal(t, expected, reqs.QueryParams)

	reqs.MustApply(QueryParam("color", "blue"))

	expected.Add("color", "blue")
	require.Equal(t, expected, reqs.QueryParams)

	t.Run("empty key", func(t *testing.T) {
		// if key arg is empty, it's a no op
		assert.Nil(t, MustNew(QueryParam("", "red")).Header)
	})
}

func TestBody(t *testing.T) {
	reqs, err := New(Body("hey"))
	require.NoError(t, err)
	require.Equal(t, "hey", reqs.Body)
}

type testMarshaler struct{}

func (*testMarshaler) Unmarshal(_ []byte, _ string, _ interface{}) error {
	panic("implement me")
}

func (*testMarshaler) Marshal(_ interface{}) (data []byte, contentType string, err error) {
	panic("implement me")
}

func TestMarshaler(t *testing.T) {
	m := &testMarshaler{}
	reqs, err := New(WithMarshaler(m))
	require.NoError(t, err)
	require.Equal(t, m, reqs.Marshaler)
}

func TestUnmarshaler(t *testing.T) {
	m := &testMarshaler{}
	reqs, err := New(WithUnmarshaler(m))
	require.NoError(t, err)
	require.Equal(t, m, reqs.Unmarshaler)
}

func TestJSON(t *testing.T) {
	reqs, err := New(JSON(false))
	require.NoError(t, err)
	if assert.IsType(t, &JSONMarshaler{}, reqs.Marshaler) {
		assert.False(t, reqs.Marshaler.(*JSONMarshaler).Indent)
	}

	err = reqs.Apply(JSON(true))
	require.NoError(t, err)
	if assert.IsType(t, &JSONMarshaler{}, reqs.Marshaler) {
		assert.True(t, reqs.Marshaler.(*JSONMarshaler).Indent)
	}
}

func TestXML(t *testing.T) {
	reqs, err := New(XML(false))
	require.NoError(t, err)
	if assert.IsType(t, &XMLMarshaler{}, reqs.Marshaler) {
		assert.False(t, reqs.Marshaler.(*XMLMarshaler).Indent)
	}

	err = reqs.Apply(XML(true))
	require.NoError(t, err)
	if assert.IsType(t, &XMLMarshaler{}, reqs.Marshaler) {
		assert.True(t, reqs.Marshaler.(*XMLMarshaler).Indent)
	}
}

func TestForm(t *testing.T) {
	reqs, err := New(Form())
	require.NoError(t, err)
	assert.IsType(t, &FormMarshaler{}, reqs.Marshaler)
}

func TestUse(t *testing.T) {

	var outputs []string

	var mw Middleware = func(next Doer) Doer {
		outputs = append(outputs, "one")
		return next
	}
	var mw2 Middleware = func(next Doer) Doer {
		outputs = append(outputs, "two")
		return next
	}
	r := MustNew(Use(mw, mw2), MockDoer(200))
	r.Receive(nil)

	assert.Equal(t, []string{"two", "one"}, outputs)
	outputs = []string{}

	r.MustApply(Use(mw))
	r.Receive(nil)

	assert.Equal(t, []string{"one", "two", "one"}, outputs)
}

func ExampleAccept() {
	r := MustNew(Accept(MediaTypeJSON))

	fmt.Println(r.Headers().Get(HeaderAccept))

	// Output: application/json
}

func ExampleAddHeader() {
	r := MustNew(
		AddHeader("color", "red"),
		AddHeader("color", "blue"),
	)

	fmt.Println(r.Headers()["Color"])

	// Output: [red blue]
}

func ExampleDeleteHeader() {
	r := Requester{
		Header: http.Header{
			"Color":  []string{"red"},
			"Flavor": []string{"vanilla"},
		},
	}

	r.MustApply(DeleteHeader("color"))

	fmt.Println(r.Header)

	// Output: map[Flavor:[vanilla]]
}

func ExampleHeader() {
	r := MustNew(Header("color", "red"))

	fmt.Println(r.Header)

	// Output: map[Color:[red]]
}

func ExampleBasicAuth() {
	r := MustNew(BasicAuth("user", "password"))

	fmt.Println(r.Header.Get(HeaderAuthorization))

	// Output: Basic dXNlcjpwYXNzd29yZA==
}

func ExampleBearerAuth() {
	r := MustNew(BearerAuth("1234"))

	fmt.Println(r.Header.Get(HeaderAuthorization))

	// Output: Bearer 1234
}

func ExampleBody() {
	v := struct {
		Color string `json:"color"`
	}{
		Color: "red",
	}

	req, _ := Request(Body(v))

	b, _ := ioutil.ReadAll(req.Body)

	fmt.Println(string(b))

	// Output: {"color":"red"}
}

// The body value doesn't need to be a struct.  So long
// as the Marshaler can marshal it.
func ExampleBody_map() {
	req, _ := Request(Body(map[string]interface{}{"color": "red"}))

	b, _ := ioutil.ReadAll(req.Body)

	fmt.Println(string(b))

	// Output: {"color":"red"}
}

func ExampleBody_raw() {
	req, _ := Request(
		// all these are equivalent
		Body("red"),
		Body([]byte("red")),
		Body(strings.NewReader("red")),
	)

	b, _ := ioutil.ReadAll(req.Body)

	fmt.Println(string(b))

	// Output: red
}

func ExampleClient() {
	Send(
		URL("https://localhost:6060"),
		Client(httpclient.SkipVerify(true)),
	)
}

func ExampleContentType() {
	r := MustNew(ContentType(MediaTypeTextPlain))

	fmt.Println(r.Headers().Get(HeaderContentType))

	// Output: text/plain
}

func ExampleMethod() {
	r := MustNew(Method("CONNECT", "/resources/", "1"))

	fmt.Println(r.Method, r.URL.String())

	// Output: CONNECT /resources/1
}

func ExampleDelete() {
	r := MustNew(Delete("/resources/", "1"))

	fmt.Println(r.Method, r.URL.String())

	// Output: DELETE /resources/1
}

func ExamplePut() {
	r := MustNew(Put("/resources/", "1"))

	fmt.Println(r.Method, r.URL.String())

	// Output: PUT /resources/1
}

func ExamplePatch() {
	r := MustNew(Patch("/resources/", "1"))

	fmt.Println(r.Method, r.URL.String())

	// Output: PATCH /resources/1
}

func ExamplePost() {
	r := MustNew(Post("/resources/", "1"))

	fmt.Println(r.Method, r.URL.String())

	// Output: POST /resources/1
}

func ExampleGet() {
	r := MustNew(Get("/resources/", "1"))

	fmt.Println(r.Method, r.URL.String())

	// Output: GET /resources/1
}

func ExampleHead() {
	r := MustNew(Head("/resources/", "1"))

	fmt.Println(r.Method, r.URL.String())

	// Output: HEAD /resources/1
}

func ExampleHost() {
	r, _ := Request(Host("api.com"))

	fmt.Println(r.Host)

	// Output: api.com
}

func ExampleQueryParam() {
	r := MustNew(QueryParam("color", "red"))

	fmt.Println(r.Params().Encode())

	// Output: color=red
}

func ExampleQueryParams() {
	type Params struct {
		Color string `url:"color"`
	}

	// QueryParams option accepts several types
	r := MustNew(QueryParams(
		Params{Color: "red"},                   // struct with url tags
		map[string]string{"flavor": "vanilla"}, // map[string]string
		map[string][]string{"size": {"big"}},   // map[string][]string
		url.Values{"volume": []string{"loud"}}, // url.Values
	))

	// params already encoded in the URL are retained
	req, _ := r.Request(RelativeURL("?weight=heavy"))

	fmt.Println(req.URL.RawQuery)

	// Output: color=red&flavor=vanilla&size=big&volume=loud&weight=heavy
}

func ExampleRelativeURL() {
	r := MustNew(
		Get("http://test.com/green/"),
		// See the docs for url.URL#ResolveReference for details
		RelativeURL("red/", "blue"),
	)

	fmt.Println(r.URL.String())

	// Output: http://test.com/green/red/blue
}

func ExampleAppendPath() {

	r := MustNew(URL("http://test.com/users/bob"))

	fmt.Println("RelativeURL: " + r.MustWith(RelativeURL("frank")).URL.String())
	fmt.Println("AppendPath:  " + r.MustWith(AppendPath("frank")).URL.String())

	fmt.Println("RelativeURL: " + r.MustWith(RelativeURL("/frank")).URL.String())
	fmt.Println("AppendPath:  " + r.MustWith(AppendPath("/frank")).URL.String())

	fmt.Println("RelativeURL: " + r.MustWith(RelativeURL("frank", "nicknames")).URL.String())
	fmt.Println("AppendPath:  " + r.MustWith(AppendPath("frank", "nicknames")).URL.String())

	// Output:
	// RelativeURL: http://test.com/users/frank
	// AppendPath:  http://test.com/users/bob/frank
	// RelativeURL: http://test.com/frank
	// AppendPath:  http://test.com/users/bob/frank
	// RelativeURL: http://test.com/users/nicknames
	// AppendPath:  http://test.com/users/bob/frank/nicknames
}

func ExampleRequester_Clone() {
	base, _ := New(Get("https://api.io/"))

	foo := base.Clone()
	foo.Apply(Get("foo/"))

	bar := base.Clone()
	bar.Apply(Get("bar/"))

}
