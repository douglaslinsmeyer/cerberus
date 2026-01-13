package artifacts

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/cerberus/backend/internal/modules/artifacts/extractors"
	"github.com/cerberus/backend/internal/platform/storage"
	"github.com/google/uuid"
)

// OCRService handles OCR processing for scanned documents
type OCRService struct {
	repo       RepositoryInterface
	storage    storage.Storage
	ocrExtractor *extractors.ImageOCRExtractor
	chunker    *ChunkingStrategy
}

// NewOCRService creates a new OCR service
func NewOCRService(repo RepositoryInterface, stor storage.Storage) *OCRService {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	return &OCRService{
		repo:       repo,
		storage:    stor,
		ocrExtractor: extractors.NewImageOCRExtractor(apiKey),
		chunker:    DefaultChunkingStrategy(),
	}
}

// ProcessOCRRequired processes an artifact that needs OCR
func (s *OCRService) ProcessOCRRequired(ctx context.Context, artifactID uuid.UUID) error {
	// Get artifact
	artifact, err := s.repo.GetByID(ctx, artifactID)
	if err != nil {
		return fmt.Errorf("failed to get artifact: %w", err)
	}

	// Verify status
	if artifact.ProcessingStatus != "ocr_required" {
		return fmt.Errorf("artifact is not in ocr_required status (current: %s)", artifact.ProcessingStatus)
	}

	// Update status to processing
	if err := s.repo.UpdateStatus(ctx, artifactID, "processing"); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Download file from storage
	fileID := extractFileIDFromPath(artifact.StoragePath)
	data, err := s.storage.Download(ctx, fileID)
	if err != nil {
		s.repo.UpdateStatus(ctx, artifactID, "failed")
		return fmt.Errorf("failed to download file: %w", err)
	}

	// Perform OCR using Claude Vision
	extractedText, err := s.ocrExtractor.Extract(ctx, data)
	if err != nil {
		s.repo.UpdateStatus(ctx, artifactID, "failed")
		return fmt.Errorf("OCR extraction failed: %w", err)
	}

	// Update artifact with extracted content
	err = s.updateArtifactContent(ctx, artifactID, extractedText)
	if err != nil {
		s.repo.UpdateStatus(ctx, artifactID, "failed")
		return fmt.Errorf("failed to update content: %w", err)
	}

	// Chunk the extracted text
	chunks := s.chunker.ChunkDocument(extractedText)

	// Save chunks
	if err := s.repo.SaveChunks(ctx, artifactID, chunks); err != nil {
		s.repo.UpdateStatus(ctx, artifactID, "failed")
		return fmt.Errorf("failed to save chunks: %w", err)
	}

	// Update status to pending for AI analysis
	if err := s.repo.UpdateStatus(ctx, artifactID, "pending"); err != nil {
		return fmt.Errorf("failed to update status to pending: %w", err)
	}

	return nil
}

// updateArtifactContent updates the raw_content field
func (s *OCRService) updateArtifactContent(ctx context.Context, artifactID uuid.UUID, content string) error {
	query := `
		UPDATE artifacts
		SET raw_content = $1,
		    updated_at = NOW()
		WHERE artifact_id = $2
	`

	_, err := s.repo.ExecContext(ctx, query, sql.NullString{String: content, Valid: true}, artifactID)
	if err != nil {
		return fmt.Errorf("failed to update content: %w", err)
	}

	return nil
}

// extractFileIDFromPath extracts file ID from storage path
func extractFileIDFromPath(path string) string {
	// Path format: "/files/{fileID}" or "artifacts/{fileID}"
	parts := []rune{}
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			break
		}
		parts = append([]rune{rune(path[i])}, parts...)
	}
	return string(parts)
}
