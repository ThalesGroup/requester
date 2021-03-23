package requester

import (
	"errors"
	"github.com/ansel1/merry"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"time"
)

// DefaultRetryConfig is the default retry configuration used if nil is passed to Retry().
// nolint:gochecknoglobals
var DefaultRetryConfig = RetryConfig{}

// DefaultBackoff is a backoff configuration with the default values.
// nolint:gochecknoglobals
var DefaultBackoff = ExponentialBackoff{
	BaseDelay:  1.0 * time.Second,
	Multiplier: 1.6,
	Jitter:     0.2,
	MaxDelay:   120 * time.Second,
}

// DefaultShouldRetry is the default ShouldRetryer.  It retries the request if the error is
// a timeout, temporary, or EOF error, or if the status code is >=500, except for 501 (Not Implemented).
func DefaultShouldRetry(attempt int, req *http.Request, resp *http.Response, err error) bool {
	var netError net.Error

	switch {
	case errors.Is(err, io.EOF):
		return true
	case errors.As(err, &netError) && (netError.Temporary() || netError.Timeout()):
		return true
	case err != nil:
		return false
	case resp.StatusCode == 500, resp.StatusCode > 501:
		return true
	}

	return false
}

// RetryConfig defines settings for the Retry middleware.
type RetryConfig struct {
	// MaxAttempts is the number of times to attempt the request.
	// Defaults to 3.
	MaxAttempts int
	// ShouldRetry tests whether a response should be retried.  Defaults to
	// DefaultShouldRetry, which retries for temporary network errors, network
	// timeout errors, or response codes >= 500, except for 501.
	ShouldRetry ShouldRetryer
	// Backoff returns how long to wait between retries.  Defaults to
	// an exponential backoff with some jitter.
	Backoff Backoffer
}

func (c *RetryConfig) normalize() {
	if c.Backoff == nil {
		c.Backoff = &DefaultBackoff
	}

	if c.ShouldRetry == nil {
		c.ShouldRetry = ShouldRetryerFunc(DefaultShouldRetry)
	}

	if c.MaxAttempts < 1 {
		c.MaxAttempts = 3
	}
}

// ShouldRetryer evaluates whether an HTTP request should be retried.  resp may be nil.  Attempt is the number of
// the attempt which was just completed, and starts at 1.  For example, if attempt=1, ShouldRetry should return true
// if attempt 2 should be tried.
type ShouldRetryer interface {
	ShouldRetry(attempt int, req *http.Request, resp *http.Response, err error) bool
}

// ShouldRetryerFunc adapts a function to the ShouldRetryer interface
type ShouldRetryerFunc func(attempt int, req *http.Request, resp *http.Response, err error) bool

// ShouldRetry implements ShouldRetryer
func (s ShouldRetryerFunc) ShouldRetry(attempt int, req *http.Request, resp *http.Response, err error) bool {
	return s(attempt, req, resp, err)
}

// Backoffer calculates how long to wait between attempts.  The attempt argument is the attempt which
// just completed, and starts at 1.  So attempt=1 should return the time to wait between attempt 1 and 2.
type Backoffer interface {
	Backoff(attempt int) time.Duration
}

// BackofferFunc adapts a function to the Backoffer interface.
type BackofferFunc func(int) time.Duration

// Backoff implements Backoffer
func (b BackofferFunc) Backoff(attempt int) time.Duration {
	return b(attempt)
}

// ExponentialBackoff defines the configuration options for an exponential backoff strategy.
// The implementation is based on the one from grpc.
type ExponentialBackoff struct {
	// BaseDelay is the amount of time to backoff after the first failure.
	BaseDelay time.Duration
	// Multiplier is the factor with which to multiply backoffs after a
	// failed retry. Should ideally be greater than 1.
	Multiplier float64
	// Jitter is the factor with which backoffs are randomized.
	Jitter float64
	// MaxDelay is the upper bound of backoff delay.
	MaxDelay time.Duration
}

func (c *ExponentialBackoff) Backoff(attempt int) time.Duration {
	if attempt == 1 {
		return c.BaseDelay
	}

	backoff, max := float64(c.BaseDelay), float64(c.MaxDelay)
	for backoff < max && attempt > 1 {
		backoff *= c.Multiplier
		attempt--
	}

	if backoff > max {
		backoff = max
	}
	// Randomize backoff delays so that if a cluster of requests start at
	// the same time, they won't operate in lockstep.
	// nolint:gosec
	backoff *= 1 + c.Jitter*(rand.Float64()*2-1)
	if backoff < 0 {
		return 0
	}

	return time.Duration(backoff)
}

// Retry retries the http request under certain conditions.  The number of retries,
// retry conditions, and the time to sleep between retries can be configured.  If
// config is nil, the DefaultRetryConfig will be used.
//
// Requests with bodies can only be retried if the request's GetBody function is
// set.  It will be used to rewind the request body for the next attempt.  This
// is set automatically for most body types, like strings, byte slices, string readers,
// or byte readers.
func Retry(config *RetryConfig) Middleware {
	var c RetryConfig
	if config == nil {
		c = DefaultRetryConfig
	} else {
		c = *config
	}

	c.normalize()

	return func(next Doer) Doer {
		return DoerFunc(func(req *http.Request) (*http.Response, error) {
			// if GetBody is not set, we can't retry anyway
			if req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
				return next.Do(req)
			}

			var resp *http.Response
			var err error
			var attempt int
			for {
				resp, err = next.Do(req)
				attempt++
				if attempt >= c.MaxAttempts || !c.ShouldRetry.ShouldRetry(attempt, req, resp, err) {
					break
				}

				// if we're going to retry, we need to fulfill some responsibilities of an http.Request consumer
				// in particular, we need to drain and close the request body.  We drain it so keepAlive connections
				// can be reused.
				if resp != nil {
					drain(resp.Body)
				}

				req, err = resetRequest(req)
				if err != nil {
					return resp, err
				}

				// sleep for backoff
				select {
				case <-req.Context().Done():
					return nil, req.Context().Err()
				case <-time.After(c.Backoff.Backoff(attempt)):
				}
			}
			return resp, err
		})
	}
}

func resetRequest(req *http.Request) (*http.Request, error) {
	// shallow copy the req.  The persistConn.writeLoop deep in the http package reads from the req on
	// another goroutine, so we can't modify it in place.
	copyReq := *req
	req = &copyReq

	// If the body was not null, get a new body.  GetBody should never be nil here, because we checked
	// for that earlier
	if req.Body != nil && req.Body != http.NoBody {
		b, err := req.GetBody()
		if err != nil {
			return nil, merry.Prepend(err, "calling req.GetBody")
		}

		req.Body = b
	}

	return req, nil
}

func drain(r io.ReadCloser) {
	if r == nil {
		return
	}
	defer func(r io.ReadCloser) {
		_ = r.Close()
	}(r)

	_, _ = io.Copy(ioutil.Discard, io.LimitReader(r, 4096))
}
