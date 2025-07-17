package requester

import (
	"encoding/json"
	"encoding/xml"
	"github.com/ansel1/merry"
	goquery "github.com/google/go-querystring/query"
	"mime"
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
//
//nolint:gochecknoglobals
var DefaultMarshaler Marshaler = &JSONMarshaler{}

// DefaultUnmarshaler is used by Requester if Requester.Unmarshaler is nil.
//
//nolint:gochecknoglobals
var DefaultUnmarshaler Unmarshaler = NewContentTypeUnmarshaler()

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
//	r := requester.Requester{
//	    Body: &JSONMarshaler{},
//	}
type JSONMarshaler struct {
	Indent bool
}

// Unmarshal implements Unmarshaler.
func (m *JSONMarshaler) Unmarshal(data []byte, _ string, v interface{}) error {
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
//	r := requester.Requester{
//	    Marshaler: &XMLMarshaler{},
//	}
type XMLMarshaler struct {
	Indent bool
}

// Unmarshal implements Unmarshaler.
func (*XMLMarshaler) Unmarshal(data []byte, _ string, v interface{}) error {
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

// ContentTypeUnmarshaler selects an unmarshaler based on the content type, which should be a
// valid media/mime type, in the form:
//
//	type "/" [tree "."] subtype ["+" suffix] *[";" parameter]
//
// Unmarshalers are registered to handle a given media type.  Parameters are ignored:
//
//	ct := NewContentTypeUnmarshaler()
//	ct.Unmarshalers["application/json"] = &JSONMarshaler{}
//
// If the full media type has no match, but there is a suffix, it will look for an Unmarshaler
// registered for <type>/<suffix>.  For example, if there was no match for `application/vnd.api+json`,
// it will look for `application/json`.
type ContentTypeUnmarshaler struct {
	Unmarshalers map[string]Unmarshaler
}

// NewContentTypeUnmarshaler returns a new ContentTypeUnmarshaler preconfigured to
// handle application/json and application/xml.
func NewContentTypeUnmarshaler() *ContentTypeUnmarshaler {
	// install defaults
	return &ContentTypeUnmarshaler{
		Unmarshalers: defaultUnmarshalers(),
	}
}

func defaultUnmarshalers() map[string]Unmarshaler {
	return map[string]Unmarshaler{
		MediaTypeJSON: &JSONMarshaler{},
		MediaTypeXML:  &XMLMarshaler{},
	}
}

// Unmarshal implements Unmarshaler.
//
// If media type parsing fails, or no Unmarshaler is found, an error is returned.
func (c *ContentTypeUnmarshaler) Unmarshal(data []byte, contentType string, v interface{}) error {
	// for zero value ContentTypeUnmarshaler, initialize with defaults.
	// This allows ContentTypeUnmarshaler to be a drop in replacement for MultiUnmarshaler
	if c.Unmarshalers == nil {
		c.Unmarshalers = map[string]Unmarshaler{
			MediaTypeJSON: &JSONMarshaler{},
			MediaTypeXML:  &XMLMarshaler{},
		}
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return merry.Prependf(err, "failed to parse content type: %s", contentType)
	}

	if u := c.Unmarshalers[mediaType]; u != nil {
		return u.Unmarshal(data, contentType, v)
	}

	// If exact match didn't find anything, try falling back to a looser match.
	if ct := generalMediaType(mediaType); ct != "" {
		if u := c.Unmarshalers[ct]; u != nil {
			return u.Unmarshal(data, contentType, v)
		}
	}

	return merry.Errorf("unsupported content type: %s", contentType)
}

// Apply implements Option
func (c *ContentTypeUnmarshaler) Apply(r *Requester) error {
	r.Unmarshaler = c
	return nil
}

// Media types can have a suffix which indicates the underlying data structure,
// e.g. application/vnd.api+json might indicate a payload that meets a strict API
// schema, but the +suffix indicates the underlying data structure is JSON.
// This will return a media type with just the suffix as the subtype, e.g.
// application/vnd.api+json -> application/json
//
// If the media type isn't correctly formatted, the subtype has no suffix, or the suffix
// is empty, this returns an empty string.
func generalMediaType(s string) string {
	i2 := strings.LastIndex(s, "+")
	if i2 > -1 && len(s) > i2+1 { // has a non-zero length suffix
		i := strings.Index(s, "/")
		if i > -1 {
			return s[:i+1] + s[i2+1:]
		}
	}
	return ""
}

// MultiUnmarshaler is a legacy alias for ContentTypeUnmarshaler.
type MultiUnmarshaler = ContentTypeUnmarshaler
