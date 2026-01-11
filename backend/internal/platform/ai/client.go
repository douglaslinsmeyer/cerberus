package ai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client is the Claude API client
type Client struct {
	apiKey         string
	baseURL        string
	httpClient     *http.Client
	cache          *redis.Client
	costCalculator *CostCalculator
	metricsTracker MetricsTracker
}

// MetricsTracker defines interface for tracking AI usage metrics
type MetricsTracker interface {
	Track(ctx context.Context, metrics *Metrics) error
}

// ClientConfig contains configuration for the Claude API client
type ClientConfig struct {
	APIKey         string
	RedisClient    *redis.Client
	MetricsTracker MetricsTracker
}

// NewClient creates a new Claude API client
func NewClient(config *ClientConfig) *Client {
	return &Client{
		apiKey:  config.APIKey,
		baseURL: "https://api.anthropic.com/v1",
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		cache:          config.RedisClient,
		costCalculator: NewCostCalculator(),
		metricsTracker: config.MetricsTracker,
	}
}

// Request sends a request to Claude API with retry logic
func (c *Client) Request(ctx context.Context, req *Request) (*Response, error) {
	// Check cache first (if not streaming)
	if !req.Stream && c.cache != nil {
		cacheKey := c.generateCacheKey(req)
		if cached, err := c.getFromCache(ctx, cacheKey); err == nil && cached != nil {
			return cached, nil
		}
	}

	// Make request with retry logic
	var resp *Response
	var lastErr error

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, lastErr = c.makeRequest(ctx, req)
		if lastErr == nil {
			break
		}

		// Don't retry on non-transient errors
		if !c.isRetryableError(lastErr) {
			return nil, lastErr
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed after 3 attempts: %w", lastErr)
	}

	// Cache response (1 hour TTL)
	if !req.Stream && c.cache != nil {
		cacheKey := c.generateCacheKey(req)
		c.cacheResponse(ctx, cacheKey, resp)
	}

	// Track metrics
	if c.metricsTracker != nil {
		cost := c.costCalculator.CalculateCost(req.Model, &resp.Usage)
		metrics := &Metrics{
			Model:        req.Model,
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
			CachedTokens: resp.Usage.CacheReadInputTokens,
			TotalTokens:  resp.Usage.InputTokens + resp.Usage.OutputTokens,
			Cost:         cost,
			CacheHit:     false,
			Timestamp:    time.Now(),
		}
		c.metricsTracker.Track(ctx, metrics)
	}

	return resp, nil
}

// makeRequest performs the actual HTTP request to Claude API
func (c *Client) makeRequest(ctx context.Context, req *Request) (*Response, error) {
	// Marshal request body
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if httpResp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			return nil, fmt.Errorf("API error (%d): %s", httpResp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("API error (%d): %s", httpResp.StatusCode, string(respBody))
	}

	// Parse response
	var apiResp Response
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &apiResp, nil
}

// generateCacheKey generates a cache key for a request
func (c *Client) generateCacheKey(req *Request) string {
	// Hash the request to generate a unique key
	h := sha256.New()
	json.NewEncoder(h).Encode(req)
	return fmt.Sprintf("ai:response:%x", h.Sum(nil))
}

// getFromCache retrieves a cached response
func (c *Client) getFromCache(ctx context.Context, key string) (*Response, error) {
	data, err := c.cache.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var resp Response
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// cacheResponse stores a response in cache
func (c *Client) cacheResponse(ctx context.Context, key string, resp *Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	// Cache for 1 hour
	return c.cache.Set(ctx, key, data, time.Hour).Err()
}

// isRetryableError determines if an error should trigger a retry
func (c *Client) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// Retry on rate limits, timeouts, and 5xx errors
	retryablePatterns := []string{
		"rate_limit",
		"timeout",
		"500",
		"502",
		"503",
		"504",
		"connection refused",
		"connection reset",
	}

	for _, pattern := range retryablePatterns {
		if contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
		containsIgnoreCase(s, substr))
}

func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// RequestWithContext creates a request with program context for prompt caching
func (c *Client) RequestWithContext(ctx context.Context, model string, systemPrompt string, staticContext string, dynamicContent string, maxTokens int) (*Response, error) {
	req := &Request{
		Model:       model,
		MaxTokens:   maxTokens,
		System:      systemPrompt,
		Temperature: 0.0, // Deterministic for extraction
		Messages: []Message{
			{
				Role: "user",
				Content: []ContentBlock{
					{
						Type: "text",
						Text: staticContext,
						CacheControl: &CacheControl{
							Type: "ephemeral",
						},
					},
					{
						Type: "text",
						Text: dynamicContent,
					},
				},
			},
		},
	}

	return c.Request(ctx, req)
}

// SimpleRequest creates a simple request without caching
func (c *Client) SimpleRequest(ctx context.Context, model string, systemPrompt string, userPrompt string, maxTokens int) (*Response, error) {
	req := &Request{
		Model:       model,
		MaxTokens:   maxTokens,
		System:      systemPrompt,
		Temperature: 0.0,
		Messages: []Message{
			{
				Role: "user",
				Content: []ContentBlock{
					{
						Type: "text",
						Text: userPrompt,
					},
				},
			},
		},
	}

	return c.Request(ctx, req)
}
