// Package artifacts provides service layer for artifact management.
// This includes upload, storage, content extraction, chunking, and metadata management.
package artifacts

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/cerberus/backend/internal/modules/artifacts/extractors"
	"github.com/cerberus/backend/internal/platform/storage"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// RepositoryInterface defines methods for artifact data access
type RepositoryInterface interface {
	Create(ctx context.Context, artifact *Artifact) error
	GetByID(ctx context.Context, artifactID uuid.UUID) (*Artifact, error)
	ListByProgram(ctx context.Context, programID uuid.UUID, limit, offset int) ([]Artifact, error)
	Delete(ctx context.Context, artifactID uuid.UUID) error
	UpdateStatus(ctx context.Context, artifactID uuid.UUID, status string) error
	SaveChunks(ctx context.Context, artifactID uuid.UUID, chunks []Chunk) error
	GetMetadata(ctx context.Context, artifactID uuid.UUID) (*ArtifactWithMetadata, error)
}

// DBExecutor defines methods for direct database access (for metadata clearing)
type DBExecutor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

// Service handles business logic for artifacts
type Service struct {
	repo       RepositoryInterface
	db         DBExecutor
	storage    storage.Storage
	extractors *extractors.ExtractorFactory
	chunker    *ChunkingStrategy
}

// NewService creates a new artifacts service
func NewService(repo *Repository, stor storage.Storage) *Service {
	return &Service{
		repo:       repo,
		db:         repo.db,
		storage:    stor,
		extractors: extractors.NewExtractorFactory(),
		chunker:    DefaultChunkingStrategy(),
	}
}

// NewServiceWithMocks creates a service with mock dependencies (useful for testing)
func NewServiceWithMocks(repo RepositoryInterface, db DBExecutor, stor storage.Storage) *Service {
	return &Service{
		repo:       repo,
		db:         db,
		storage:    stor,
		extractors: extractors.NewExtractorFactory(),
		chunker:    DefaultChunkingStrategy(),
	}
}

// UploadArtifact processes and stores a new artifact
func (s *Service) UploadArtifact(ctx context.Context, req UploadRequest) (uuid.UUID, error) {
	// Validate request
	if req.ProgramID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("program_id is required")
	}
	if req.UploadedBy == uuid.Nil {
		return uuid.Nil, fmt.Errorf("uploaded_by is required")
	}
	if req.Filename == "" {
		return uuid.Nil, fmt.Errorf("filename is required")
	}
	if len(req.Data) == 0 {
		return uuid.Nil, fmt.Errorf("file data is required")
	}

	// Calculate SHA-256 content hash
	hasher := sha256.New()
	hasher.Write(req.Data)
	contentHash := hex.EncodeToString(hasher.Sum(nil))

	// Check if extractor is available for this MIME type
	if !s.extractors.CanExtract(req.MimeType) {
		return uuid.Nil, fmt.Errorf("unsupported file type: %s", req.MimeType)
	}

	// Upload file to storage
	fileInfo, err := s.storage.Upload(ctx, req.Filename, req.Data)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to upload file to storage: %w", err)
	}

	// Extract text content
	rawContent, err := s.extractors.Extract(ctx, req.MimeType, req.Data)
	if err != nil {
		// Clean up uploaded file on extraction failure
		_ = s.storage.Delete(ctx, fileInfo.ID)
		return uuid.Nil, fmt.Errorf("failed to extract content: %w", err)
	}

	// Chunk the document
	chunks := s.chunker.ChunkDocument(rawContent)

	// Determine file type from MIME type or extension
	fileType := s.inferFileType(req.MimeType, req.Filename)

	// Create artifact record
	artifactID := uuid.New()
	artifact := &Artifact{
		ArtifactID:       artifactID,
		ProgramID:        req.ProgramID,
		Filename:         req.Filename,
		StoragePath:      fileInfo.Path,
		FileType:         fileType,
		FileSizeBytes:    fileInfo.Size,
		MimeType:         req.MimeType,
		ContentHash:      contentHash,
		RawContent:       sql.NullString{String: rawContent, Valid: true},
		ProcessingStatus: "pending",
		UploadedBy:       req.UploadedBy,
		UploadedAt:       time.Now(),
		VersionNumber:    1,
	}

	// Save artifact to database
	err = s.repo.Create(ctx, artifact)
	if err != nil {
		// Check for duplicate constraint violation
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" && strings.Contains(pqErr.Constraint, "content_hash_program_unique") {
				// Clean up uploaded file on duplicate
				_ = s.storage.Delete(ctx, fileInfo.ID)
				return uuid.Nil, fmt.Errorf("duplicate artifact: file with same content already exists in this program")
			}
		}
		// Clean up uploaded file on database error
		_ = s.storage.Delete(ctx, fileInfo.ID)
		return uuid.Nil, fmt.Errorf("failed to create artifact record: %w", err)
	}

	// Save chunks to database
	chunkRecords := make([]Chunk, len(chunks))
	for i, chunk := range chunks {
		chunkRecords[i] = Chunk{
			Index:       chunk.Index,
			Text:        chunk.Text,
			StartOffset: chunk.StartOffset,
			EndOffset:   chunk.EndOffset,
			TokenCount:  chunk.TokenCount,
		}
	}

	err = s.repo.SaveChunks(ctx, artifactID, chunkRecords)
	if err != nil {
		// Artifact is created but chunks failed - mark as failed
		_ = s.repo.UpdateStatus(ctx, artifactID, "failed")
		return uuid.Nil, fmt.Errorf("failed to save chunks: %w", err)
	}

	return artifactID, nil
}

// GetArtifact retrieves an artifact by ID
func (s *Service) GetArtifact(ctx context.Context, artifactID uuid.UUID) (*Artifact, error) {
	if artifactID == uuid.Nil {
		return nil, fmt.Errorf("artifact_id is required")
	}

	artifact, err := s.repo.GetByID(ctx, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	return artifact, nil
}

// GetArtifactWithMetadata retrieves an artifact with all its metadata
func (s *Service) GetArtifactWithMetadata(ctx context.Context, artifactID uuid.UUID) (*ArtifactWithMetadata, error) {
	if artifactID == uuid.Nil {
		return nil, fmt.Errorf("artifact_id is required")
	}

	metadata, err := s.repo.GetMetadata(ctx, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact metadata: %w", err)
	}

	return metadata, nil
}

// ListArtifacts retrieves artifacts for a program with optional status filter
func (s *Service) ListArtifacts(ctx context.Context, programID uuid.UUID, status string, limit, offset int) ([]Artifact, error) {
	if programID == uuid.Nil {
		return nil, fmt.Errorf("program_id is required")
	}

	// Validate limit and offset
	if limit <= 0 {
		limit = 50 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Max limit
	}
	if offset < 0 {
		offset = 0
	}

	// If status filter is provided, use filtered query
	if status != "" {
		return s.listArtifactsByStatus(ctx, programID, status, limit, offset)
	}

	// Otherwise use standard list
	artifacts, err := s.repo.ListByProgram(ctx, programID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list artifacts: %w", err)
	}

	return artifacts, nil
}

// listArtifactsByStatus is a helper for filtering by status
func (s *Service) listArtifactsByStatus(ctx context.Context, programID uuid.UUID, status string, limit, offset int) ([]Artifact, error) {
	// Validate status
	validStatuses := map[string]bool{
		"pending":    true,
		"processing": true,
		"completed":  true,
		"failed":     true,
	}
	if !validStatuses[status] {
		return nil, fmt.Errorf("invalid status: %s (must be pending, processing, completed, or failed)", status)
	}

	// Query artifacts with status filter
	query := `
		SELECT artifact_id, program_id, filename, storage_path, file_type,
			   file_size_bytes, mime_type, content_hash,
			   artifact_category, artifact_subcategory,
			   processing_status, processed_at, ai_model_version,
			   uploaded_by, uploaded_at, version_number
		FROM artifacts
		WHERE program_id = $1
		  AND processing_status = $2
		  AND deleted_at IS NULL
		ORDER BY uploaded_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := s.db.QueryContext(ctx, query, programID, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query artifacts by status: %w", err)
	}
	defer rows.Close()

	var artifacts []Artifact
	for rows.Next() {
		var a Artifact
		err := rows.Scan(
			&a.ArtifactID,
			&a.ProgramID,
			&a.Filename,
			&a.StoragePath,
			&a.FileType,
			&a.FileSizeBytes,
			&a.MimeType,
			&a.ContentHash,
			&a.ArtifactCategory,
			&a.ArtifactSubcategory,
			&a.ProcessingStatus,
			&a.ProcessedAt,
			&a.AIModelVersion,
			&a.UploadedBy,
			&a.UploadedAt,
			&a.VersionNumber,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan artifact: %w", err)
		}
		artifacts = append(artifacts, a)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating artifacts: %w", err)
	}

	return artifacts, nil
}

// DeleteArtifact soft-deletes an artifact and removes it from storage
func (s *Service) DeleteArtifact(ctx context.Context, artifactID uuid.UUID) error {
	if artifactID == uuid.Nil {
		return fmt.Errorf("artifact_id is required")
	}

	// Get artifact to retrieve storage path
	artifact, err := s.repo.GetByID(ctx, artifactID)
	if err != nil {
		return fmt.Errorf("failed to get artifact: %w", err)
	}

	// Soft delete in database first
	err = s.repo.Delete(ctx, artifactID)
	if err != nil {
		return fmt.Errorf("failed to delete artifact: %w", err)
	}

	// Extract file ID from storage path
	// StoragePath format is expected to be something like "artifacts/file_id"
	fileID := filepath.Base(artifact.StoragePath)

	// Delete from storage (best effort - don't fail if storage delete fails)
	err = s.storage.Delete(ctx, fileID)
	if err != nil {
		// Log the error but don't fail the operation since DB is already updated
		// In production, this should be logged properly
		fmt.Printf("warning: failed to delete file from storage: %v\n", err)
	}

	return nil
}

// QueueForReanalysis resets an artifact for reprocessing
func (s *Service) QueueForReanalysis(ctx context.Context, artifactID uuid.UUID) error {
	if artifactID == uuid.Nil {
		return fmt.Errorf("artifact_id is required")
	}

	// Verify artifact exists
	_, err := s.repo.GetByID(ctx, artifactID)
	if err != nil {
		return fmt.Errorf("failed to get artifact: %w", err)
	}

	// Clear existing metadata in a transaction-like manner
	// Delete in reverse order of dependencies
	err = s.clearArtifactMetadata(ctx, artifactID)
	if err != nil {
		return fmt.Errorf("failed to clear metadata: %w", err)
	}

	// Reset status to pending
	err = s.repo.UpdateStatus(ctx, artifactID, "pending")
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// clearArtifactMetadata removes all AI-generated metadata for an artifact
func (s *Service) clearArtifactMetadata(ctx context.Context, artifactID uuid.UUID) error {
	// Delete insights
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM artifact_insights WHERE artifact_id = $1
	`, artifactID)
	if err != nil {
		return fmt.Errorf("failed to delete insights: %w", err)
	}

	// Delete facts
	_, err = s.db.ExecContext(ctx, `
		DELETE FROM artifact_facts WHERE artifact_id = $1
	`, artifactID)
	if err != nil {
		return fmt.Errorf("failed to delete facts: %w", err)
	}

	// Delete persons
	_, err = s.db.ExecContext(ctx, `
		DELETE FROM artifact_persons WHERE artifact_id = $1
	`, artifactID)
	if err != nil {
		return fmt.Errorf("failed to delete persons: %w", err)
	}

	// Delete topics
	_, err = s.db.ExecContext(ctx, `
		DELETE FROM artifact_topics WHERE artifact_id = $1
	`, artifactID)
	if err != nil {
		return fmt.Errorf("failed to delete topics: %w", err)
	}

	// Delete summary
	_, err = s.db.ExecContext(ctx, `
		DELETE FROM artifact_summaries WHERE artifact_id = $1
	`, artifactID)
	if err != nil {
		return fmt.Errorf("failed to delete summary: %w", err)
	}

	// Delete embeddings
	_, err = s.db.ExecContext(ctx, `
		DELETE FROM artifact_embeddings WHERE artifact_id = $1
	`, artifactID)
	if err != nil {
		return fmt.Errorf("failed to delete embeddings: %w", err)
	}

	return nil
}

// inferFileType determines file type from MIME type or filename extension
func (s *Service) inferFileType(mimeType, filename string) string {
	// First try MIME type
	switch mimeType {
	case "application/pdf":
		return "pdf"
	case "text/plain":
		return "text"
	case "text/markdown":
		return "markdown"
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return "docx"
	case "application/msword":
		return "doc"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return "xlsx"
	case "application/vnd.ms-excel":
		return "xls"
	case "text/csv":
		return "csv"
	case "application/json":
		return "json"
	case "text/html":
		return "html"
	case "application/xml", "text/xml":
		return "xml"
	}

	// Fall back to file extension
	ext := strings.ToLower(filepath.Ext(filename))
	if len(ext) > 0 && ext[0] == '.' {
		return ext[1:] // Remove the leading dot
	}

	// Default to unknown
	return "unknown"
}
