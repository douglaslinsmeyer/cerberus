package artifacts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// EmbeddingsService generates vector embeddings for semantic search
type EmbeddingsService struct {
	apiKey     string
	httpClient *http.Client
	repo       RepositoryInterface
}

// NewEmbeddingsService creates a new embeddings service
// Uses OpenAI embeddings API (text-embedding-3-small model)
func NewEmbeddingsService(apiKey string, repo RepositoryInterface) *EmbeddingsService {
	return &EmbeddingsService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		repo: repo,
	}
}

// GenerateEmbeddings generates vector embeddings for all chunks of an artifact
func (s *EmbeddingsService) GenerateEmbeddings(ctx context.Context, artifactID uuid.UUID) error {
	// Get all chunks for the artifact
	chunks, err := s.repo.GetChunks(ctx, artifactID)
	if err != nil {
		return fmt.Errorf("failed to get chunks: %w", err)
	}

	if len(chunks) == 0 {
		return fmt.Errorf("no chunks found for artifact")
	}

	// Generate embedding for each chunk
	for _, chunk := range chunks {
		embedding, err := s.generateEmbedding(ctx, chunk.ChunkText)
		if err != nil {
			return fmt.Errorf("failed to generate embedding for chunk %d: %w", chunk.ChunkIndex, err)
		}

		// Save embedding to database
		if err := s.saveEmbedding(ctx, artifactID, chunk.ChunkID, embedding); err != nil {
			return fmt.Errorf("failed to save embedding for chunk %d: %w", chunk.ChunkIndex, err)
		}
	}

	return nil
}

// generateEmbedding calls OpenAI API to generate embedding vector
func (s *EmbeddingsService) generateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// If no API key configured, return placeholder
	if s.apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not configured (set OPENAI_API_KEY environment variable)")
	}

	// OpenAI embeddings API request
	reqBody := map[string]interface{}{
		"input": text,
		"model": "text-embedding-3-small", // 1536 dimensions, $0.02/MTok
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	// Send request
	httpResp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", httpResp.StatusCode, string(respBody))
	}

	// Parse response
	var resp struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return resp.Data[0].Embedding, nil
}

// saveEmbedding stores an embedding vector in the database
func (s *EmbeddingsService) saveEmbedding(ctx context.Context, artifactID, chunkID uuid.UUID, embedding []float32) error {
	// Convert []float32 to PostgreSQL array format
	vectorStr := formatVector(embedding)

	query := `
		INSERT INTO artifact_embeddings (
			artifact_id, chunk_id, embedding, embedding_model
		) VALUES ($1, $2, $3, $4)
	`

	_, err := s.repo.ExecContext(ctx, query,
		artifactID,
		chunkID,
		vectorStr,
		"text-embedding-3-small",
	)

	if err != nil {
		return fmt.Errorf("failed to save embedding: %w", err)
	}

	return nil
}

// formatVector converts a float32 slice to PostgreSQL vector format
func formatVector(vec []float32) string {
	var buf bytes.Buffer
	buf.WriteString("[")
	for i, v := range vec {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(fmt.Sprintf("%f", v))
	}
	buf.WriteString("]")
	return buf.String()
}
