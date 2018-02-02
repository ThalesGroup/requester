package requester

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/ansel1/merry"
	goquery "github.com/google/go-querystring/query"
	"net/url"
	"strings"
)

// DefaultMarshaler is used by Requester if Requester.Marshaler is nil.
var DefaultMarshaler BodyMarshaler = &JSONMarshaler{}

// DefaultUnmarshaler is used by Requester if Requester.Unmarshaler is nil.
var DefaultUnmarshaler BodyUnmarshaler = &MultiUnmarshaler{}

// BodyMarshaler marshals structs into a []byte, and supplies a matching
// Content-Type header.
type BodyMarshaler interface {
	Marshal(v interface{}) (data []byte, contentType string, err error)
}

// BodyUnmarshaler unmarshals a []byte response body into a value.  It is provided
// the value of the Content-Type header from the response.
type BodyUnmarshaler interface {
	Unmarshal(data []byte, contentType string, v interface{}) error
}

// MarshalFunc adapts a function to the BodyMarshaler interface.
type MarshalFunc func(v interface{}) ([]byte, string, error)

// Marshal implements the BodyMarshaler interface.
func (f MarshalFunc) Marshal(v interface{}) ([]byte, string, error) {
	return f(v)
}

// UnmarshalFunc adapts a function to the BodyUnmarshaler interface.
type UnmarshalFunc func(data []byte, contentType string, v interface{}) error

// Unmarshal implements the BodyUnmarshaler interface.
func (f UnmarshalFunc) Unmarshal(data []byte, contentType string, v interface{}) error {
	return f(data, contentType, v)
}

// JSONMarshaler implement BodyMarshaler and BodyUnmarshaler.  It marshals values to and
// from JSON.  If Indent is true, marshaled JSON will be indented.
//
//   r := requester.Requester{
//       Body: &JSONMarshaler{},
//   }
//
type JSONMarshaler struct {
	Indent bool
}

// Unmarshal implements BodyUnmarshaler.
func (m *JSONMarshaler) Unmarshal(data []byte, contentType string, v interface{}) error {
	return json.Unmarshal(data, v)
}

// Marshal implements BodyMarshaler.
func (m *JSONMarshaler) Marshal(v interface{}) (data []byte, contentType string, err error) {
	if m.Indent {
		data, err = json.MarshalIndent(v, "", "  ")
	} else {
		data, err = json.Marshal(v)
	}

	return data, MediaTypeJSON, err
}

// XMLMarshaler implements BodyMarshaler and BodyUnmarshaler.  It marshals values to
// and from XML.  If Indent is true, marshaled XML will be indented.
//
//     r := requester.Requester{
//         Marshaler: &XMLMarshaler{},
//     }
//
type XMLMarshaler struct {
	Indent bool
}

// Unmarshal implements BodyUnmarshaler.
func (*XMLMarshaler) Unmarshal(data []byte, contentType string, v interface{}) error {
	return xml.Unmarshal(data, v)
}

// Marshal implements BodyMarshaler.
func (m *XMLMarshaler) Marshal(v interface{}) (data []byte, contentType string, err error) {
	if m.Indent {
		data, err = xml.MarshalIndent(v, "", "  ")
	} else {
		data, err = xml.Marshal(v)
	}
	return data, MediaTypeXML, err
}

// FormMarshaler implements BodyMarshaler.  It marshals values into URL-Encoded form data.
//
// The value can be either a map[string][]string, map[string]string, url.Values, or a struct with `url` tags.
type FormMarshaler struct{}

// Marshal implements BodyMarshaler.
func (*FormMarshaler) Marshal(v interface{}) (data []byte, contentType string, err error) {
	switch t := v.(type) {
	case map[string][]string:
		urlV := url.Values(t)
		return []byte(urlV.Encode()), MediaTypeForm, nil
	case map[string]string:
		urlV := url.Values{}
		for key, value := range t {
			urlV.Set(key, value)
		}
		return []byte(urlV.Encode()), MediaTypeForm, nil
	case url.Values:
		return []byte(t.Encode()), MediaTypeForm, nil
	default:
		values, err := goquery.Values(v)
		if err != nil {
			return nil, "", merry.Prepend(err, "invalid form struct")
		}
		return []byte(values.Encode()), MediaTypeForm, nil
	}
}

// MultiUnmarshaler implements BodyUnmarshaler.  It uses the value of the Content-Type header in the
// response to choose between the JSON and XML unmarshalers.  If Content-Type is something else,
// an error is returned.
type MultiUnmarshaler struct {
	jsonMar JSONMarshaler
	xmlMar  XMLMarshaler
}

// Unmarshal implements BodyUnmarshaler.
func (m *MultiUnmarshaler) Unmarshal(data []byte, contentType string, v interface{}) error {
	switch {
	case strings.Contains(contentType, MediaTypeJSON):
		return m.jsonMar.Unmarshal(data, contentType, v)
	case strings.Contains(contentType, MediaTypeXML):
		return m.xmlMar.Unmarshal(data, contentType, v)
	}
	return fmt.Errorf("unsupported content type: %s", contentType)
}
