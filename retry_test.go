package requester_test

import (
	"context"
	. "github.com/gemalto/requester"
	"github.com/gemalto/requester/httptestutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestExponentialBackoff_Backoff(t *testing.T) {
	tests := []struct {
		name           string
		backoff        ExponentialBackoff
		expected       [5]time.Duration
		expectedJitter float64
	}{
		{
			name: "zero base delay",
			backoff: ExponentialBackoff{
				BaseDelay:  0,
				Multiplier: 1,
				Jitter:     1,
				MaxDelay:   time.Second,
			},
			expected: [5]time.Duration{0, 0, 0, 0, 0},
		},
		{
			name: "zero multiplier",
			backoff: ExponentialBackoff{
				BaseDelay:  time.Second,
				Multiplier: 0,
				Jitter:     .2,
				MaxDelay:   time.Minute,
			},
			expected:       [5]time.Duration{time.Second, time.Second, time.Second, time.Second, time.Second},
			expectedJitter: 0.2,
		},
		{
			name: "zero jitter",
			backoff: ExponentialBackoff{
				BaseDelay:  1,
				Multiplier: 2,
				Jitter:     0,
				MaxDelay:   time.Second,
			},
			expected: [5]time.Duration{1, 2, 4, 8, 16},
		},
		{
			name: "zero max",
			backoff: ExponentialBackoff{
				BaseDelay:  1,
				Multiplier: 2,
				Jitter:     0,
				MaxDelay:   0,
			},
			expected: [5]time.Duration{1, 2, 4, 8, 16},
		},
		{
			name: "constant",
			backoff: ExponentialBackoff{
				BaseDelay:  30,
				Multiplier: 0,
				Jitter:     0,
				MaxDelay:   time.Second,
			},
			expected: [5]time.Duration{30, 30, 30, 30, 30},
		},
		{
			name: "max",
			backoff: ExponentialBackoff{
				BaseDelay:  30,
				Multiplier: 2,
				Jitter:     0,
				MaxDelay:   100,
			},
			expected: [5]time.Duration{30, 60, 100, 100, 100},
		},
		{
			name: "jitter",
			backoff: ExponentialBackoff{
				BaseDelay:  time.Second,
				Multiplier: 2,
				Jitter:     .1,
				MaxDelay:   time.Minute,
			},
			expected:       [5]time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second},
			expectedJitter: 0.1,
		},
		{
			name: "base more than max",
			backoff: ExponentialBackoff{
				BaseDelay:  2 * time.Second,
				Multiplier: 0,
				Jitter:     0,
				MaxDelay:   time.Second,
			},
			expected: [5]time.Duration{time.Second, time.Second, time.Second, time.Second, time.Second},
		},
		{
			name:     "no delay",
			backoff:  ExponentialBackoff{},
			expected: [5]time.Duration{0, 0, 0, 0, 0},
		},
		{
			name: "fixed delay",
			backoff: ExponentialBackoff{
				BaseDelay: time.Second,
			},
			expected: [5]time.Duration{time.Second, time.Second, time.Second, time.Second, time.Second},
		},
		{
			name: "fixed delay with jitter",
			backoff: ExponentialBackoff{
				BaseDelay: time.Second,
				Jitter:    .2,
			},
			expected:       [5]time.Duration{time.Second, time.Second, time.Second, time.Second, time.Second},
			expectedJitter: 0.2,
		},
		{
			name: "jitter wont go over max",
			backoff: ExponentialBackoff{
				BaseDelay: time.Second,
				Jitter:    .2,
				MaxDelay:  time.Second,
			},
			expected:       [5]time.Duration{900 * time.Millisecond, 900 * time.Millisecond, 900 * time.Millisecond, 900 * time.Millisecond, 900 * time.Millisecond},
			expectedJitter: 0.1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var results [5]time.Duration
			for i := 0; i < 5; i++ {
				results[i] = test.backoff.Backoff(i + 1)
			}
			if test.expectedJitter > 0 {
				for i, duration := range test.expected {
					assert.InDelta(t, duration, results[i], float64(duration)*test.backoff.Jitter)
				}
				assert.NotEqual(t, test.expected, results, "shouldn't be exactly equal, missing the jitter")
			} else {
				assert.Equal(t, test.expected, results)
			}
			if test.backoff.MaxDelay > 0 {
				for _, duration := range results {
					assert.LessOrEqual(t, duration, test.backoff.MaxDelay)
				}
			}
		})
	}
}

type netError struct {
	timeout bool
}

func (m *netError) Error() string {
	return "neterror"
}

func (m *netError) Timeout() bool {
	return m.timeout
}

func (m *netError) Temporary() bool {
	return false
}

func TestDefaultShouldRetry(t *testing.T) {
	assert.True(t, DefaultShouldRetry(1, nil, nil, &net.OpError{
		Op:  "accept",
		Err: syscall.ECONNRESET,
	}))
	assert.True(t, DefaultShouldRetry(1, nil, nil, &net.OpError{
		Op:  "accept",
		Err: syscall.ECONNABORTED,
	}))
	assert.True(t, DefaultShouldRetry(1, nil, nil, syscall.EPIPE))
	assert.True(t, DefaultShouldRetry(1, nil, nil, &netError{timeout: true}))
	assert.False(t, DefaultShouldRetry(1, nil, nil, &netError{}))
	assert.False(t, DefaultShouldRetry(1, nil, MockResponse(400), nil))
	assert.True(t, DefaultShouldRetry(1, nil, MockResponse(500), nil))
	assert.False(t, DefaultShouldRetry(1, nil, MockResponse(501), nil))
	assert.True(t, DefaultShouldRetry(1, nil, MockResponse(502), nil))
	assert.True(t, DefaultShouldRetry(1, nil, MockResponse(429), nil))
}

func TestOnlyIdempotentShouldRetry(t *testing.T) {
	tests := []struct {
		method   string
		expected bool
	}{
		{http.MethodGet, true},
		{http.MethodOptions, true},
		{http.MethodHead, true},
		{http.MethodTrace, true},
		{http.MethodPost, false},
		{http.MethodPut, false},
		{http.MethodPatch, false},
		{http.MethodDelete, false},
	}

	for _, test := range tests {
		t.Run(test.method, func(t *testing.T) {
			req, err := http.NewRequest(test.method, "http://test.com", nil)
			require.NoError(t, err)

			if test.expected {
				assert.True(t, OnlyIdempotentShouldRetry(1, req, nil, nil))
			} else {
				assert.False(t, OnlyIdempotentShouldRetry(1, req, nil, nil))
			}
		})
	}
}

func TestAllRetryers(t *testing.T) {
	r := AllRetryers(ShouldRetryerFunc(DefaultShouldRetry), ShouldRetryerFunc(OnlyIdempotentShouldRetry))

	// false + false = false
	req, err := http.NewRequest(http.MethodPost, "http://test.com", nil)
	require.NoError(t, err)
	assert.False(t, r.ShouldRetry(1, req, MockResponse(400), nil))

	// true + false = false
	assert.False(t, r.ShouldRetry(1, req, MockResponse(500), nil))

	// false + true = false
	req, err = http.NewRequest(http.MethodGet, "http://test.com", nil)
	require.NoError(t, err)
	assert.False(t, r.ShouldRetry(1, req, MockResponse(400), nil))

	// true + true = true
	assert.True(t, r.ShouldRetry(1, req, MockResponse(500), nil))

}

func TestRetry(t *testing.T) {
	// this test asserts that requests are retried the right number of times, and with the
	// correct time interval between retries.

	// to test this, create a server that always returns 500s, and a client with the an exponential
	// backoff retry.  It should try one request immediately, then retry 3 times, after 50ms, 100ms, and 200ms
	// respectively.

	// we inject an inspector into the server to watch the requests.  We spawn a goroutine to
	// call the client, then receive messages from the server inspector on a channel when a
	// request happens.

	// we record when we saw the requests, and confirm they happened roughly when expected.

	s := httptest.NewServer(MockHandler(500))
	defer s.Close()

	r := httptestutil.Requester(s, Retry(&RetryConfig{
		MaxAttempts: 4,
		Backoff: &ExponentialBackoff{
			BaseDelay:  50 * time.Millisecond,
			Multiplier: 2,
			Jitter:     0,
			MaxDelay:   time.Second,
		},
	}))

	i := httptestutil.Inspect(s)

	var resp *http.Response
	var err error
	t0 := time.Now()
	done := make(chan bool)
	go func() {
		// spawn a go routine to call the client.  this will block until all the retries
		// finish.
		resp, _, err = r.Receive(nil)
		// capture the response, and send a signal that the client finished.
		done <- true
	}()

	// total requests detected
	var count int
	// how long was the time between each request.
	var waits []time.Duration

loop:
	for {
		// on each loop, wait for the inspector to send a request on its channel.
		// break out of the loop if the requester goroutine reported that the client
		// call returned, or if we time out.
		select {
		case <-i.Exchanges:
			count++
			if count > 1 {
				// keep track of the waits between retries
				t1 := time.Now()
				waits = append(waits, t1.Sub(t0))
				t0 = t1
			}
		case <-time.After(time.Second):
			require.Fail(t, "timeout", "after %v requests", count)
		case <-done:
			break loop
		}
	}

	assert.NoError(t, err)
	if assert.NotNil(t, resp) {
		assert.Equal(t, 500, resp.StatusCode)
	}

	assert.Equal(t, 4, count)
	require.Len(t, waits, 3)
	t.Log(waits)
	assert.InDelta(t, 50*time.Millisecond, waits[0], float64(10*time.Millisecond))
	assert.InDelta(t, 100*time.Millisecond, waits[1], float64(10*time.Millisecond))
	assert.InDelta(t, 200*time.Millisecond, waits[2], float64(10*time.Millisecond))
}

func TestRetry_post(t *testing.T) {
	// POST requests can only be retried if http.Request.GetBody is no nil.  test cases where
	// it's set and the requests can be retried, and cases where it is nil and the request
	// can't be retried.

	s := httptest.NewServer(MockHandler(500))
	defer s.Close()

	r := httptestutil.Requester(s, Retry(&RetryConfig{
		MaxAttempts: 4,
		Backoff:     &ExponentialBackoff{BaseDelay: 0},
	}))

	i := httptestutil.Inspect(s)

	expectBody := true

	// consumes all pending requests in the inspector, asserts they have the right request and body,
	// and returns how many there were.
	count := func(t *testing.T) int {
		var count int

		for {
			e := i.NextExchange()
			if e == nil {
				break
			}

			count++
			assert.Equal(t, "POST", e.Request.Method)
			if expectBody {
				assert.Equal(t, "fudge", e.RequestBody.String())
			}
		}

		return count
	}

	// most body types will be automatically wrapped with an appropriate GetBody function, so they can
	// be correctly replayed.
	resp, _, err := r.Receive(Post(), Body("fudge"))

	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
	assert.Equal(t, 4, count(t))

	// This type of body can't be converted, so the request's GetBody function will be nil.
	// This will not be retried.
	resp, _, err = r.Receive(Post(), Body(&dummyReader{next: strings.NewReader("fudge")}))
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
	assert.Equal(t, 1, count(t))

	// http.NoBody is a special case.  It's a non-nil sentinel value indicating the request has
	// no body.  We should be able to retry this, even though GetBody will be nil.
	expectBody = false
	resp, _, err = r.Receive(Post(), Body(http.NoBody))
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)
	assert.Equal(t, 4, count(t))
}

type dummyReader struct {
	next io.Reader
}

func (d *dummyReader) Read(p []byte) (n int, err error) {
	return d.next.Read(p)
}

func TestRetry_respDrained(t *testing.T) {
	// when retrying a request, the response body of the last attempt must be
	// fully drained first, or there will be a leak.
	s := httptest.NewServer(MockHandler(500, Body("fudge")))
	defer s.Close()

	var resps []*http.Response

	r := httptestutil.Requester(s, Retry(&RetryConfig{
		MaxAttempts: 4,
		Backoff:     &ExponentialBackoff{BaseDelay: 0},
	}), Middleware(func(doer Doer) Doer {
		return DoerFunc(func(req *http.Request) (*http.Response, error) {
			resp, err := doer.Do(req)
			resps = append(resps, resp)
			return resp, err
		})
	}))

	_, body, err := r.Receive(nil)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(resps))
	assert.Equal(t, "fudge", string(body))

	// all the response bodies should have been drained
	for i, resp := range resps {
		t.Log("checking response", i)
		require.NotNil(t, resp)
		bytes := make([]byte, 39)
		_, err := resp.Body.Read(bytes)
		assert.EqualError(t, err, "http: read on closed response body")
	}
}

func TestRetry_cancelContext(t *testing.T) {
	// context cancellation can be used to abort retries
	s := httptest.NewServer(MockHandler(500, Body("fudge")))
	defer s.Close()

	r := httptestutil.Requester(s, Retry(&RetryConfig{
		MaxAttempts: 4,
		Backoff:     &ExponentialBackoff{BaseDelay: 2 * time.Second},
	}))

	ctx, cancelFunc := context.WithCancel(context.Background())

	var err error
	done := make(chan bool)
	go func() {
		_, _, err = r.ReceiveContext(ctx, nil)
		done <- true
	}()

	cancelFunc()

	select {
	case <-time.After(time.Second):
		require.Fail(t, "timed out")
	case <-done:
	}

	require.ErrorIs(t, err, context.Canceled)
}

func TestRetry_shouldRetry(t *testing.T) {
	// test a custom ShouldRetry function.  also test that Retry calls the ShouldRetry function
	// with the right args.
	var srvCount int
	s := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		srvCount++
		writer.WriteHeader(501 + srvCount)
		writer.Write([]byte("fudge"))
	}))
	defer s.Close()

	var count int
	var attempts []int
	var reqs []*http.Request
	var resps []*http.Response

	r := httptestutil.Requester(s, Retry(&RetryConfig{
		MaxAttempts: 4,
		Backoff:     &ExponentialBackoff{BaseDelay: 0},
		ShouldRetry: ShouldRetryerFunc(func(attempt int, req *http.Request, resp *http.Response, err error) bool {
			count++
			attempts = append(attempts, attempt)
			reqs = append(reqs, req)
			resps = append(resps, resp)
			return attempt != 3
		}),
	}))

	_, _, err := r.Receive(nil)
	require.NoError(t, err)

	// our should function should tell it stop after 3 attempts, not 4
	assert.Equal(t, 3, count)
	assert.Len(t, attempts, 3)
	for i := 0; i < 3; i++ {
		// attempts should be 1, 2, and 3
		assert.Equal(t, i+1, attempts[i])
		// reqs and resps should be non nil
		assert.NotNil(t, reqs[i])
		if assert.NotNil(t, resps[i]) {
			// each response should have a different code: 502, 503, and 504
			assert.Equal(t, 501+attempts[i], resps[i].StatusCode)
		}
	}
}

func TestRetry_success(t *testing.T) {
	// if request succeeds, no retries
	s := httptest.NewServer(MockHandler(200, Body("fudge")))
	defer s.Close()

	r := httptestutil.Requester(s, Retry(nil))

	i := httptestutil.Inspect(s)

	resp, body, err := r.Receive(nil)
	require.NoError(t, err)
	assert.Equal(t, "fudge", string(body))
	assert.Equal(t, 200, resp.StatusCode)

	// it should not have retried, since the first attempt was a success
	assert.Len(t, i.Drain(), 1)
}

// poisonedReader returns "fu" in the first call, and a connection
// reset error in the next call.
type poisonedReader struct {
	remaining int
}

func (r *poisonedReader) Read(p []byte) (n int, err error) {
	if r.remaining > 0 {
		n = copy(p, "fu"[r.remaining:])
		r.remaining -= n
		return n, nil
	}
	return 0, &net.OpError{
		Op:  "accept",
		Err: syscall.ECONNRESET,
	}
}

func TestRetry_readResponse(t *testing.T) {
	// optionally, Retry can retry the request if an error occurs in the middle
	// of reading the response body.  This is accomplished by having Retry
	// read the entire response body into memory in the middleware.  This is not
	// not the default: when downloading a file or a large response, it may not
	// be desirable to read the entire response into memory.
	// to test, use a mock Doer which simulates a connection reset error halfway
	// through reading the response body.
	var count int

	retryConfig := RetryConfig{
		MaxAttempts: 4,
		Backoff: &ExponentialBackoff{
			BaseDelay:  1,
			Multiplier: 1,
			Jitter:     0,
			MaxDelay:   time.Second,
		},
	}

	newRequester := func() *Requester {
		r, err := New(
			Retry(&retryConfig),
			WithDoer(DoerFunc(func(req *http.Request) (*http.Response, error) {
				count++
				// I can't cause a real connection reset error using httptest, so I need to simulate
				// it with a fake Doer.  https://groups.google.com/g/golang-nuts/c/AtxmEDJ4zvc
				if count > 2 {
					// on the third attempt, just return a valid response
					return MockResponse(200, Body("fudge")), nil
				}

				// return a response with a poisoned response body will will thrown an error after
				// a few bytes
				resp := MockResponse(200)
				resp.Body = io.NopCloser(&poisonedReader{})
				return resp, nil
			})),
		)
		require.NoError(t, err)
		return r
	}

	r := newRequester()

	// without setting flag, it should fail after the first attempt.
	// it will not be retried
	_, _, err := r.Receive(nil)
	assert.ErrorIs(t, err, syscall.ECONNRESET)
	assert.Equal(t, 1, count)

	// now try the flag
	count = 0
	retryConfig.ReadResponse = true
	r = newRequester()

	resp, body, err := r.Receive(nil)
	require.NoError(t, err)
	assert.Equal(t, "fudge", string(body))
	assert.Equal(t, 200, resp.StatusCode)

	// should have taken 3 tries
	assert.Equal(t, 3, count)

}
