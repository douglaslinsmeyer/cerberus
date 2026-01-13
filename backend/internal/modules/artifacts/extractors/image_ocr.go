package extractors

import (
	"context"
	"encoding/base64"
	"fmt"
)

// ImageOCRExtractor uses Claude Vision API for OCR on scanned PDFs and images
type ImageOCRExtractor struct {
	apiKey string
}

// NewImageOCRExtractor creates a new image OCR extractor
func NewImageOCRExtractor(apiKey string) *ImageOCRExtractor {
	return &ImageOCRExtractor{
		apiKey: apiKey,
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

	// For now, return a helpful message
	// TODO: Implement Claude Vision API integration
	// This would use the Messages API with image content blocks

	// Encode to base64
	base64Data := base64.StdEncoding.EncodeToString(data)
	_ = base64Data // Will use this for Claude Vision API

	return "", fmt.Errorf("scanned PDF/image OCR requires Claude Vision API integration (coming in Phase 4). For now, please use text-based PDFs or text files")
}
