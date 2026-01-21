package extractors

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// ZipExtractor extracts text from ZIP archive files
type ZipExtractor struct {
	factory *ExtractorFactory
}

// NewZipExtractor creates a new ZIP extractor
func NewZipExtractor() *ZipExtractor {
	return &ZipExtractor{}
}

// SetFactory sets the extractor factory for extracting files within the ZIP
func (e *ZipExtractor) SetFactory(factory *ExtractorFactory) {
	e.factory = factory
}

// CanExtract returns true for ZIP MIME types
func (e *ZipExtractor) CanExtract(mimeType string) bool {
	zipTypes := []string{
		"application/zip",
		"application/x-zip-compressed",
		"application/x-zip",
	}

	for _, t := range zipTypes {
		if strings.HasPrefix(mimeType, t) {
			return true
		}
	}

	return false
}

// Extract extracts text content from all files in a ZIP archive
func (e *ZipExtractor) Extract(ctx context.Context, data []byte) (string, error) {
	// Create a reader from the byte data
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("failed to open ZIP archive: %w", err)
	}

	var extractedTexts []string
	var fileCount int

	// Extract each file in the archive
	for _, file := range reader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		// Open the file within the ZIP
		rc, err := file.Open()
		if err != nil {
			// Log error but continue with other files
			extractedTexts = append(extractedTexts, fmt.Sprintf("\n--- %s ---\n[Error: Could not open file: %v]\n", file.Name, err))
			continue
		}

		// Read the file content
		fileData, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			extractedTexts = append(extractedTexts, fmt.Sprintf("\n--- %s ---\n[Error: Could not read file: %v]\n", file.Name, err))
			continue
		}

		// Skip empty files
		if len(fileData) == 0 {
			continue
		}

		// Determine MIME type from file extension
		mimeType := getMimeTypeFromFilename(file.Name)

		// Try to extract text if we have an appropriate extractor
		var extractedText string
		if e.factory != nil && e.factory.CanExtract(mimeType) {
			text, err := e.factory.Extract(ctx, mimeType, fileData)
			if err != nil {
				// If extraction fails, note the error but continue
				extractedText = fmt.Sprintf("[Error: Could not extract text from %s: %v]", file.Name, err)
			} else {
				extractedText = text
			}
		} else {
			// Try to treat as plain text if no extractor is available
			if isTextFile(fileData) {
				extractedText = string(fileData)
			} else {
				extractedText = fmt.Sprintf("[Binary file - no text extractor available for type: %s]", mimeType)
			}
		}

		// Add the extracted text with a clear file marker
		fileCount++
		extractedTexts = append(extractedTexts, fmt.Sprintf("\n--- File: %s ---\n%s\n", file.Name, strings.TrimSpace(extractedText)))
	}

	if fileCount == 0 {
		return "", fmt.Errorf("no extractable files found in ZIP archive")
	}

	// Combine all extracted texts
	result := fmt.Sprintf("ZIP Archive Contents (%d files):\n%s", fileCount, strings.Join(extractedTexts, "\n"))

	// Sanitize the result to remove null bytes (PostgreSQL doesn't allow them in UTF-8 text)
	result = sanitizeText(result)

	return result, nil
}

// getMimeTypeFromFilename guesses MIME type based on file extension
func getMimeTypeFromFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	mimeTypes := map[string]string{
		".pdf":  "application/pdf",
		".txt":  "text/plain",
		".md":   "text/markdown",
		".csv":  "text/csv",
		".json": "application/json",
		".xml":  "application/xml",
		".html": "text/html",
		".htm":  "text/html",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".xls":  "application/vnd.ms-excel",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".eml":  "message/rfc822",
		".msg":  "application/vnd.ms-outlook",
	}

	if mimeType, ok := mimeTypes[ext]; ok {
		return mimeType
	}

	// Default to plain text
	return "text/plain"
}

// isTextFile checks if the data appears to be text
func isTextFile(data []byte) bool {
	// Check if file is valid UTF-8 and doesn't contain too many null bytes
	if len(data) == 0 {
		return false
	}

	// Sample first 512 bytes (or less if file is smaller)
	sampleSize := 512
	if len(data) < sampleSize {
		sampleSize = len(data)
	}

	nullCount := 0
	for i := 0; i < sampleSize; i++ {
		if data[i] == 0 {
			nullCount++
		}
	}

	// If more than 10% null bytes, consider it binary
	if float64(nullCount)/float64(sampleSize) > 0.1 {
		return false
	}

	return true
}

// sanitizeText removes null bytes and other invalid characters from text
// PostgreSQL doesn't allow null bytes (0x00) in UTF-8 text columns
func sanitizeText(text string) string {
	// Remove null bytes
	text = strings.ReplaceAll(text, "\x00", "")

	// Remove other problematic control characters (except common whitespace)
	var sanitized strings.Builder
	sanitized.Grow(len(text))

	for _, r := range text {
		// Allow printable characters, newlines, tabs, and carriage returns
		if r >= 32 || r == '\n' || r == '\t' || r == '\r' {
			sanitized.WriteRune(r)
		}
		// Skip other control characters
	}

	return sanitized.String()
}
