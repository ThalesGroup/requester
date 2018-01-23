package httpclient

import (
	"crypto/tls"
	"github.com/ansel1/merry"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

// NoRedirects configures the client to no perform any redirects.
func NoRedirects() Option {
	return OptionFunc(func(client *http.Client) error {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
		return nil
	})
}

// MaxRedirects configures the max number of redirects the client will perform before
// giving up.
func MaxRedirects(max int) Option {
	return OptionFunc(func(client *http.Client) error {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= max {
				return merry.Errorf("stopped after max %d requests", len(via))
			}
			return nil
		}
		return nil
	})
}

// CookieJar installs a cookie jar into the client, configured with the options argument.
//
// The argument will be nil.
func CookieJar(opts *cookiejar.Options) Option {
	return OptionFunc(func(client *http.Client) error {
		jar, err := cookiejar.New(opts)
		if err != nil {
			return merry.Wrap(err)
		}
		client.Jar = jar
		return nil
	})
}

// ProxyURL will proxy all calls through a single proxy URL.
func ProxyURL(proxyURL string) Option {
	return TransportOption(func(t *http.Transport) error {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return merry.Wrap(err)
		}
		t.Proxy = func(request *http.Request) (*url.URL, error) {
			return u, nil
		}
		return nil
	})
}

// ProxyFunc configures the client's proxy function.
func ProxyFunc(f func(request *http.Request) (*url.URL, error)) Option {
	return TransportOption(func(t *http.Transport) error {
		t.Proxy = f
		return nil
	})
}

// Timeout configures the client's Timeout property.
func Timeout(d time.Duration) Option {
	return OptionFunc(func(client *http.Client) error {
		client.Timeout = d
		return nil
	})
}

// SkipVerify sets the TLS config's InsecureSkipVerify flag.
func SkipVerify(skip bool) Option {
	return TLSOption(func(c *tls.Config) error {
		c.InsecureSkipVerify = skip
		return nil
	})
}
