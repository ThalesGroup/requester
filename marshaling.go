package requester

import (
	"encoding/json"
	"encoding/xml"
	"github.com/ansel1/merry"
	goquery "github.com/google/go-querystring/query"
	"net/url"
	"strings"
)

// Requester marshals values into request bodies, and unmarshals
// response bodies into structs, using instances of the
// Marshaler and Unmarshaler interfaces.  Implementations of these can
// be installed in a requester with the WithMarshaler and WithUnmarshaler Options.
//
// This package comes with a number of implementations built in, which can
// be installed with the JSON(), XML(), and Form() Options.
//
// If not set, requesters fall back on the DefaultMarshaler and
// DefaultUnmarshaler.  The DefaultMarshaler marshals into JSON, and the
// DefaultUnmarshaler uses the response's Content-Type header to
// determine which unmarshaler to delegate it.  It supports JSON and XML.

// DefaultMarshaler is used by Requester if Requester.Marshaler is nil.
// nolint:gochecknoglobals
var DefaultMarshaler Marshaler = &JSONMarshaler{}

// DefaultUnmarshaler is used by Requester if Requester.Unmarshaler is nil.
// nolint:gochecknoglobals
var DefaultUnmarshaler Unmarshaler = &MultiUnmarshaler{}

const (
	contentTypeForm = MediaTypeForm + "; charset=UTF-8"
	contentTypeXML  = MediaTypeXML + "; charset=UTF-8"
	contentTypeJSON = MediaTypeJSON + "; charset=UTF-8"
)

// Marshaler marshals values into a []byte.
//
// If the content type returned is not empty, it
// will be used in the request's Content-Type header.
type Marshaler interface {
	Marshal(v interface{}) (data []byte, contentType string, err error)
}

// Unmarshaler unmarshals a []byte response body into a value.  It is provided
// the value of the Content-Type header from the response.
type Unmarshaler interface {
	Unmarshal(data []byte, contentType string, v interface{}) error
}

// MarshalFunc adapts a function to the Marshaler interface.
type MarshalFunc func(v interface{}) ([]byte, string, error)

// Apply implements Option.  MarshalFunc can be applied as a requester option, which
// install itself as the Marshaler.
func (f MarshalFunc) Apply(r *Requester) error {
	r.Marshaler = f
	return nil
}

// Marshal implements the Marshaler interface.
func (f MarshalFunc) Marshal(v interface{}) ([]byte, string, error) {
	return f(v)
}

// UnmarshalFunc adapts a function to the Unmarshaler interface.
type UnmarshalFunc func(data []byte, contentType string, v interface{}) error

// Apply implements Option.  UnmarshalFunc can be applied as a requester option, which
// install itself as the Unmarshaler.
func (f UnmarshalFunc) Apply(r *Requester) error {
	r.Unmarshaler = f
	return nil
}

// Unmarshal implements the Unmarshaler interface.
func (f UnmarshalFunc) Unmarshal(data []byte, contentType string, v interface{}) error {
	return f(data, contentType, v)
}

// JSONMarshaler implement Marshaler and Unmarshaler.  It marshals values to and
// from JSON.  If Indent is true, marshaled JSON will be indented.
//
//   r := requester.Requester{
//       Body: &JSONMarshaler{},
//   }
//
type JSONMarshaler struct {
	Indent bool
}

// Unmarshal implements Unmarshaler.
func (m *JSONMarshaler) Unmarshal(data []byte, contentType string, v interface{}) error {
	return merry.Wrap(json.Unmarshal(data, v))
}

// Marshal implements Marshaler.
func (m *JSONMarshaler) Marshal(v interface{}) (data []byte, contentType string, err error) {
	if m.Indent {
		data, err = json.MarshalIndent(v, "", "  ")
	} else {
		data, err = json.Marshal(v)
	}

	return data, contentTypeJSON, merry.Wrap(err)
}

// Apply implements Option.
func (m *JSONMarshaler) Apply(r *Requester) error {
	r.Marshaler = m
	return nil
}

// XMLMarshaler implements Marshaler and Unmarshaler.  It marshals values to
// and from XML.  If Indent is true, marshaled XML will be indented.
//
//     r := requester.Requester{
//         Marshaler: &XMLMarshaler{},
//     }
//
type XMLMarshaler struct {
	Indent bool
}

// Unmarshal implements Unmarshaler.
func (*XMLMarshaler) Unmarshal(data []byte, contentType string, v interface{}) error {
	return merry.Wrap(xml.Unmarshal(data, v))
}

// Marshal implements Marshaler.
func (m *XMLMarshaler) Marshal(v interface{}) (data []byte, contentType string, err error) {
	if m.Indent {
		data, err = xml.MarshalIndent(v, "", "  ")
	} else {
		data, err = xml.Marshal(v)
	}
	return data, contentTypeXML, merry.Wrap(err)
}

// Apply implements Option.
func (m *XMLMarshaler) Apply(r *Requester) error {
	r.Marshaler = m
	return nil
}

// FormMarshaler implements Marshaler.  It marshals values into URL-Encoded form data.
//
// The value can be either a map[string][]string, map[string]string, url.Values, or a struct with `url` tags.
type FormMarshaler struct{}

// Marshal implements Marshaler.
func (*FormMarshaler) Marshal(v interface{}) (data []byte, contentType string, err error) {
	switch t := v.(type) {
	case map[string][]string:
		urlV := url.Values(t)
		return []byte(urlV.Encode()), contentTypeForm, nil
	case map[string]string:
		urlV := url.Values{}
		for key, value := range t {
			urlV.Set(key, value)
		}
		return []byte(urlV.Encode()), contentTypeForm, nil
	case url.Values:
		return []byte(t.Encode()), contentTypeForm, nil
	default:
		values, err := goquery.Values(v)
		if err != nil {
			return nil, "", merry.Prepend(err, "invalid form struct")
		}
		return []byte(values.Encode()), contentTypeForm, nil
	}
}

// Apply implements Option.
func (m *FormMarshaler) Apply(r *Requester) error {
	r.Marshaler = m
	return nil
}

// MultiUnmarshaler implements Unmarshaler.  It uses the value of the Content-Type header in the
// response to choose between the JSON and XML unmarshalers.  If Content-Type is something else,
// an error is returned.
//
// MultiUnmarshaler is the default Unmarshaler.
type MultiUnmarshaler struct {
	jsonMar JSONMarshaler
	xmlMar  XMLMarshaler
}

// Unmarshal implements Unmarshaler.
func (m *MultiUnmarshaler) Unmarshal(data []byte, contentType string, v interface{}) error {
	switch {
	case strings.Contains(contentType, MediaTypeJSON):
		return m.jsonMar.Unmarshal(data, contentType, v)
	case strings.Contains(contentType, MediaTypeXML):
		return m.xmlMar.Unmarshal(data, contentType, v)
	}
	return merry.Errorf("unsupported content type: %s", contentType)
}

// Apply implements Unmarshaler
func (m *MultiUnmarshaler) Apply(r *Requester) error {
	r.Unmarshaler = m
	return nil
}
