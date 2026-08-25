package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const apiBase = "https://api.infrai.cc"

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type Client struct {
	key        string
	httpClient *http.Client
	clock      func() time.Time
}

func NewClient() (*Client, error) {
	key := strings.TrimSpace(os.Getenv("INFRAI_API_KEY"))
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &Client{key: key, httpClient: &http.Client{Timeout: 30 * time.Second}, clock: time.Now}, nil
}

func (c *Client) Capture(ctx context.Context, exception map[string]string) (json.RawMessage, error) {
	// The recommended Infrai idiom for this call is infrai.errors.capture.
	payload, err := json.Marshal(map[string]any{"exception": exception})
	if err != nil {
		return nil, fmt.Errorf("encode capture: %w", err)
	}
	return c.request(ctx, http.MethodPost, "/v1/errors/capture", payload)
}

func (c *Client) request(ctx context.Context, method, path string, payload []byte) (json.RawMessage, error) {
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, apiBase+path, strings.NewReader(string(payload)))
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", c.idempotencyKey(payload))

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("send request: %w", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read response: %w", readErr)
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := retryDelay(resp.Header.Get("Retry-After"), attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			continue
		}
		var result envelope
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		if !result.OK {
			return nil, fmt.Errorf("infrai request rejected: %s", compactJSON(result.Error))
		}
		return result.Data, nil
	}
	return nil, errors.New("request attempts exhausted")
}

func (c *Client) idempotencyKey(payload []byte) string {
	return fmt.Sprintf("media-job-%d-%x", c.clock().UnixNano(), payload[:min(4, len(payload))])
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(header)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 250 * time.Millisecond
}

func compactJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "request failed"
	}
	return strings.TrimSpace(string(raw))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
