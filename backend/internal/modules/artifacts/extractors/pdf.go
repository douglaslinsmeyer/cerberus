package extractors

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	pdf "github.com/ledongthuc/pdf"
)

// PDFExtractor extracts text from PDF files
type PDFExtractor struct{}

// NewPDFExtractor creates a new PDF extractor
func NewPDFExtractor() *PDFExtractor {
	return &PDFExtractor{}
}

// CanExtract returns true for PDF MIME types
func (e *PDFExtractor) CanExtract(mimeType string) bool {
	return mimeType == "application/pdf" ||
		strings.HasPrefix(mimeType, "application/pdf")
}

// Extract extracts text content from PDF data
func (e *PDFExtractor) Extract(ctx context.Context, data []byte) (string, error) {
	reader := bytes.NewReader(data)

	// Open PDF
	pdfReader, err := pdf.NewReader(reader, int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}

	// Extract text from all pages
	var content strings.Builder
	numPages := pdfReader.NumPage()

	for pageNum := 1; pageNum <= numPages; pageNum++ {
		page := pdfReader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			// Log but continue with other pages
			continue
		}

		content.WriteString(text)
		content.WriteString("\n\n")
	}

	extractedText := content.String()
	if strings.TrimSpace(extractedText) == "" {
		return "", fmt.Errorf("no text content extracted from PDF")
	}

	return extractedText, nil
}
