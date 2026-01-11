package extractors

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

// TextExtractor extracts text from plain text files
type TextExtractor struct{}

// NewTextExtractor creates a new text extractor
func NewTextExtractor() *TextExtractor {
	return &TextExtractor{}
}

// CanExtract returns true for text MIME types
func (e *TextExtractor) CanExtract(mimeType string) bool {
	textTypes := []string{
		"text/plain",
		"text/markdown",
		"text/csv",
		"text/html",
		"text/xml",
		"application/json",
		"application/xml",
	}

	for _, t := range textTypes {
		if strings.HasPrefix(mimeType, t) {
			return true
		}
	}

	return false
}

// Extract extracts text content from plain text data
func (e *TextExtractor) Extract(ctx context.Context, data []byte) (string, error) {
	// Validate UTF-8
	if !utf8.Valid(data) {
		return "", fmt.Errorf("file is not valid UTF-8 text")
	}

	text := string(data)

	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("file contains no text content")
	}

	return text, nil
}
