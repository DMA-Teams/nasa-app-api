package sources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

const (
	defaultMaxAttempts = 6
	defaultBaseDelay   = time.Second
	defaultMaxDelay    = time.Minute
	defaultMaxBody     = 16 << 20
	userAgent          = "nasa-app-api-etl/1.0"
)

var ErrResponseTooLarge = errors.New("response body exceeds the configured limit")

type StatusError struct {
	Code int
	URL  string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("GET %s: unexpected status %d", e.URL, e.Code)
}

type Client struct {
	http        *http.Client
	limiter     *rate.Limiter
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
	maxBody     int64
	jitter      func(time.Duration) time.Duration
	bearer      string
}

type Option func(*Client)

func WithRate(limit rate.Limit, burst int) Option {
	return func(c *Client) { c.limiter = rate.NewLimiter(limit, burst) }
}

func WithRetries(attempts int, base, maxDelay time.Duration) Option {
	return func(c *Client) { c.maxAttempts, c.baseDelay, c.maxDelay = attempts, base, maxDelay }
}

func WithMaxBody(n int64) Option {
	return func(c *Client) { c.maxBody = n }
}

func WithTransport(rt http.RoundTripper) Option {
	return func(c *Client) { c.http.Transport = rt }
}

func WithJitter(f func(time.Duration) time.Duration) Option {
	return func(c *Client) { c.jitter = f }
}

func WithBearer(token string) Option {
	return func(c *Client) { c.bearer = token }
}

func NewClient(timeout time.Duration, opts ...Option) *Client {
	c := &Client{
		http: &http.Client{Timeout: timeout, Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: timeout,
			MaxIdleConnsPerHost:   4,
			IdleConnTimeout:       90 * time.Second,
		}},
		maxAttempts: defaultMaxAttempts,
		baseDelay:   defaultBaseDelay,
		maxDelay:    defaultMaxDelay,
		maxBody:     defaultMaxBody,
		jitter:      fullJitter,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func fullJitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}

	return time.Duration(rand.Int64N(int64(d)))
}

func (c *Client) GetJSON(ctx context.Context, url string, v any) error {
	body, err := c.Get(ctx, url)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", url, err)
	}

	return nil
}

func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error

	for attempt := range c.maxAttempts {
		if c.limiter != nil {
			if err := c.limiter.Wait(ctx); err != nil {
				return nil, fmt.Errorf("rate limiter: %w", err)
			}
		}

		body, retryAfter, err := c.once(ctx, url)
		if err == nil {
			return body, nil
		}

		if ctx.Err() != nil {
			return nil, fmt.Errorf("GET %s: %w", url, ctx.Err())
		}

		lastErr = err
		if !retryable(err) || attempt == c.maxAttempts-1 {
			break
		}

		if err := sleep(ctx, c.delay(attempt, retryAfter)); err != nil {
			return nil, err
		}
	}

	return nil, lastErr
}

func (c *Client) once(ctx context.Context, url string) ([]byte, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	if c.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearer)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("GET %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))

		return nil, parseRetryAfter(resp.Header.Get("Retry-After")), &StatusError{Code: resp.StatusCode, URL: url}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBody+1))
	if err != nil {
		return nil, 0, fmt.Errorf("read %s: %w", url, err)
	}

	if int64(len(body)) > c.maxBody {
		return nil, 0, fmt.Errorf("GET %s: %w", url, ErrResponseTooLarge)
	}

	return body, 0, nil
}

func retryable(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, ErrResponseTooLarge) {
		return false
	}

	if status, ok := errors.AsType[*StatusError](err); ok {
		return status.Code == http.StatusTooManyRequests || status.Code >= http.StatusInternalServerError
	}

	return true
}

func (c *Client) delay(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return min(retryAfter, c.maxDelay)
	}

	backoff := c.baseDelay << attempt
	if backoff <= 0 || backoff > c.maxDelay {
		backoff = c.maxDelay
	}

	return c.jitter(backoff)
}

func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}

	if seconds, err := strconv.Atoi(v); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	if t, err := http.ParseTime(v); err == nil {
		return max(time.Until(t), 0)
	}

	return 0
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("retry wait: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}
