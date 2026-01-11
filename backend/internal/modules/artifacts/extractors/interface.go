package extractors

import "context"

// ContentExtractor defines the interface for extracting text content from different file types
type ContentExtractor interface {
	// CanExtract returns true if this extractor can handle the given MIME type
	CanExtract(mimeType string) bool

	// Extract extracts text content from the file data
	Extract(ctx context.Context, data []byte) (string, error)
}

// ExtractionResult contains the extracted content and metadata
type ExtractionResult struct {
	Content    string
	PageCount  int
	WordCount  int
	Metadata   map[string]string
}
