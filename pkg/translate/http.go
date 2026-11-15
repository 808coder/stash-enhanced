package translate

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/stashapp/stash/pkg/logger"
)

const (
	maxRetries     = 3
	baseRetryDelay = time.Second
	// maxRetryDelay caps both the exponential backoff and a server's
	// Retry-After, so a misbehaving server cannot stall a job for hours.
	maxRetryDelay = 2 * time.Minute

	maxResponseSize = 10 << 20
	maxErrorSnippet = 200
)

// retryPolicy sends HTTP requests, retrying on HTTP 429 and 503 with
// exponential backoff and honouring Retry-After.
type retryPolicy struct {
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
	// sleep waits for d or until ctx is done. Replaced in tests.
	sleep func(ctx context.Context, d time.Duration) error
	// now is used to evaluate Retry-After dates. Replaced in tests.
	now func() time.Time
}

func defaultRetryPolicy() retryPolicy {
	return retryPolicy{
		maxRetries: maxRetries,
		baseDelay:  baseRetryDelay,
		maxDelay:   maxRetryDelay,
		sleep:      sleepContext,
		now:        time.Now,
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// do sends the request built by newReq and returns the body of a 2xx
// response. newReq is called once per attempt, because a request body can
// only be read once.
func (p retryPolicy) do(ctx context.Context, client *http.Client, newReq func(ctx context.Context) (*http.Request, error)) ([]byte, error) {
	for attempt := 0; ; attempt++ {
		req, err := newReq(ctx)
		if err != nil {
			return nil, newError(ErrorKindResponse, "building request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, &Error{Kind: ErrorKindConnection, Err: err}
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
		resp.Body.Close()
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, &Error{Kind: ErrorKindConnection, Err: fmt.Errorf("reading response: %w", err)}
		}

		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			return body, nil
		case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
			return nil, newError(ErrorKindAuth, "HTTP %d: %s", resp.StatusCode, snippet(body))
		case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable:
			if attempt >= p.maxRetries {
				return nil, newError(ErrorKindRateLimit, "HTTP %d after %d retries: %s", resp.StatusCode, attempt, snippet(body))
			}
			d := p.delay(attempt, resp.Header.Get("Retry-After"))
			logger.Tracef("Translation provider answered HTTP %d, retry %d of %d in %s", resp.StatusCode, attempt+1, p.maxRetries, d)
			if err := p.sleep(ctx, d); err != nil {
				return nil, err
			}
		default:
			return nil, newError(ErrorKindResponse, "HTTP %d: %s", resp.StatusCode, snippet(body))
		}
	}
}

// delay returns how long to wait before retry number attempt+1: the server's
// Retry-After if present and valid, otherwise baseDelay * 2^attempt. Both are
// capped at maxDelay.
func (p retryPolicy) delay(attempt int, retryAfter string) time.Duration {
	d, ok := parseRetryAfter(retryAfter, p.now())
	if !ok {
		d = p.baseDelay << attempt
	}

	if d > p.maxDelay {
		d = p.maxDelay
	}
	return d
}

// parseRetryAfter parses a Retry-After header, which is either a number of
// seconds or an HTTP date.
func parseRetryAfter(v string, now time.Time) (time.Duration, bool) {
	if v == "" {
		return 0, false
	}

	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			return 0, false
		}
		return time.Duration(secs) * time.Second, true
	}

	if t, err := http.ParseTime(v); err == nil {
		d := t.Sub(now)
		if d < 0 {
			d = 0
		}
		return d, true
	}

	return 0, false
}

func snippet(body []byte) string {
	if len(body) > maxErrorSnippet {
		return string(body[:maxErrorSnippet]) + "…"
	}
	return string(body)
}
