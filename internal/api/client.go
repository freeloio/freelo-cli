package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/freeloio/freelo-cli/internal/auth"
	"github.com/freeloio/freelo-cli/internal/config"
)

// Client handles all HTTP communication with the Freelo API.
type Client struct {
	baseURL    string
	auth       auth.Provider
	httpClient *http.Client

	// Simple rate limiter: track last request time
	mu          sync.Mutex
	lastRequest time.Time
}

// NewClient creates an API client from config and auth provider.
func NewClient(cfg *config.Config, authProvider auth.Provider) *Client {
	return &Client{
		baseURL: cfg.BaseURL,
		auth:    authProvider,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Get performs a GET request to the given API path.
func (c *Client) Get(path string) (json.RawMessage, error) {
	return c.do("GET", path, nil)
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(path string, body any) (json.RawMessage, error) {
	return c.do("POST", path, body)
}

// Put performs a PUT request with a JSON body.
func (c *Client) Put(path string, body any) (json.RawMessage, error) {
	return c.do("PUT", path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(path string) (json.RawMessage, error) {
	return c.do("DELETE", path, nil)
}

func (c *Client) do(method, path string, body any) (json.RawMessage, error) {
	c.rateLimit()

	email, secret, err := c.auth.GetCredentials()
	if err != nil {
		return nil, err
	}

	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = strings.NewReader(string(data))
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(email, secret)
	req.Header.Set("User-Agent", "FreeloCLI")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// SECURITY: Limit response body to 10MB to prevent OOM from malicious servers
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle rate limiting
	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited (25 req/min) — wait 60 seconds and retry")
	}

	if resp.StatusCode >= 400 {
		// SECURITY: Truncate error body to prevent credential leakage
		// (a rogue server could echo back auth headers in the response)
		errBody := string(respBody)
		if len(errBody) > 500 {
			errBody = errBody[:500] + "... (truncated)"
		}
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, errBody)
	}

	// Some endpoints return empty body on success (DELETE, etc.)
	if len(respBody) == 0 {
		return json.RawMessage("null"), nil
	}

	return json.RawMessage(respBody), nil
}

// rateLimit enforces a minimum interval between requests to stay under 25/min.
func (c *Client) rateLimit() {
	c.mu.Lock()
	defer c.mu.Unlock()

	minInterval := time.Second * 60 / 25 // ~2.4s between requests
	elapsed := time.Since(c.lastRequest)
	if elapsed < minInterval {
		time.Sleep(minInterval - elapsed)
	}
	c.lastRequest = time.Now()
}
