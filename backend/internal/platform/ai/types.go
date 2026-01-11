package ai

import "time"

// Model constants for Claude API
const (
	ModelOpus4   = "claude-opus-4-5-20251101"
	ModelSonnet4 = "claude-sonnet-4-5-20250929"
)

// Request represents a Claude API request
type Request struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	System      string          `json:"system,omitempty"`
	Messages    []Message       `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

// Message represents a message in the conversation
type Message struct {
	Role    string         `json:"role"` // "user" or "assistant"
	Content []ContentBlock `json:"content"`
}

// ContentBlock represents a content block in a message
type ContentBlock struct {
	Type         string        `json:"type"` // "text" or "image"
	Text         string        `json:"text,omitempty"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// CacheControl marks content for prompt caching
type CacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

// Response represents a Claude API response
type Response struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Role       string    `json:"role"`
	Content    []Content `json:"content"`
	Model      string    `json:"model"`
	StopReason string    `json:"stop_reason"`
	Usage      Usage     `json:"usage"`
}

// Content represents response content
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Usage represents token usage in a response
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Metrics tracks AI usage metrics
type Metrics struct {
	ProgramID    string
	Module       string
	JobType      string
	Model        string
	InputTokens  int
	OutputTokens int
	CachedTokens int
	TotalTokens  int
	Cost         float64
	Duration     time.Duration
	CacheHit     bool
	Timestamp    time.Time
}

// CostCalculator calculates API costs
type CostCalculator struct{}

// NewCostCalculator creates a new cost calculator
func NewCostCalculator() *CostCalculator {
	return &CostCalculator{}
}

// CalculateCost calculates the cost of an API request
func (c *CostCalculator) CalculateCost(model string, usage *Usage) float64 {
	var inputCost, outputCost float64

	switch model {
	case ModelOpus4:
		inputCost = 15.0 / 1_000_000  // $15 per MTok
		outputCost = 75.0 / 1_000_000 // $75 per MTok
	case ModelSonnet4:
		inputCost = 3.0 / 1_000_000   // $3 per MTok
		outputCost = 15.0 / 1_000_000 // $15 per MTok
	default:
		// Default to Sonnet pricing
		inputCost = 3.0 / 1_000_000
		outputCost = 15.0 / 1_000_000
	}

	// Cached tokens cost 10% of normal input cost
	cachedCost := float64(usage.CacheReadInputTokens) * inputCost * 0.1

	// Normal input tokens (excluding cached)
	normalInputTokens := usage.InputTokens - usage.CacheReadInputTokens
	normalInputCost := float64(normalInputTokens) * inputCost

	// Output tokens
	outputCostTotal := float64(usage.OutputTokens) * outputCost

	return cachedCost + normalInputCost + outputCostTotal
}

// GetExtractedText returns the text content from a response
func (r *Response) GetExtractedText() string {
	if len(r.Content) == 0 {
		return ""
	}
	return r.Content[0].Text
}
