package requester

import (
	"bytes"
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
// a timeout, temporary, or EOF error, or if the status code is 429, >=500, except for 501 (Not Implemented).
func DefaultShouldRetry(attempt int, req *http.Request, resp *http.Response, err error) bool {
	var netError net.Error

	switch {
	case errors.Is(err, io.EOF):
		return true
	case errors.As(err, &netError) && (netError.Temporary() || netError.Timeout()):
		return true
	case err != nil:
		return false
	case resp.StatusCode == 500, resp.StatusCode > 501, resp.StatusCode == 429:
		return true
	}

	return false
}

// OnlyIdempotentShouldRetry returns true if the request is using one of the HTTP methods which
// are intended to be idempotent: GET, HEAD, OPTIONS, and TRACE.  Should be combined with other criteria
// using AllRetryers(), for example:
//
//     c.ShouldRetry = AllRetryers(ShouldRetryerFunc(DefaultShouldRetry), ShouldRetryerFunc(OnlyIdempotentShouldRetry))
//
func OnlyIdempotentShouldRetry(_ int, req *http.Request, _ *http.Response, _ error) bool {
	switch req.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
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
	// ReadResponse will ensure the entire response is read before
	// consider the request a success
	ReadResponse bool
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

// AllRetryers returns a ShouldRetryer which returns true only if all the supplied retryers return true.
func AllRetryers(s ...ShouldRetryer) ShouldRetryer {
	return ShouldRetryerFunc(func(attempt int, req *http.Request, resp *http.Response, err error) bool {
		for _, shouldRetryer := range s {
			if !shouldRetryer.ShouldRetry(attempt, req, resp, err) {
				return false
			}
		}
		return true
	})
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

				// if ReadResponse, then also read the entire response into a buffer, to ensure no
				// error occurs
				if err == nil && c.ReadResponse {
					resp.Body, err = bufRespBody(resp.Body)
				}

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

type errCloser struct {
	io.Reader
	err error
}

func (e *errCloser) Close() error {
	return e.err
}

// bufRespBody reads all of b to memory and then returns a ReadCloser yielding
// the same bytes.  It returns an error if reading from the input fails.  If
// closing the input fails, it returns a ReadCloser with a Close() that returns
// this error.
func bufRespBody(b io.ReadCloser) (r io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		return b, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, err
	}
	if err := b.Close(); err != nil {
		return &errCloser{
			Reader: &buf,
			err:    err,
		}, nil
	}
	return io.NopCloser(&buf), nil
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
