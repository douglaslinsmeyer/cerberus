package artifacts

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// SearchService handles semantic search using vector embeddings
type SearchService struct {
	repo              RepositoryInterface
	embeddingsService *EmbeddingsService
}

// NewSearchService creates a new search service
func NewSearchService(repo RepositoryInterface, embeddingsService *EmbeddingsService) *SearchService {
	return &SearchService{
		repo:              repo,
		embeddingsService: embeddingsService,
	}
}

// SemanticSearch performs vector similarity search
func (s *SearchService) SemanticSearch(ctx context.Context, req *SearchRequest) ([]SearchResult, error) {
	// Generate embedding for search query
	queryEmbedding, err := s.embeddingsService.generateEmbedding(ctx, req.Query)
	if err != nil {
		// Fall back to full-text search if embeddings fail
		return s.fullTextSearch(ctx, req)
	}

	// Query vector similarity using pgvector
	query := `
		SELECT DISTINCT
			a.artifact_id,
			a.program_id,
			a.filename,
			a.file_type,
			a.file_size_bytes,
			a.mime_type,
			a.artifact_category,
			a.processing_status,
			a.uploaded_at,
			ae.chunk_id,
			ac.chunk_text,
			1 - (ae.embedding <=> $1::vector) AS similarity
		FROM artifacts a
		JOIN artifact_embeddings ae ON a.artifact_id = ae.artifact_id
		JOIN artifact_chunks ac ON ae.chunk_id = ac.chunk_id
		WHERE a.program_id = $2
		  AND a.deleted_at IS NULL
		  AND a.processing_status = 'completed'
		ORDER BY similarity DESC
		LIMIT $3
	`

	vectorStr := formatVector(queryEmbedding)
	rows, err := s.repo.QueryContext(ctx, query, vectorStr, req.ProgramID, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	seenArtifacts := make(map[uuid.UUID]bool)

	for rows.Next() {
		var result SearchResult
		var chunkID uuid.UUID
		var similarity float64

		err := rows.Scan(
			&result.Artifact.ArtifactID,
			&result.Artifact.ProgramID,
			&result.Artifact.Filename,
			&result.Artifact.FileType,
			&result.Artifact.FileSizeBytes,
			&result.Artifact.MimeType,
			&result.Artifact.ArtifactCategory,
			&result.Artifact.ProcessingStatus,
			&result.Artifact.UploadedAt,
			&chunkID,
			&result.Snippet,
			&similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		result.Similarity = similarity

		// Only include each artifact once (highest similarity chunk)
		if !seenArtifacts[result.Artifact.ArtifactID] {
			results = append(results, result)
			seenArtifacts[result.Artifact.ArtifactID] = true
		}
	}

	return results, nil
}

// fullTextSearch performs PostgreSQL full-text search as fallback
func (s *SearchService) fullTextSearch(ctx context.Context, req *SearchRequest) ([]SearchResult, error) {
	query := `
		SELECT
			artifact_id,
			program_id,
			filename,
			file_type,
			file_size_bytes,
			mime_type,
			artifact_category,
			processing_status,
			uploaded_at,
			ts_headline('english', raw_content, plainto_tsquery('english', $1)) AS snippet,
			ts_rank(to_tsvector('english', raw_content), plainto_tsquery('english', $1)) AS rank
		FROM artifacts
		WHERE program_id = $2
		  AND deleted_at IS NULL
		  AND processing_status = 'completed'
		  AND to_tsvector('english', raw_content) @@ plainto_tsquery('english', $1)
		ORDER BY rank DESC
		LIMIT $3
	`

	rows, err := s.repo.QueryContext(ctx, query, req.Query, req.ProgramID, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute full-text search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		var rank float64

		err := rows.Scan(
			&result.Artifact.ArtifactID,
			&result.Artifact.ProgramID,
			&result.Artifact.Filename,
			&result.Artifact.FileType,
			&result.Artifact.FileSizeBytes,
			&result.Artifact.MimeType,
			&result.Artifact.ArtifactCategory,
			&result.Artifact.ProcessingStatus,
			&result.Artifact.UploadedAt,
			&result.Snippet,
			&rank,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		result.Similarity = rank // Use rank as similarity for full-text search
		results = append(results, result)
	}

	return results, nil
}

// HybridSearch combines vector and full-text search
func (s *SearchService) HybridSearch(ctx context.Context, req *SearchRequest) ([]SearchResult, error) {
	// Try vector search first
	vectorResults, vectorErr := s.SemanticSearch(ctx, req)

	// If vector search fails, fall back to full-text
	if vectorErr != nil {
		return s.fullTextSearch(ctx, req)
	}

	// TODO: In the future, combine results from both methods
	// For now, just return vector results
	return vectorResults, nil
}
