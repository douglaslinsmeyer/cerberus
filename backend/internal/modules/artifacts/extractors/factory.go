package extractors

import (
	"context"
	"fmt"
)

// ExtractorFactory manages content extractors
type ExtractorFactory struct {
	extractors []ContentExtractor
}

// NewExtractorFactory creates a new extractor factory with all available extractors
func NewExtractorFactory() *ExtractorFactory {
	return &ExtractorFactory{
		extractors: []ContentExtractor{
			NewPDFExtractor(),
			NewTextExtractor(),
			NewExcelExtractor(),
			// Future: Add DOCX, Image extractors
		},
	}
}

// GetExtractor returns the appropriate extractor for the given MIME type
func (f *ExtractorFactory) GetExtractor(mimeType string) (ContentExtractor, error) {
	for _, extractor := range f.extractors {
		if extractor.CanExtract(mimeType) {
			return extractor, nil
		}
	}

	return nil, fmt.Errorf("no extractor available for MIME type: %s", mimeType)
}

// Extract extracts content using the appropriate extractor
func (f *ExtractorFactory) Extract(ctx context.Context, mimeType string, data []byte) (string, error) {
	extractor, err := f.GetExtractor(mimeType)
	if err != nil {
		return "", err
	}

	return extractor.Extract(ctx, data)
}

// CanExtract checks if any extractor can handle the MIME type
func (f *ExtractorFactory) CanExtract(mimeType string) bool {
	for _, extractor := range f.extractors {
		if extractor.CanExtract(mimeType) {
			return true
		}
	}
	return false
}
