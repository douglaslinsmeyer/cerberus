package artifacts

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cerberus/backend/internal/platform/db"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Repository handles database operations for artifacts
type Repository struct {
	db *db.DB
}

// NewRepository creates a new artifacts repository
func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

// ExecContext executes a query without returning rows
func (r *Repository) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return r.db.ExecContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows
func (r *Repository) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return r.db.QueryContext(ctx, query, args...)
}

// Create inserts a new artifact
func (r *Repository) Create(ctx context.Context, artifact *Artifact) error {
	query := `
		INSERT INTO artifacts (
			artifact_id, program_id, filename, storage_path, file_type,
			file_size_bytes, mime_type, content_hash, raw_content,
			processing_status, uploaded_by, uploaded_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.ExecContext(ctx, query,
		artifact.ArtifactID,
		artifact.ProgramID,
		artifact.Filename,
		artifact.StoragePath,
		artifact.FileType,
		artifact.FileSizeBytes,
		artifact.MimeType,
		artifact.ContentHash,
		artifact.RawContent,
		artifact.ProcessingStatus,
		artifact.UploadedBy,
		artifact.UploadedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create artifact: %w", err)
	}

	return nil
}

// GetByID retrieves an artifact by ID
func (r *Repository) GetByID(ctx context.Context, artifactID uuid.UUID) (*Artifact, error) {
	query := `
		SELECT artifact_id, program_id, filename, storage_path, file_type,
			   file_size_bytes, mime_type, content_hash, raw_content,
			   artifact_category, artifact_subcategory,
			   processing_status, processed_at, ai_model_version, ai_processing_time_ms,
			   uploaded_by, uploaded_at, version_number, superseded_by, deleted_at
		FROM artifacts
		WHERE artifact_id = $1 AND deleted_at IS NULL
	`

	var artifact Artifact
	err := r.db.QueryRowContext(ctx, query, artifactID).Scan(
		&artifact.ArtifactID,
		&artifact.ProgramID,
		&artifact.Filename,
		&artifact.StoragePath,
		&artifact.FileType,
		&artifact.FileSizeBytes,
		&artifact.MimeType,
		&artifact.ContentHash,
		&artifact.RawContent,
		&artifact.ArtifactCategory,
		&artifact.ArtifactSubcategory,
		&artifact.ProcessingStatus,
		&artifact.ProcessedAt,
		&artifact.AIModelVersion,
		&artifact.AIProcessingTimeMs,
		&artifact.UploadedBy,
		&artifact.UploadedAt,
		&artifact.VersionNumber,
		&artifact.SupersededBy,
		&artifact.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("artifact not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	return &artifact, nil
}

// ListByProgram retrieves all artifacts for a program
func (r *Repository) ListByProgram(ctx context.Context, programID uuid.UUID, limit, offset int) ([]Artifact, error) {
	query := `
		SELECT artifact_id, program_id, filename, storage_path, file_type,
			   file_size_bytes, mime_type, content_hash,
			   artifact_category, artifact_subcategory,
			   processing_status, processed_at, ai_model_version,
			   uploaded_by, uploaded_at, version_number
		FROM artifacts
		WHERE program_id = $1 AND deleted_at IS NULL
		ORDER BY uploaded_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, programID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list artifacts: %w", err)
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

	return artifacts, nil
}

// UpdateStatus updates the processing status of an artifact
func (r *Repository) UpdateStatus(ctx context.Context, artifactID uuid.UUID, status string) error {
	query := `
		UPDATE artifacts
		SET processing_status = $1,
		    processed_at = CASE WHEN $2 IN ('completed', 'failed') THEN NOW() ELSE processed_at END
		WHERE artifact_id = $3
	`

	_, err := r.db.ExecContext(ctx, query, status, status, artifactID)
	if err != nil {
		return fmt.Errorf("failed to update artifact status: %w", err)
	}

	return nil
}

// GetPendingArtifacts retrieves artifacts that need AI processing
func (r *Repository) GetPendingArtifacts(ctx context.Context, limit int) ([]Artifact, error) {
	query := `
		SELECT artifact_id, program_id, filename, storage_path, file_type,
			   file_size_bytes, mime_type, content_hash, raw_content,
			   processing_status, uploaded_by, uploaded_at
		FROM artifacts
		WHERE processing_status = 'pending'
		  AND deleted_at IS NULL
		ORDER BY uploaded_at ASC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending artifacts: %w", err)
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
			&a.RawContent,
			&a.ProcessingStatus,
			&a.UploadedBy,
			&a.UploadedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan artifact: %w", err)
		}
		artifacts = append(artifacts, a)
	}

	return artifacts, nil
}

// Delete soft-deletes an artifact
func (r *Repository) Delete(ctx context.Context, artifactID uuid.UUID) error {
	query := `
		UPDATE artifacts
		SET deleted_at = NOW()
		WHERE artifact_id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, artifactID)
	if err != nil {
		return fmt.Errorf("failed to delete artifact: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("artifact not found or already deleted")
	}

	return nil
}

// SaveChunks stores document chunks
func (r *Repository) SaveChunks(ctx context.Context, artifactID uuid.UUID, chunks []Chunk) error {
	query := `
		INSERT INTO artifact_chunks (
			artifact_id, chunk_index, chunk_text,
			chunk_start_offset, chunk_end_offset, token_count
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, chunk := range chunks {
		_, err := r.db.ExecContext(ctx, query,
			artifactID,
			chunk.Index,
			chunk.Text,
			chunk.StartOffset,
			chunk.EndOffset,
			chunk.TokenCount,
		)
		if err != nil {
			return fmt.Errorf("failed to save chunk %d: %w", chunk.Index, err)
		}
	}

	return nil
}

// GetChunks retrieves all chunks for an artifact
func (r *Repository) GetChunks(ctx context.Context, artifactID uuid.UUID) ([]ArtifactChunk, error) {
	query := `
		SELECT chunk_id, artifact_id, chunk_index, chunk_text,
			   chunk_start_offset, chunk_end_offset, token_count, created_at
		FROM artifact_chunks
		WHERE artifact_id = $1
		ORDER BY chunk_index
	`

	rows, err := r.db.QueryContext(ctx, query, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunks: %w", err)
	}
	defer rows.Close()

	var chunks []ArtifactChunk
	for rows.Next() {
		var c ArtifactChunk
		err := rows.Scan(
			&c.ChunkID,
			&c.ArtifactID,
			&c.ChunkIndex,
			&c.ChunkText,
			&c.ChunkStartOffset,
			&c.ChunkEndOffset,
			&c.TokenCount,
			&c.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}
		chunks = append(chunks, c)
	}

	return chunks, nil
}

// SaveSummary stores an AI-generated summary
func (r *Repository) SaveSummary(ctx context.Context, summary *ArtifactSummary) error {
	query := `
		INSERT INTO artifact_summaries (
			summary_id, artifact_id, executive_summary, key_takeaways,
			sentiment, priority, confidence_score, ai_model
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (artifact_id) DO UPDATE SET
			executive_summary = EXCLUDED.executive_summary,
			key_takeaways = EXCLUDED.key_takeaways,
			sentiment = EXCLUDED.sentiment,
			priority = EXCLUDED.priority,
			confidence_score = EXCLUDED.confidence_score,
			ai_model = EXCLUDED.ai_model
	`

	_, err := r.db.ExecContext(ctx, query,
		summary.SummaryID,
		summary.ArtifactID,
		summary.ExecutiveSummary,
		pq.Array(summary.KeyTakeaways),
		summary.Sentiment,
		summary.Priority,
		summary.ConfidenceScore,
		summary.AIModel,
	)

	if err != nil {
		return fmt.Errorf("failed to save summary: %w", err)
	}

	return nil
}

// SaveTopics stores extracted topics
func (r *Repository) SaveTopics(ctx context.Context, topics []Topic) error {
	if len(topics) == 0 {
		return nil
	}

	query := `
		INSERT INTO artifact_topics (
			topic_id, artifact_id, topic_name, confidence_score, topic_level
		) VALUES ($1, $2, $3, $4, $5)
	`

	for _, topic := range topics {
		_, err := r.db.ExecContext(ctx, query,
			topic.TopicID,
			topic.ArtifactID,
			topic.TopicName,
			topic.ConfidenceScore,
			topic.TopicLevel,
		)
		if err != nil {
			return fmt.Errorf("failed to save topic: %w", err)
		}
	}

	return nil
}

// SavePersons stores extracted person mentions
func (r *Repository) SavePersons(ctx context.Context, persons []Person) error {
	if len(persons) == 0 {
		return nil
	}

	query := `
		INSERT INTO artifact_persons (
			person_id, artifact_id, person_name, person_role,
			person_organization, mention_count, context_snippets, confidence_score
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	for _, person := range persons {
		_, err := r.db.ExecContext(ctx, query,
			person.PersonID,
			person.ArtifactID,
			person.PersonName,
			person.PersonRole,
			person.PersonOrganization,
			person.MentionCount,
			person.ContextSnippets,
			person.ConfidenceScore,
		)
		if err != nil {
			return fmt.Errorf("failed to save person: %w", err)
		}
	}

	return nil
}

// SaveFacts stores extracted facts
func (r *Repository) SaveFacts(ctx context.Context, facts []Fact) error {
	if len(facts) == 0 {
		return nil
	}

	query := `
		INSERT INTO artifact_facts (
			fact_id, artifact_id, fact_type, fact_key, fact_value,
			normalized_value_numeric, normalized_value_date,
			normalized_value_boolean, unit, confidence_score, context_snippet
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	for _, fact := range facts {
		_, err := r.db.ExecContext(ctx, query,
			fact.FactID,
			fact.ArtifactID,
			fact.FactType,
			fact.FactKey,
			fact.FactValue,
			fact.NormalizedValueNumeric,
			fact.NormalizedValueDate,
			fact.NormalizedValueBoolean,
			fact.Unit,
			fact.ConfidenceScore,
			fact.ContextSnippet,
		)
		if err != nil {
			return fmt.Errorf("failed to save fact: %w", err)
		}
	}

	return nil
}

// SaveInsights stores AI-generated insights
func (r *Repository) SaveInsights(ctx context.Context, insights []Insight) error {
	if len(insights) == 0 {
		return nil
	}

	query := `
		INSERT INTO artifact_insights (
			insight_id, artifact_id, insight_type, insight_category,
			title, description, severity, suggested_action,
			impacted_modules, confidence_score
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	for _, insight := range insights {
		_, err := r.db.ExecContext(ctx, query,
			insight.InsightID,
			insight.ArtifactID,
			insight.InsightType,
			insight.InsightCategory,
			insight.Title,
			insight.Description,
			insight.Severity,
			insight.SuggestedAction,
			pq.Array(insight.ImpactedModules),
			insight.ConfidenceScore,
		)
		if err != nil {
			return fmt.Errorf("failed to save insight: %w", err)
		}
	}

	return nil
}

// GetMetadata retrieves all metadata for an artifact
func (r *Repository) GetMetadata(ctx context.Context, artifactID uuid.UUID) (*ArtifactWithMetadata, error) {
	// Get artifact
	artifact, err := r.GetByID(ctx, artifactID)
	if err != nil {
		return nil, err
	}

	result := &ArtifactWithMetadata{
		Artifact: *artifact,
	}

	// Get summary
	var summary ArtifactSummary
	err = r.db.QueryRowContext(ctx, `
		SELECT summary_id, artifact_id, executive_summary, key_takeaways,
			   sentiment, priority, confidence_score, ai_model, created_at
		FROM artifact_summaries
		WHERE artifact_id = $1
	`, artifactID).Scan(
		&summary.SummaryID,
		&summary.ArtifactID,
		&summary.ExecutiveSummary,
		pq.Array(&summary.KeyTakeaways),
		&summary.Sentiment,
		&summary.Priority,
		&summary.ConfidenceScore,
		&summary.AIModel,
		&summary.CreatedAt,
	)
	if err != sql.ErrNoRows {
		if err != nil {
			return nil, fmt.Errorf("failed to get summary: %w", err)
		}
		result.Summary = &summary
	}

	// Get topics
	topicRows, err := r.db.QueryContext(ctx, `
		SELECT topic_id, artifact_id, topic_name, confidence_score, parent_topic_id, topic_level, extracted_at
		FROM artifact_topics
		WHERE artifact_id = $1
		ORDER BY confidence_score DESC
	`, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get topics: %w", err)
	}
	defer topicRows.Close()

	for topicRows.Next() {
		var t Topic
		if err := topicRows.Scan(&t.TopicID, &t.ArtifactID, &t.TopicName, &t.ConfidenceScore, &t.ParentTopicID, &t.TopicLevel, &t.ExtractedAt); err != nil {
			return nil, err
		}
		result.Topics = append(result.Topics, t)
	}

	// Get persons
	personRows, err := r.db.QueryContext(ctx, `
		SELECT person_id, artifact_id, person_name, person_role, person_organization,
			   mention_count, context_snippets, confidence_score, stakeholder_id, extracted_at
		FROM artifact_persons
		WHERE artifact_id = $1
		ORDER BY confidence_score DESC
	`, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get persons: %w", err)
	}
	defer personRows.Close()

	for personRows.Next() {
		var p Person
		if err := personRows.Scan(&p.PersonID, &p.ArtifactID, &p.PersonName, &p.PersonRole, &p.PersonOrganization,
			&p.MentionCount, &p.ContextSnippets, &p.ConfidenceScore, &p.StakeholderID, &p.ExtractedAt); err != nil {
			return nil, err
		}
		result.Persons = append(result.Persons, p)
	}

	// Get facts
	factRows, err := r.db.QueryContext(ctx, `
		SELECT fact_id, artifact_id, fact_type, fact_key, fact_value,
			   normalized_value_numeric, normalized_value_date, normalized_value_boolean,
			   unit, confidence_score, context_snippet, extracted_at
		FROM artifact_facts
		WHERE artifact_id = $1
		ORDER BY fact_type, fact_key
	`, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get facts: %w", err)
	}
	defer factRows.Close()

	for factRows.Next() {
		var f Fact
		if err := factRows.Scan(&f.FactID, &f.ArtifactID, &f.FactType, &f.FactKey, &f.FactValue,
			&f.NormalizedValueNumeric, &f.NormalizedValueDate, &f.NormalizedValueBoolean,
			&f.Unit, &f.ConfidenceScore, &f.ContextSnippet, &f.ExtractedAt); err != nil {
			return nil, err
		}
		result.Facts = append(result.Facts, f)
	}

	// Get insights
	insightRows, err := r.db.QueryContext(ctx, `
		SELECT insight_id, artifact_id, insight_type, insight_category,
			   title, description, severity, suggested_action,
			   impacted_modules, confidence_score, user_rating, user_feedback,
			   is_dismissed, extracted_at, dismissed_at, dismissed_by
		FROM artifact_insights
		WHERE artifact_id = $1 AND is_dismissed = FALSE
		ORDER BY severity DESC, confidence_score DESC
	`, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get insights: %w", err)
	}
	defer insightRows.Close()

	for insightRows.Next() {
		var i Insight
		if err := insightRows.Scan(&i.InsightID, &i.ArtifactID, &i.InsightType, &i.InsightCategory,
			&i.Title, &i.Description, &i.Severity, &i.SuggestedAction,
			pq.Array(&i.ImpactedModules), &i.ConfidenceScore, &i.UserRating, &i.UserFeedback,
			&i.IsDismissed, &i.ExtractedAt, &i.DismissedAt, &i.DismissedBy); err != nil {
			return nil, err
		}
		result.Insights = append(result.Insights, i)
	}

	return result, nil
}
