package extractors

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
)

// EMLExtractor extracts text from email files (.eml)
type EMLExtractor struct{}

// NewEMLExtractor creates a new EML extractor
func NewEMLExtractor() *EMLExtractor {
	return &EMLExtractor{}
}

// CanExtract returns true for email MIME types
func (e *EMLExtractor) CanExtract(mimeType string) bool {
	emailTypes := []string{
		"message/rfc822",
		"application/vnd.ms-outlook",
		"message/x-emlx",
	}

	for _, t := range emailTypes {
		if strings.HasPrefix(mimeType, t) {
			return true
		}
	}

	return false
}

// Extract extracts text content from an email message
func (e *EMLExtractor) Extract(ctx context.Context, data []byte) (string, error) {
	// Parse the email message
	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to parse email: %w", err)
	}

	var result strings.Builder

	// Extract headers
	result.WriteString("=== Email Message ===\n\n")
	result.WriteString("--- Headers ---\n")

	if from := msg.Header.Get("From"); from != "" {
		result.WriteString(fmt.Sprintf("From: %s\n", from))
	}
	if to := msg.Header.Get("To"); to != "" {
		result.WriteString(fmt.Sprintf("To: %s\n", to))
	}
	if cc := msg.Header.Get("Cc"); cc != "" {
		result.WriteString(fmt.Sprintf("Cc: %s\n", cc))
	}
	if subject := msg.Header.Get("Subject"); subject != "" {
		result.WriteString(fmt.Sprintf("Subject: %s\n", subject))
	}
	if date := msg.Header.Get("Date"); date != "" {
		result.WriteString(fmt.Sprintf("Date: %s\n", date))
	}

	result.WriteString("\n--- Body ---\n\n")

	// Extract body content
	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		// Default to plain text if no content type specified
		body, err := io.ReadAll(msg.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read email body: %w", err)
		}
		result.WriteString(string(body))
	} else {
		// Parse the content type
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			// If parsing fails, try to read as plain text
			body, err := io.ReadAll(msg.Body)
			if err != nil {
				return "", fmt.Errorf("failed to read email body: %w", err)
			}
			result.WriteString(string(body))
		} else {
			// Handle different content types
			if strings.HasPrefix(mediaType, "multipart/") {
				// Handle multipart messages
				boundary := params["boundary"]
				if boundary == "" {
					return "", fmt.Errorf("multipart message missing boundary")
				}

				mr := multipart.NewReader(msg.Body, boundary)
				bodyText, attachments, err := e.extractMultipart(ctx, mr)
				if err != nil {
					return "", fmt.Errorf("failed to extract multipart content: %w", err)
				}

				result.WriteString(bodyText)

				if len(attachments) > 0 {
					result.WriteString("\n\n--- Attachments ---\n")
					for _, att := range attachments {
						result.WriteString(fmt.Sprintf("- %s (%s)\n", att.Filename, att.ContentType))
					}
				}
			} else if strings.HasPrefix(mediaType, "text/") {
				// Handle simple text content
				body, err := io.ReadAll(msg.Body)
				if err != nil {
					return "", fmt.Errorf("failed to read email body: %w", err)
				}
				result.WriteString(string(body))
			} else {
				// Unsupported content type, try to read anyway
				body, err := io.ReadAll(msg.Body)
				if err != nil {
					return "", fmt.Errorf("failed to read email body: %w", err)
				}
				result.WriteString(fmt.Sprintf("[Content Type: %s]\n%s", mediaType, string(body)))
			}
		}
	}

	extracted := result.String()
	if strings.TrimSpace(extracted) == "" {
		return "", fmt.Errorf("no text content found in email")
	}

	return extracted, nil
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string
	ContentType string
	Size        int64
}

// extractMultipart extracts content from a multipart email message
func (e *EMLExtractor) extractMultipart(ctx context.Context, mr *multipart.Reader) (string, []Attachment, error) {
	var bodyText strings.Builder
	var attachments []Attachment
	var plainTextPart string
	var htmlPart string

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return "", nil, ctx.Err()
		default:
		}

		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", nil, fmt.Errorf("failed to read multipart part: %w", err)
		}

		contentType := part.Header.Get("Content-Type")
		mediaType, _, _ := mime.ParseMediaType(contentType)

		contentDisposition := part.Header.Get("Content-Disposition")
		disposition, params, _ := mime.ParseMediaType(contentDisposition)

		// Check if this is an attachment
		if disposition == "attachment" || (disposition == "inline" && params["filename"] != "") {
			filename := params["filename"]
			if filename == "" {
				filename = "unnamed"
			}

			// Read the attachment to get size (but don't store content)
			data, err := io.ReadAll(part)
			if err != nil {
				continue
			}

			attachments = append(attachments, Attachment{
				Filename:    filename,
				ContentType: mediaType,
				Size:        int64(len(data)),
			})
			continue
		}

		// Extract body content based on content type
		switch {
		case strings.HasPrefix(mediaType, "text/plain"):
			data, err := io.ReadAll(part)
			if err != nil {
				continue
			}
			plainTextPart = string(data)

		case strings.HasPrefix(mediaType, "text/html"):
			data, err := io.ReadAll(part)
			if err != nil {
				continue
			}
			htmlPart = string(data)

		case strings.HasPrefix(mediaType, "multipart/"):
			// Nested multipart (e.g., multipart/alternative within multipart/mixed)
			// For simplicity, we'll just note it
			bodyText.WriteString("[Nested multipart content]\n")

		default:
			// Skip other content types
			continue
		}
	}

	// Prefer plain text over HTML
	if plainTextPart != "" {
		bodyText.WriteString(plainTextPart)
	} else if htmlPart != "" {
		// Strip basic HTML tags for better readability
		cleanHtml := stripBasicHTMLTags(htmlPart)
		bodyText.WriteString(cleanHtml)
	}

	return bodyText.String(), attachments, nil
}

// stripBasicHTMLTags removes common HTML tags to make content more readable
func stripBasicHTMLTags(html string) string {
	// Simple HTML tag removal (not comprehensive, but good enough for basic emails)
	replacements := map[string]string{
		"<br>":    "\n",
		"<br/>":   "\n",
		"<br />":  "\n",
		"</p>":    "\n\n",
		"</div>":  "\n",
		"</h1>":   "\n\n",
		"</h2>":   "\n\n",
		"</h3>":   "\n\n",
		"</li>":   "\n",
		"&nbsp;":  " ",
		"&amp;":   "&",
		"&lt;":    "<",
		"&gt;":    ">",
		"&quot;":  "\"",
		"&#39;":   "'",
	}

	result := html
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}

	// Remove remaining HTML tags (simple approach)
	var cleaned strings.Builder
	inTag := false
	for _, ch := range result {
		if ch == '<' {
			inTag = true
			continue
		}
		if ch == '>' {
			inTag = false
			continue
		}
		if !inTag {
			cleaned.WriteRune(ch)
		}
	}

	return strings.TrimSpace(cleaned.String())
}
