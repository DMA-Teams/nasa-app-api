package sources

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

type scriptedTransport struct {
	calls     atomic.Int32
	responses []func(*http.Request) *http.Response
}

func (s *scriptedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	i := int(s.calls.Add(1)) - 1
	if i >= len(s.responses) {
		i = len(s.responses) - 1
	}

	return s.responses[i](r), nil
}

func respond(code int, body string, header map[string]string) func(*http.Request) *http.Response {
	return func(r *http.Request) *http.Response {
		h := http.Header{}
		for k, v := range header {
			h.Set(k, v)
		}

		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: h, Request: r}
	}
}

func noJitter(d time.Duration) time.Duration { return d }

func TestClient_RetriesTransientErrorsWithBackoff(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		tr := &scriptedTransport{responses: []func(*http.Request) *http.Response{
			respond(http.StatusServiceUnavailable, "", nil),
			respond(http.StatusTooManyRequests, "", map[string]string{"Retry-After": "7"}),
			respond(http.StatusOK, `{"ok":true}`, nil),
		}}
		c := NewClient(time.Second, WithTransport(tr), WithJitter(noJitter), WithRetries(4, time.Second, time.Minute))

		start := time.Now()
		body, err := c.Get(context.Background(), "https://example.test/data")

		if err != nil || string(body) != `{"ok":true}` {
			t.Fatalf("Get = %q, %v", body, err)
		}

		if elapsed := time.Since(start); elapsed != 8*time.Second {
			t.Errorf("waited %v, want 1s backoff + 7s Retry-After", elapsed)
		}

		if tr.calls.Load() != 3 {
			t.Errorf("calls = %d, want 3", tr.calls.Load())
		}
	})
}

func TestClient_DoesNotRetryClientErrors(t *testing.T) {
	t.Parallel()

	tr := &scriptedTransport{responses: []func(*http.Request) *http.Response{respond(http.StatusNotFound, "", nil)}}
	c := NewClient(time.Second, WithTransport(tr), WithJitter(noJitter))

	_, err := c.Get(context.Background(), "https://example.test/missing")

	var status *StatusError
	if !errors.As(err, &status) || status.Code != http.StatusNotFound || tr.calls.Load() != 1 {
		t.Errorf("err = %v, calls = %d; want one 404 attempt", err, tr.calls.Load())
	}
}

func TestClient_StopsRetryingWhenContextIsCanceled(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		tr := &scriptedTransport{responses: []func(*http.Request) *http.Response{respond(http.StatusBadGateway, "", nil)}}
		c := NewClient(time.Second, WithTransport(tr), WithJitter(noJitter), WithRetries(10, time.Minute, time.Hour))

		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		_, err := c.Get(ctx, "https://example.test/slow")
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("err = %v, want deadline exceeded", err)
		}

		if tr.calls.Load() != 2 {
			t.Errorf("calls = %d, want 2 before the deadline", tr.calls.Load())
		}
	})
}

func TestClient_RejectsOversizedBodies(t *testing.T) {
	t.Parallel()

	tr := &scriptedTransport{responses: []func(*http.Request) *http.Response{respond(http.StatusOK, strings.Repeat("x", 100), nil)}}
	c := NewClient(time.Second, WithTransport(tr), WithMaxBody(10))

	if _, err := c.Get(context.Background(), "https://example.test/big"); !errors.Is(err, ErrResponseTooLarge) {
		t.Errorf("err = %v, want ErrResponseTooLarge", err)
	}
}

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()

	if got := parseRetryAfter("12"); got != 12*time.Second {
		t.Errorf("seconds = %v, want 12s", got)
	}

	if got := parseRetryAfter("soon"); got != 0 {
		t.Errorf("invalid = %v, want 0", got)
	}
}

type timeoutThenOK struct {
	calls atomic.Int32
}

func (t *timeoutThenOK) RoundTrip(r *http.Request) (*http.Response, error) {
	if t.calls.Add(1) == 1 {
		return nil, context.DeadlineExceeded
	}

	return respond(http.StatusOK, "ok", nil)(r), nil
}

func TestClient_RetriesPerAttemptTimeouts(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		tr := &timeoutThenOK{}
		c := NewClient(time.Second, WithTransport(tr), WithJitter(noJitter), WithRetries(3, time.Second, time.Minute))

		body, err := c.Get(context.Background(), "https://example.test/slow-once")
		if err != nil || string(body) != "ok" || tr.calls.Load() != 2 {
			t.Errorf("Get = %q, %v after %d calls; want ok after a retried timeout", body, err, tr.calls.Load())
		}
	})
}

func TestClient_SendsBearerTokenOnlyWhenConfigured(t *testing.T) {
	for _, tc := range []struct {
		name  string
		token string
		want  string
	}{
		{"with token", "secret", "Bearer secret"},
		{"without token", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			rt := &scriptedTransport{responses: []func(*http.Request) *http.Response{func(r *http.Request) *http.Response {
				got = r.Header.Get("Authorization")

				return respond(http.StatusOK, "ok", nil)(r)
			}}}

			c := NewClient(time.Second, WithTransport(rt), WithBearer(tc.token))
			if _, err := c.Get(t.Context(), "https://example.test/x"); err != nil {
				t.Fatal(err)
			}

			if got != tc.want {
				t.Errorf("Authorization = %q, want %q", got, tc.want)
			}
		})
	}
}
