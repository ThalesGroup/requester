package requester

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/url"
	"testing"
)

func TestJSONMarshaler_Marshal(t *testing.T) {
	m := JSONMarshaler{}

	v := map[string]interface{}{"color": "red"}

	expected, err := json.Marshal(v)
	require.NoError(t, err)

	expectedIndented, err := json.MarshalIndent(v, "", "  ")
	require.NoError(t, err)

	d, ct, err := m.Marshal(v)
	require.NoError(t, err)
	require.Equal(t, "application/json; charset=UTF-8", ct)
	require.Equal(t, expected, d)

	m.Indent = true
	d, _, err = m.Marshal(v)
	require.NoError(t, err)
	require.Equal(t, expectedIndented, d)
}

func TestJSONMarshaler_Unmarshal(t *testing.T) {
	m := JSONMarshaler{}

	var v interface{}
	d := []byte(`{"color":"red"}`)
	err := m.Unmarshal(d, "", &v)
	require.NoError(t, err)

	require.Equal(t, map[string]interface{}{"color": "red"}, v)
}

type testModel struct {
	Color string `xml:"color" json:"color" url:"color"`
	Count int    `xml:"count" json:"count" url:"count"`
}

func TestXMLMarshaler_Marshal(t *testing.T) {
	m := XMLMarshaler{}

	b, ct, err := m.Marshal(testModel{"red", 30})
	require.NoError(t, err)

	assert.Equal(t, "application/xml; charset=UTF-8", ct)

	assert.Equal(t, `<testModel><color>red</color><count>30</count></testModel>`, string(b))

	m.Indent = true
	b, _, err = m.Marshal(testModel{"red", 30})
	require.NoError(t, err)

	assert.Equal(t, `<testModel>
  <color>red</color>
  <count>30</count>
</testModel>`, string(b))
}

func TestXMLMarshaler_Unmarshal(t *testing.T) {
	m := XMLMarshaler{}

	var v testModel
	data := []byte(`<testModel><color>red</color><count>30</count></testModel>`)
	err := m.Unmarshal(data, "", &v)
	require.NoError(t, err)

	assert.Equal(t, testModel{"red", 30}, v)
}

func TestMultiUnmarshaler_Unmarshal(t *testing.T) {
	m := MultiUnmarshaler{}

	cases := []struct {
		input       string
		contentType string
	}{
		{
			input:       `<testModel><color>red</color><count>30</count></testModel>`,
			contentType: `application/xml`,
		},
		{
			input:       `{"color":"red","count":30}`,
			contentType: `application/json`,
		},
	}
	for _, c := range cases {
		t.Run(c.contentType, func(t *testing.T) {
			var v testModel
			err := m.Unmarshal([]byte(c.input), c.contentType, &v)
			require.NoError(t, err)
			require.Equal(t, testModel{"red", 30}, v)
		})
	}

	t.Run("unknown", func(t *testing.T) {
		err := m.Unmarshal([]byte(`{"color":"red","count":30}`), "asdf", &testModel{})
		require.Error(t, err)
	})
}

func TestMultiUnmarshaler_Apply(t *testing.T) {
	r := MustNew()
	r.Marshaler = nil

	m := &MultiUnmarshaler{}
	r.MustApply(m)

	assert.Equal(t, m, r.Unmarshaler)
}

func TestContentTypeUnmarshaler_Unmarshal(t *testing.T) {
	m := NewContentTypeUnmarshaler()
	m.Unmarshalers["another/thing"] = &JSONMarshaler{}

	cases := []struct {
		input       string
		contentType string
	}{
		{
			input:       `<testModel><color>red</color><count>30</count></testModel>`,
			contentType: `application/xml`,
		},
		{
			input:       `{"color":"red","count":30}`,
			contentType: `application/json`,
		},
		{
			input:       `{"color":"red","count":30}`,
			contentType: `application/tree.subtype+json`,
		},
		{
			input:       `{"color":"red","count":30}`,
			contentType: `another/thing`,
		},
	}
	for _, c := range cases {
		t.Run(c.contentType, func(t *testing.T) {
			var v testModel
			err := m.Unmarshal([]byte(c.input), c.contentType, &v)
			require.NoError(t, err)
			require.Equal(t, testModel{"red", 30}, v)
		})
	}

	t.Run("unknown", func(t *testing.T) {
		err := m.Unmarshal([]byte(`{"color":"red","count":30}`), "application/unknown", &testModel{})
		require.Error(t, err)
	})

	t.Run("invalid media type", func(t *testing.T) {
		err := m.Unmarshal([]byte(`{"color":"red","count":30}`), "application|json", &testModel{})
		require.Error(t, err)
	})
}

func TestContentTypeUnmarshaler_Apply(t *testing.T) {
	r := MustNew()
	r.Marshaler = nil

	m := NewContentTypeUnmarshaler()
	r.MustApply(m)

	assert.Equal(t, m, r.Unmarshaler)
}

func TestFormMarshaler_Marshal(t *testing.T) {

	testCases := []struct {
		input  interface{}
		output string
	}{
		{
			input:  testModel{"red", 30},
			output: "color=red&count=30",
		},
		{
			input:  map[string][]string{"color": {"green", "red"}, "count": {"40"}},
			output: "color=green&color=red&count=40",
		},
		{
			input:  url.Values{"color": {"green", "red"}, "count": {"40"}},
			output: "color=green&color=red&count=40",
		},
		{
			input:  map[string]string{"color": "green", "count": "40"},
			output: "color=green&count=40",
		},
	}
	for _, testCase := range testCases {
		m := FormMarshaler{}
		d, ct, err := m.Marshal(testCase.input)

		require.NoError(t, err)
		assert.Equal(t, "application/x-www-form-urlencoded; charset=UTF-8", ct)
		assert.Equal(t, testCase.output, string(d))
	}

	// m := FormMarshaler{}
	// d, ct, err := m.Marshal(testModel{"red", 30})
	// require.NoError(t, err)
	//
	// assert.Equal(t, "application/x-www-form-urlencoded", ct)
	// assert.Equal(t, "color=red&count=30", string(d))
}

func TestMarshalFunc_Apply(t *testing.T) {
	var mf MarshalFunc = func(_ interface{}) (bytes []byte, s string, e error) {
		return nil, "red", nil
	}

	_, s, _ := MustNew(mf).Marshaler.Marshal(nil)
	assert.Equal(t, "red", s)
}

func ExampleFormMarshaler() {
	// *FormMarshaler implements Option, so it can be passed directly to functions
	// which accept Options.
	req, _ := Request(&FormMarshaler{}, Body(url.Values{"color": []string{"red"}}))

	b, _ := ioutil.ReadAll(req.Body)

	fmt.Println(string(b))
	fmt.Println(req.Header.Get(HeaderContentType))

	// Output:
	// color=red
	// application/x-www-form-urlencoded; charset=UTF-8
}

func ExampleJSONMarshaler() {
	// *JSONMarshaler implements Option, so it can be passed directly to functions
	// which accept Options.
	req, _ := Request(&JSONMarshaler{Indent: false}, Body(map[string]interface{}{"color": "red"}))

	b, _ := ioutil.ReadAll(req.Body)

	fmt.Println(string(b))
	fmt.Println(req.Header.Get(HeaderContentType))

	// Output:
	// {"color":"red"}
	// application/json; charset=UTF-8
}

func ExampleXMLMarshaler() {
	type Resource struct {
		Color string
	}

	// *XMLMarshaler implements Option, so it can be passed directly to functions
	// which accept Options.
	req, _ := Request(&XMLMarshaler{Indent: false}, Body(Resource{Color: "red"}))

	b, _ := ioutil.ReadAll(req.Body)

	fmt.Println(string(b))
	fmt.Println(req.Header.Get(HeaderContentType))

	// Output:
	// <Resource><Color>red</Color></Resource>
	// application/xml; charset=UTF-8
}
