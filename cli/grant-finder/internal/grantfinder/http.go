package grantfinder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const userAgent = "grant-finder/0.1"

type AdaptiveLimiter struct{}

func (AdaptiveLimiter) Wait(context.Context) error { return nil }
func (AdaptiveLimiter) OnSuccess()                 {}
func (AdaptiveLimiter) OnRateLimit()               {}

type RateLimitError struct {
	URL string `json:"url"`
}

func (e *RateLimitError) Error() string {
	return "rate limited: " + e.URL
}

func httpClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

func getBytes(ctx context.Context, url string, timeout time.Duration) ([]byte, int, string, error) {
	return getBytesLimit(ctx, url, timeout, 2<<20)
}

func getBytesLimit(ctx context.Context, url string, timeout time.Duration, limit int64) ([]byte, int, string, error) {
	limiter := AdaptiveLimiter{}
	if err := limiter.Wait(ctx); err != nil {
		return nil, 0, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/json, text/xml, text/html;q=0.8, */*;q=0.5")
	resp, err := httpClient(timeout).Do(req)
	if err != nil {
		return nil, 0, "", err
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if readErr != nil {
		return nil, resp.StatusCode, resp.Header.Get("Content-Type"), readErr
	}
	if int64(len(body)) > limit {
		return body[:limit], resp.StatusCode, resp.Header.Get("Content-Type"), fmt.Errorf("GET %s: response exceeded %d byte limit", url, limit)
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == 429 {
		limiter.OnRateLimit()
		return body, resp.StatusCode, resp.Header.Get("Content-Type"), &RateLimitError{URL: url}
	}
	limiter.OnSuccess()
	if resp.StatusCode >= 400 {
		return body, resp.StatusCode, resp.Header.Get("Content-Type"), fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}
	return body, resp.StatusCode, resp.Header.Get("Content-Type"), nil
}

func postJSON(ctx context.Context, url string, payload any, timeout time.Duration) (map[string]any, error) {
	limiter := AdaptiveLimiter{}
	if err := limiter.Wait(ctx); err != nil {
		return nil, err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient(timeout).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == 429 {
		limiter.OnRateLimit()
		return nil, &RateLimitError{URL: url}
	}
	limiter.OnSuccess()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("POST %s: HTTP %d: %s", url, resp.StatusCode, string(body))
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func FetchJSON(ctx context.Context, url string, timeout time.Duration) (map[string]any, error) {
	body, _, _, err := getBytes(ctx, url, timeout)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}
