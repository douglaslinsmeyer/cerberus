package extractors

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ImageOCRExtractor uses Claude Vision API for OCR on scanned PDFs and images
type ImageOCRExtractor struct {
	apiKey     string
	httpClient *http.Client
}

// NewImageOCRExtractor creates a new image OCR extractor
func NewImageOCRExtractor(apiKey string) *ImageOCRExtractor {
	return &ImageOCRExtractor{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// CanExtract returns true for image types and PDFs (as fallback)
func (e *ImageOCRExtractor) CanExtract(mimeType string) bool {
	// This is a fallback extractor - will be tried after PDF extractor fails
	return mimeType == "application/pdf" ||
		mimeType == "image/png" ||
		mimeType == "image/jpeg" ||
		mimeType == "image/jpg" ||
		mimeType == "image/gif" ||
		mimeType == "image/webp"
}

// Extract performs OCR using Claude Vision API
func (e *ImageOCRExtractor) Extract(ctx context.Context, data []byte) (string, error) {
	if e.apiKey == "" {
		return "", fmt.Errorf("Claude API key not configured for image OCR")
	}

	// Determine media type
	mediaType := e.detectMediaType(data)

	// Claude Vision only supports image formats, not PDF
	// For scanned PDFs, would need PDF → Image conversion first
	if mediaType == "application/pdf" {
		return "", fmt.Errorf("scanned PDF OCR requires PDF-to-image conversion (not yet implemented). Please use text-based PDFs or extract pages as images (PNG/JPEG)")
	}

	// Encode to base64
	base64Data := base64.StdEncoding.EncodeToString(data)

	// Create Claude Vision API request
	reqBody := map[string]interface{}{
		"model":      "claude-sonnet-4-5-20250929",
		"max_tokens": 4096,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "image",
						"source": map[string]interface{}{
							"type":       "base64",
							"media_type": mediaType,
							"data":       base64Data,
						},
					},
					{
						"type": "text",
						"text": "Please extract all text from this document. Preserve the structure and formatting as much as possible. Return ONLY the extracted text content, without any commentary or explanation.",
					},
				},
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", e.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("no content in OCR response")
	}

	extractedText := apiResp.Content[0].Text
	if extractedText == "" {
		return "", fmt.Errorf("no text extracted from document via OCR")
	}

	return extractedText, nil
}

// detectMediaType determines the media type from file signature
func (e *ImageOCRExtractor) detectMediaType(data []byte) string {
	if len(data) < 4 {
		return "application/pdf"
	}

	// Check PDF signature
	if bytes.HasPrefix(data, []byte("%PDF")) {
		return "application/pdf"
	}

	// Check PNG signature
	if bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47}) {
		return "image/png"
	}

	// Check JPEG signature
	if bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}) {
		return "image/jpeg"
	}

	// Check GIF signature
	if bytes.HasPrefix(data, []byte("GIF8")) {
		return "image/gif"
	}

	// Check WebP signature
	if bytes.HasPrefix(data, []byte("RIFF")) && len(data) >= 12 && bytes.Equal(data[8:12], []byte("WEBP")) {
		return "image/webp"
	}

	// Default to PDF for unknown
	return "application/pdf"
}

