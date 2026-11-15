package translate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPolicy returns a retry policy that records sleeps instead of sleeping.
func testPolicy(slept *[]time.Duration, now time.Time) retryPolicy {
	p := defaultRetryPolicy()
	p.sleep = func(_ context.Context, d time.Duration) error {
		*slept = append(*slept, d)
		return nil
	}
	p.now = func() time.Time { return now }
	return p
}

func getRequest(url string) func(ctx context.Context) (*http.Request, error) {
	return func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	}
}

// statusSequence serves the given statuses in order, repeating the last one.
func statusSequence(t *testing.T, header http.Header, statuses ...int) (*httptest.Server, *int32) {
	t.Helper()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		i := int(atomic.AddInt32(&calls, 1)) - 1
		if i >= len(statuses) {
			i = len(statuses) - 1
		}
		for k, v := range header {
			w.Header()[k] = v
		}
		w.WriteHeader(statuses[i])
		_, _ = w.Write([]byte("body"))
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func TestRetryPolicy_Success(t *testing.T) {
	srv, calls := statusSequence(t, nil, http.StatusOK)
	var slept []time.Duration

	body, err := testPolicy(&slept, time.Now()).do(context.Background(), srv.Client(), getRequest(srv.URL))

	require.NoError(t, err)
	assert.Equal(t, "body", string(body))
	assert.EqualValues(t, 1, *calls)
	assert.Empty(t, slept)
}

func TestRetryPolicy_BackoffThenSuccess(t *testing.T) {
	srv, calls := statusSequence(t, nil, http.StatusTooManyRequests, http.StatusServiceUnavailable, http.StatusOK)
	var slept []time.Duration

	_, err := testPolicy(&slept, time.Now()).do(context.Background(), srv.Client(), getRequest(srv.URL))

	require.NoError(t, err)
	assert.EqualValues(t, 3, *calls)
	assert.Equal(t, []time.Duration{time.Second, 2 * time.Second}, slept)
}

func TestRetryPolicy_RetriesExhausted(t *testing.T) {
	srv, calls := statusSequence(t, nil, http.StatusTooManyRequests)
	var slept []time.Duration

	_, err := testPolicy(&slept, time.Now()).do(context.Background(), srv.Client(), getRequest(srv.URL))

	assert.True(t, IsKind(err, ErrorKindRateLimit), "got %v", err)
	assert.EqualValues(t, maxRetries+1, *calls)
	assert.Equal(t, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}, slept)
}

func TestRetryPolicy_RetryAfterSeconds(t *testing.T) {
	srv, _ := statusSequence(t, http.Header{"Retry-After": {"7"}}, http.StatusTooManyRequests, http.StatusOK)
	var slept []time.Duration

	_, err := testPolicy(&slept, time.Now()).do(context.Background(), srv.Client(), getRequest(srv.URL))

	require.NoError(t, err)
	assert.Equal(t, []time.Duration{7 * time.Second}, slept)
}

func TestRetryPolicy_RetryAfterDateIsCapped(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour).Format(http.TimeFormat)
	srv, _ := statusSequence(t, http.Header{"Retry-After": {later}}, http.StatusServiceUnavailable, http.StatusOK)
	var slept []time.Duration

	_, err := testPolicy(&slept, now).do(context.Background(), srv.Client(), getRequest(srv.URL))

	require.NoError(t, err)
	assert.Equal(t, []time.Duration{maxRetryDelay}, slept)
}

func TestRetryPolicy_AuthErrorNotRetried(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		srv, calls := statusSequence(t, nil, status)
		var slept []time.Duration

		_, err := testPolicy(&slept, time.Now()).do(context.Background(), srv.Client(), getRequest(srv.URL))

		assert.True(t, IsKind(err, ErrorKindAuth), "status %d: got %v", status, err)
		assert.EqualValues(t, 1, *calls)
		assert.Empty(t, slept)
	}
}

func TestRetryPolicy_OtherStatusNotRetried(t *testing.T) {
	srv, calls := statusSequence(t, nil, http.StatusInternalServerError)
	var slept []time.Duration

	_, err := testPolicy(&slept, time.Now()).do(context.Background(), srv.Client(), getRequest(srv.URL))

	assert.True(t, IsKind(err, ErrorKindResponse), "got %v", err)
	assert.EqualValues(t, 1, *calls)
}

func TestRetryPolicy_ConnectionError(t *testing.T) {
	srv, _ := statusSequence(t, nil, http.StatusOK)
	url := srv.URL
	srv.Close()
	var slept []time.Duration

	_, err := testPolicy(&slept, time.Now()).do(context.Background(), http.DefaultClient, getRequest(url))

	assert.True(t, IsKind(err, ErrorKindConnection), "got %v", err)
	assert.Empty(t, slept)
}

func TestRetryPolicy_CancelledDuringBackoff(t *testing.T) {
	srv, _ := statusSequence(t, nil, http.StatusTooManyRequests)
	ctx, cancel := context.WithCancel(context.Background())
	p := defaultRetryPolicy()
	p.sleep = func(ctx context.Context, _ time.Duration) error {
		cancel()
		return ctx.Err()
	}

	_, err := p.do(ctx, srv.Client(), getRequest(srv.URL))

	assert.ErrorIs(t, err, context.Canceled)
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		in     string
		want   time.Duration
		wantOK bool
	}{
		{"", 0, false},
		{"3", 3 * time.Second, true},
		{"-1", 0, false},
		{"soon", 0, false},
		{now.Add(30 * time.Second).Format(http.TimeFormat), 30 * time.Second, true},
		{now.Add(-time.Minute).Format(http.TimeFormat), 0, true},
	}

	for _, tt := range tests {
		got, ok := parseRetryAfter(tt.in, now)
		assert.Equal(t, tt.wantOK, ok, tt.in)
		assert.Equal(t, tt.want, got, tt.in)
	}
}
