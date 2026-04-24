// Package api wires the oapi-codegen-generated Freelo client with the
// cross-cutting concerns the CLI needs: Basic Auth, User-Agent, rate
// limiting, and retry-with-backoff on transient failures.
//
// The generated client lives in internal/api/freelo. This file is the
// production seam — it stays small and testable so Phase 3's command
// migration doesn't re-implement any of it per command.
package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"

	"github.com/freeloio/freelo-cli/internal/api/freelo"
	"github.com/freeloio/freelo-cli/internal/auth"
	"github.com/freeloio/freelo-cli/internal/config"
)

const (
	// Freelo publishes a 25 req/min rate limit. Spacing requests by ~2.4s
	// keeps us safely under it. Matches the handwritten client's behavior
	// so we don't regress during the migration.
	minRequestInterval = 60 * time.Second / 25

	// Maximum number of attempts per request (initial + 2 retries).
	maxAttempts = 3

	// Bounds for exponential backoff with full jitter.
	backoffBase = 500 * time.Millisecond
	backoffMax  = 8 * time.Second

	// Cap response bodies we read from the server to guard against memory
	// exhaustion from a rogue/broken upstream. 10 MB is generous for an
	// API that returns JSON.
	maxResponseBody = 10 * 1024 * 1024
)

// NewFreeloClient builds a typed Freelo client ready to call against the live
// API. The caller gets back the `ClientWithResponses` surface from
// oapi-codegen, which yields typed response structs (JSON200, JSON400,
// JSON429, ...) for every endpoint.
func NewFreeloClient(cfg *config.Config, authProvider auth.Provider, userAgent string) (*freelo.ClientWithResponses, error) {
	doer := newRetryingDoer(&http.Client{Timeout: 30 * time.Second})

	return freelo.NewClientWithResponses(
		cfg.BaseURL,
		freelo.WithHTTPClient(doer),
		freelo.WithRequestEditorFn(basicAuthEditor(authProvider)),
		freelo.WithRequestEditorFn(headerEditor(userAgent)),
	)
}

// basicAuthEditor injects the user's email + API key on every request.
// It resolves credentials per-request so that a CLI session that runs
// `freelo auth login` mid-flight picks up the new creds next call.
func basicAuthEditor(p auth.Provider) freelo.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		email, key, err := p.GetCredentials()
		if err != nil {
			return err
		}
		req.SetBasicAuth(email, key)
		return nil
	}
}

// headerEditor sets the headers Freelo expects on every request. Freelo
// rejects requests without a User-Agent; all Freelo-authored clients set
// one for telemetry, matching the pattern established by the claude skill
// (Freelo-Claude-Skill/1.0.0).
func headerEditor(userAgent string) freelo.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		req.Header.Set("User-Agent", userAgent)
		if req.Header.Get("Accept") == "" {
			req.Header.Set("Accept", "application/json")
		}
		return nil
	}
}

// retryingDoer is an HttpRequestDoer that enforces a client-side rate limit,
// retries transient failures with exponential backoff + full jitter, and
// honors Retry-After headers on 429 responses.
type retryingDoer struct {
	inner    *http.Client
	interval time.Duration // min spacing between requests; tests override to 0

	mu          sync.Mutex
	lastRequest time.Time
}

func newRetryingDoer(inner *http.Client) *retryingDoer {
	return &retryingDoer{inner: inner, interval: minRequestInterval}
}

// Do is the HttpRequestDoer contract from oapi-codegen.
func (d *retryingDoer) Do(req *http.Request) (*http.Response, error) {
	// The request body is consumed after the first attempt; capture it
	// so retries can replay the same payload.
	var body []byte
	if req.Body != nil && req.GetBody == nil {
		b, err := io.ReadAll(io.LimitReader(req.Body, maxResponseBody))
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
		_ = req.Body.Close()
		body = b
		req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
		req.Body, _ = req.GetBody()
	}

	var lastResp *http.Response
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			// Rewind request body before retrying.
			if req.GetBody != nil {
				rc, err := req.GetBody()
				if err != nil {
					return nil, fmt.Errorf("rewind body: %w", err)
				}
				req.Body = rc
			}
		}

		d.waitForRateLimit()

		resp, err := d.inner.Do(req)
		lastResp, lastErr = resp, err

		// Network failure → retry with backoff.
		if err != nil {
			if !isRetryableNetErr(err) {
				return nil, err
			}
			sleepWithBackoff(attempt, req.Context())
			continue
		}

		// HTTP error we should retry on (429 or 5xx).
		if shouldRetryStatus(resp.StatusCode) && attempt < maxAttempts-1 {
			delay := parseRetryAfter(resp.Header.Get("Retry-After"))
			_ = drainAndClose(resp.Body)
			if delay > 0 {
				sleepCtx(req.Context(), delay)
			} else {
				sleepWithBackoff(attempt, req.Context())
			}
			continue
		}

		return resp, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all %d attempts failed: %w", maxAttempts, lastErr)
	}
	return lastResp, nil
}

// waitForRateLimit blocks until enough time has elapsed since the last request.
func (d *retryingDoer) waitForRateLimit() {
	d.mu.Lock()
	defer d.mu.Unlock()
	elapsed := time.Since(d.lastRequest)
	if elapsed < d.interval {
		time.Sleep(d.interval - elapsed)
	}
	d.lastRequest = time.Now()
}

func shouldRetryStatus(code int) bool {
	return code == http.StatusTooManyRequests || (code >= 500 && code <= 599)
}

// isRetryableNetErr tells us whether a non-HTTP error (dns, tcp, tls handshake)
// looks transient. Today we retry any error the http.Client returns — all of
// them are IO-level and not semantic (we already got past the URL parsing).
func isRetryableNetErr(err error) bool {
	return err != nil && err != context.Canceled && err != context.DeadlineExceeded
}

// parseRetryAfter understands both delta-seconds (most common) and HTTP-date
// forms of the Retry-After header. Returns 0 if unparseable.
func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if secs, err := time.ParseDuration(v + "s"); err == nil && secs > 0 {
		return secs
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}

// sleepWithBackoff sleeps for backoffBase * 2^attempt + jitter, capped at
// backoffMax. "Full jitter" per the AWS architecture blog: pick a uniform
// random value in [0, base * 2^attempt].
func sleepWithBackoff(attempt int, ctx context.Context) {
	exp := backoffBase << attempt
	if exp > backoffMax {
		exp = backoffMax
	}
	delay := time.Duration(rand.Int64N(int64(exp)))
	sleepCtx(ctx, delay)
}

func sleepCtx(ctx context.Context, d time.Duration) {
	if d <= 0 {
		return
	}
	if ctx == nil {
		time.Sleep(d)
		return
	}
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}

func drainAndClose(body io.ReadCloser) error {
	if body == nil {
		return nil
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(body, maxResponseBody))
	return body.Close()
}
