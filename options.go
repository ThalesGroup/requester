package requests

import (
	"encoding/base64"
	"github.com/ansel1/merry"
	"github.com/ansel1/sling/clients"
	goquery "github.com/google/go-querystring/query"
	"net/http"
	"net/url"
)

// HTTP constants.
const (
	HeaderAccept        = "Accept"
	HeaderContentType   = "Content-Type"
	HeaderAuthorization = "Authorization"

	ContentTypeJSON = "application/json"
	ContentTypeXML  = "application/xml"
	ContentTypeForm = "application/x-www-form-urlencoded"
)

// Option applies some setting to a Requests object.  Options can be passed
// as arguments to most of Requests' methods.
type Option interface {

	// Apply modifies the Requests argument.  The Requests pointer will never be nil.
	// Returning an error will stop applying the request of the Options, and the error
	// will float up to the original caller.
	Apply(*Requests) error
}

// OptionFunc adapts a function to the Option interface.
type OptionFunc func(*Requests) error

// Apply implements Option.
func (f OptionFunc) Apply(r *Requests) error {
	return f(r)
}

// With clones the Requests object, then applies the options
// to the clone.
func (r *Requests) With(opts ...Option) (*Requests, error) {
	r2 := r.Clone()
	err := r2.Apply(opts...)
	if err != nil {
		return nil, err
	}
	return r2, nil
}

// Apply applies the options to the receiver.
func (r *Requests) Apply(opts ...Option) error {
	for _, o := range opts {
		err := o.Apply(r)
		if err != nil {
			return merry.Prepend(err, "applying options")
		}
	}
	return nil
}

// Method sets the HTTP method (e.g. GET/DELETE/etc).
// If path arguments are passed, they will be applied
// via the RelativeURL option.
func Method(m string, paths ...string) Option {
	return OptionFunc(func(r *Requests) error {
		r.Method = m
		if len(paths) > 0 {
			err := RelativeURL(paths...).Apply(r)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// Head sets the HTTP method to "HEAD".  Optional path arguments
// will be applied via the RelativeURL option.
func Head(paths ...string) Option {
	return Method("HEAD", paths...)
}

// Get sets the HTTP method to "GET".  Optional path arguments
// will be applied via the RelativeURL option.
func Get(paths ...string) Option {
	return Method("GET", paths...)
}

// Post sets the HTTP method to "POST".  Optional path arguments
// will be applied via the RelativeURL option.
func Post(paths ...string) Option {
	return Method("POST", paths...)
}

// Put sets the HTTP method to "PUT".  Optional path arguments
// will be applied via the RelativeURL option.
func Put(paths ...string) Option {
	return Method("PUT", paths...)
}

// Patch sets the HTTP method to "PATCH".  Optional path arguments
// will be applied via the RelativeURL option.
func Patch(paths ...string) Option {
	return Method("PATCH", paths...)
}

// Delete sets the HTTP method to "DELETE".  Optional path arguments
// will be applied via the RelativeURL option.
func Delete(paths ...string) Option {
	return Method("DELETE", paths...)
}

// AddHeader adds a header value, using Header.Add()
func AddHeader(key, value string) Option {
	return OptionFunc(func(b *Requests) error {
		if b.Header == nil {
			b.Header = make(http.Header)
		}
		b.Header.Add(key, value)
		return nil
	})
}

// Header sets a header value, using Header.Set()
func Header(key, value string) Option {
	return OptionFunc(func(b *Requests) error {
		if b.Header == nil {
			b.Header = make(http.Header)
		}
		b.Header.Set(key, value)
		return nil
	})
}

// DeleteHeader deletes a header key, using Header.Del()
func DeleteHeader(key string) Option {
	return OptionFunc(func(b *Requests) error {
		b.Header.Del(key)
		return nil
	})
}

// BasicAuth sets the Authorization header to "Basic <encoded username and password>".
// If username and password are empty, it deletes the Authorization header.
func BasicAuth(username, password string) Option {
	if username == "" && password == "" {
		return DeleteHeader(HeaderAuthorization)
	}
	return Header(HeaderAuthorization, "Basic "+basicAuth(username, password))
}

// basicAuth returns the base64 encoded username:password for basic auth copied
// from net/http.
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// BearerAuth sets the Authorization header to "Bearer <token>".
// If the token is empty, it deletes the Authorization header.
func BearerAuth(token string) Option {
	if token == "" {
		return DeleteHeader(HeaderAuthorization)
	}
	return Header(HeaderAuthorization, "Bearer "+token)
}

// URL sets the request URL.  Returns an error if arg is not
// a valid URL.
func URL(p string) Option {
	return OptionFunc(func(b *Requests) error {
		u, err := url.Parse(p)
		if err != nil {
			return merry.Prepend(err, "invalid url")
		}
		b.URL = u
		return nil
	})
}

// RelativeURL resolves the arg as a relative URL references against
// the current URL, using the standard lib's url.URL.ResolveReference() method.
// For example:
//
//     r, _ := requests.New(Get("http://test.com"), RelativeURL("red"))
//     fmt.Println(r.URL.String())  // http://test.com/red
//
// Multiple arguments will be resolved in order:
//
//     r, _ := requests.New(Get("http://test.com"), RelativeURL("red", "blue"))
//     fmt.Println(r.URL.String())  // http://test.com/red/blue
//
func RelativeURL(paths ...string) Option {
	return OptionFunc(func(r *Requests) error {
		for _, p := range paths {
			u, err := url.Parse(p)
			if err != nil {
				return merry.Prepend(err, "invalid url")
			}
			if r.URL == nil {
				r.URL = u
			} else {
				r.URL = r.URL.ResolveReference(u)
			}
		}
		return nil
	})
}

// QueryParams adds params to the Requests.QueryParams member.
// The arguments may be either map[string][]string, url.Values, or a struct.
// The argument values are merged into Requests.QueryParams, overriding existing
// values.
//
// If the arg is a struct, the struct is marshaled into a url.Values object using
// the github.com/google/go-querystring/query package.  Structs should tag
// their members with the "url" tag, e.g.:
//
//     type ReqParams struct {
//         Color string `url:"color"`
//     }
//
// An error will be returned if marshaling the struct fails.
func QueryParams(queryStructs ...interface{}) Option {
	return OptionFunc(func(s *Requests) error {
		if s.QueryParams == nil {
			s.QueryParams = url.Values{}
		}
		for _, queryStruct := range queryStructs {
			var values url.Values
			switch t := queryStruct.(type) {
			case nil:
			case map[string][]string:
				values = url.Values(t)
			case url.Values:
				values = t
			default:
				// encodes query structs into a url.Values map and merges maps
				var err error
				values, err = goquery.Values(queryStruct)
				if err != nil {
					return merry.Prepend(err, "invalid query struct")
				}
			}

			// merges new values into existing
			for key, values := range values {
				for _, value := range values {
					s.QueryParams.Add(key, value)
				}
			}
		}
		return nil
	})
}

// Body sets Requests.Body
func Body(body interface{}) Option {
	return OptionFunc(func(b *Requests) error {
		b.Body = body
		return nil
	})
}

// Marshaler sets Requests.Marshaler
func Marshaler(m BodyMarshaler) Option {
	return OptionFunc(func(b *Requests) error {
		b.Marshaler = m
		return nil
	})
}

// Unmarshaler sets Requests.Unmarshaler
func Unmarshaler(m BodyUnmarshaler) Option {
	return OptionFunc(func(b *Requests) error {
		b.Unmarshaler = m
		return nil
	})
}

// Accept sets the Accept header.
func Accept(accept string) Option {
	return Header(HeaderAccept, accept)
}

// ContentType sets the Content-Type header.
func ContentType(contentType string) Option {
	return Header(HeaderContentType, contentType)
}

// Host sets Requests.Host
func Host(host string) Option {
	return OptionFunc(func(b *Requests) error {
		b.Host = host
		return nil
	})
}

func joinOpts(opts ...Option) Option {
	return OptionFunc(func(r *Requests) error {
		for _, opt := range opts {
			err := opt.Apply(r)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// JSON sets Requests.Marshaler to the JSONMarshaler.
// If the arg is true, the generated JSON will be indented.
// The JSONMarshaler will set the Content-Type header to
// "application/json" unless explicitly overwritten.
func JSON(indent bool) Option {
	return joinOpts(
		Marshaler(&JSONMarshaler{Indent: indent}),
		ContentType(ContentTypeJSON),
		Accept(ContentTypeJSON),
	)
}

// XML sets Requests.Marshaler to the XMLMarshaler.
// If the arg is true, the generated XML will be indented.
// The XMLMarshaler will set the Content-Type header to
// "application/xml" unless explicitly overwritten.
func XML(indent bool) Option {
	return joinOpts(
		Marshaler(&XMLMarshaler{Indent: indent}),
		ContentType(ContentTypeXML),
		Accept(ContentTypeXML),
	)
}

// Form sets Requests.Marshaler to the FormMarshaler,
// which marshals the body into form-urlencoded.
// The FormMarshaler will set the Content-Type header to
// "application/x-www-form-urlencoded" unless explicitly overwritten.
func Form() Option {
	return Marshaler(&FormMarshaler{})
}

// Client replaces Requests.Doer with an *http.Client.  The client
// will be created and configured using the `clients` package.
func Client(opts ...clients.Option) Option {
	return OptionFunc(func(b *Requests) error {
		c, err := clients.NewClient(opts...)
		if err != nil {
			return err
		}
		b.Doer = c
		return nil
	})
}

// Use appends middlware to Requests.Middleware.  Middleware
// is invoked in the order added.
func Use(m ...Middleware) Option {
	return OptionFunc(func(r *Requests) error {
		r.Middleware = append(r.Middleware, m...)
		return nil
	})
}

// WithDoer replaces Requests.Doer.  If nil, Requests will
// revert to using the http.DefaultClient.
func WithDoer(d Doer) Option {
	return OptionFunc(func(r *Requests) error {
		r.Doer = d
		return nil
	})
}
