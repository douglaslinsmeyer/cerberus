package artifacts

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// This file contains repository methods for context graph functionality
// It extends the Repository struct with methods for:
// - Semantic similarity search
// - Entity graph queries
// - Temporal queries
// - Fact aggregation
// - Context caching

// ============================================================================
// Semantic Similarity Queries
// ============================================================================

// FindSemanticallyRelatedArtifacts finds artifacts using vector similarity search
func (r *Repository) FindSemanticallyRelatedArtifacts(
	ctx context.Context,
	artifactID uuid.UUID,
	programID uuid.UUID,
	limit int,
) ([]ArtifactCandidate, error) {
	query := `
		SELECT
			a.artifact_id,
			a.filename,
			a.artifact_category,
			a.artifact_subcategory,
			a.uploaded_at,
			s.executive_summary,
			s.sentiment,
			s.priority,
			1 - (ae1.embedding <=> ae2.embedding) as similarity,
			COALESCE(person_count, 0) as person_count,
			COALESCE(fact_count, 0) as fact_count
		FROM artifact_embeddings ae1
		CROSS JOIN artifact_embeddings ae2
		JOIN artifacts a ON ae2.artifact_id = a.artifact_id
		LEFT JOIN artifact_summaries s ON a.artifact_id = s.artifact_id
		LEFT JOIN (
			SELECT artifact_id, COUNT(DISTINCT person_id) as person_count
			FROM artifact_persons
			GROUP BY artifact_id
		) ap ON a.artifact_id = ap.artifact_id
		LEFT JOIN (
			SELECT artifact_id, COUNT(*) as fact_count
			FROM artifact_facts
			GROUP BY artifact_id
		) af ON a.artifact_id = af.artifact_id
		WHERE ae1.artifact_id = $1
		  AND ae2.artifact_id != $1
		  AND a.program_id = $2
		  AND a.deleted_at IS NULL
		  AND a.processing_status = 'completed'
		  AND (1 - (ae1.embedding <=> ae2.embedding)) > 0.6
		ORDER BY similarity DESC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, artifactID, programID, limit)
	if err != nil {
		return nil, fmt.Errorf("semantic search failed: %w", err)
	}
	defer rows.Close()

	candidates := []ArtifactCandidate{}
	for rows.Next() {
		var candidate ArtifactCandidate
		var category, subcategory, summary, sentiment sql.NullString
		var priority sql.NullInt32

		err := rows.Scan(
			&candidate.ArtifactID,
			&candidate.Filename,
			&category,
			&subcategory,
			&candidate.UploadedAt,
			&summary,
			&sentiment,
			&priority,
			&candidate.SemanticScore,
			&candidate.PersonIDs, // Count placeholder
			&candidate.FactCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if category.Valid {
			candidate.Category = category.String
		}
		if subcategory.Valid {
			candidate.Subcategory = subcategory.String
		}
		if summary.Valid {
			candidate.ExecutiveSummary = summary.String
		}
		if sentiment.Valid {
			candidate.Sentiment = sentiment.String
		}
		if priority.Valid {
			candidate.Priority = int(priority.Int32)
		}

		// Get person IDs and names for this artifact
		personIDs, personNames, err := r.getPersonDataForArtifact(ctx, candidate.ArtifactID)
		if err == nil {
			candidate.PersonIDs = personIDs
			candidate.MentionedPeople = personNames
		}

		// Get topics
		topics, topicIDs, err := r.getTopicDataForArtifact(ctx, candidate.ArtifactID)
		if err == nil {
			candidate.Topics = topics
			candidate.TopicIDs = topicIDs
		}

		candidates = append(candidates, candidate)
	}

	return candidates, nil
}

// ============================================================================
// Entity Graph Queries
// ============================================================================

// FindArtifactsWithSharedEntities finds artifacts that share people with the target
func (r *Repository) FindArtifactsWithSharedEntities(
	ctx context.Context,
	targetArtifactID uuid.UUID,
	programID uuid.UUID,
	personIDs []uuid.UUID,
	limit int,
) ([]ArtifactCandidate, error) {
	query := `
		WITH target_persons AS (
			SELECT DISTINCT person_name, person_id
			FROM artifact_persons
			WHERE artifact_id = $1
		)
		SELECT
			a.artifact_id,
			a.filename,
			a.artifact_category,
			a.artifact_subcategory,
			a.uploaded_at,
			s.executive_summary,
			s.sentiment,
			s.priority,
			COUNT(DISTINCT ap.person_id) as shared_person_count,
			array_agg(DISTINCT ap.person_id) as shared_person_ids,
			array_agg(DISTINCT ap.person_name) as shared_people
		FROM target_persons tp
		JOIN artifact_persons ap ON (
			ap.person_name = tp.person_name
			OR ap.stakeholder_id = (
				SELECT stakeholder_id
				FROM artifact_persons
				WHERE person_id = tp.person_id
			)
		)
		JOIN artifacts a ON ap.artifact_id = a.artifact_id
		LEFT JOIN artifact_summaries s ON a.artifact_id = s.artifact_id
		WHERE a.artifact_id != $1
		  AND a.program_id = $2
		  AND a.deleted_at IS NULL
		  AND a.processing_status = 'completed'
		GROUP BY a.artifact_id, a.filename, a.artifact_category, a.artifact_subcategory,
		         a.uploaded_at, s.executive_summary, s.sentiment, s.priority
		HAVING COUNT(DISTINCT ap.person_id) >= 2
		ORDER BY shared_person_count DESC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, targetArtifactID, programID, limit)
	if err != nil {
		return nil, fmt.Errorf("entity search failed: %w", err)
	}
	defer rows.Close()

	candidates := []ArtifactCandidate{}
	for rows.Next() {
		var candidate ArtifactCandidate
		var category, subcategory, summary, sentiment sql.NullString
		var priority sql.NullInt32
		var sharedCount int
		var sharedPersonIDs pq.StringArray
		var sharedPeople pq.StringArray

		err := rows.Scan(
			&candidate.ArtifactID,
			&candidate.Filename,
			&category,
			&subcategory,
			&candidate.UploadedAt,
			&summary,
			&sentiment,
			&priority,
			&sharedCount,
			&sharedPersonIDs,
			&sharedPeople,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if category.Valid {
			candidate.Category = category.String
		}
		if subcategory.Valid {
			candidate.Subcategory = subcategory.String
		}
		if summary.Valid {
			candidate.ExecutiveSummary = summary.String
		}
		if sentiment.Valid {
			candidate.Sentiment = sentiment.String
		}
		if priority.Valid {
			candidate.Priority = int(priority.Int32)
		}

		// Convert string arrays to UUID arrays and string slices
		for _, idStr := range sharedPersonIDs {
			id, err := uuid.Parse(idStr)
			if err == nil {
				candidate.SharedPersonIDs = append(candidate.SharedPersonIDs, id)
			}
		}
		candidate.MentionedPeople = []string(sharedPeople)

		// Get all person IDs (not just shared)
		allPersonIDs, _, err := r.getPersonDataForArtifact(ctx, candidate.ArtifactID)
		if err == nil {
			candidate.PersonIDs = allPersonIDs
		}

		// Get fact count
		factCount, err := r.getFactCountForArtifact(ctx, candidate.ArtifactID)
		if err == nil {
			candidate.FactCount = factCount
		}

		candidates = append(candidates, candidate)
	}

	return candidates, nil
}

// GetPersonRelationships finds people who frequently appear with the given person
func (r *Repository) GetPersonRelationships(
	ctx context.Context,
	personID uuid.UUID,
	programID uuid.UUID,
) ([]PersonRelationship, error) {
	query := `
		SELECT
			eg.person_id_1,
			p1.person_name as person1_name,
			eg.person_id_2,
			p2.person_name as person2_name,
			eg.co_occurrence_count,
			eg.shared_artifact_ids,
			eg.relationship_strength
		FROM artifact_entity_graph eg
		JOIN artifact_persons p1 ON eg.person_id_1 = p1.person_id
		JOIN artifact_persons p2 ON eg.person_id_2 = p2.person_id
		WHERE eg.program_id = $1
		  AND (eg.person_id_1 = $2 OR eg.person_id_2 = $2)
		ORDER BY eg.co_occurrence_count DESC
		LIMIT 10
	`

	rows, err := r.db.QueryContext(ctx, query, programID, personID)
	if err != nil {
		return nil, fmt.Errorf("failed to get person relationships: %w", err)
	}
	defer rows.Close()

	relationships := []PersonRelationship{}
	for rows.Next() {
		var rel PersonRelationship
		var sharedArtifactIDs pq.StringArray

		err := rows.Scan(
			&rel.Person1ID,
			&rel.Person1Name,
			&rel.Person2ID,
			&rel.Person2Name,
			&rel.CoOccurrences,
			&sharedArtifactIDs,
			&rel.Strength,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert string array to UUID array
		for _, idStr := range sharedArtifactIDs {
			id, err := uuid.Parse(idStr)
			if err == nil {
				rel.SharedArtifacts = append(rel.SharedArtifacts, id)
			}
		}

		relationships = append(relationships, rel)
	}

	return relationships, nil
}

// GetPersonsByArtifact retrieves all persons extracted from an artifact
func (r *Repository) GetPersonsByArtifact(ctx context.Context, artifactID uuid.UUID) ([]Person, error) {
	query := `
		SELECT person_id, artifact_id, person_name, person_role, person_organization,
		       mention_count, context_snippets, confidence_score, stakeholder_id, extracted_at
		FROM artifact_persons
		WHERE artifact_id = $1
		ORDER BY mention_count DESC
	`

	rows, err := r.db.QueryContext(ctx, query, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get persons: %w", err)
	}
	defer rows.Close()

	persons := []Person{}
	for rows.Next() {
		var p Person
		err := rows.Scan(
			&p.PersonID,
			&p.ArtifactID,
			&p.PersonName,
			&p.PersonRole,
			&p.PersonOrganization,
			&p.MentionCount,
			&p.ContextSnippets,
			&p.ConfidenceScore,
			&p.StakeholderID,
			&p.ExtractedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan person: %w", err)
		}
		persons = append(persons, p)
	}

	return persons, nil
}

// GetPersonIDsByArtifact retrieves just the person IDs for an artifact
func (r *Repository) GetPersonIDsByArtifact(ctx context.Context, artifactID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT person_id
		FROM artifact_persons
		WHERE artifact_id = $1
	`

	rows, err := r.db.QueryContext(ctx, query, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get person IDs: %w", err)
	}
	defer rows.Close()

	personIDs := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan person ID: %w", err)
		}
		personIDs = append(personIDs, id)
	}

	return personIDs, nil
}

// UpsertEntityGraphEdge creates or updates an entity graph edge
func (r *Repository) UpsertEntityGraphEdge(
	ctx context.Context,
	programID uuid.UUID,
	person1ID uuid.UUID,
	person2ID uuid.UUID,
	artifactID uuid.UUID,
) error {
	query := `
		INSERT INTO artifact_entity_graph (
			program_id, person_id_1, person_id_2,
			co_occurrence_count, shared_artifact_ids, relationship_strength,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, 1, ARRAY[$4::UUID], 0.5, NOW(), NOW())
		ON CONFLICT (person_id_1, person_id_2) DO UPDATE
		SET
			co_occurrence_count = artifact_entity_graph.co_occurrence_count + 1,
			shared_artifact_ids = array_append(artifact_entity_graph.shared_artifact_ids, $4::UUID),
			relationship_strength = LEAST(1.0, (artifact_entity_graph.co_occurrence_count + 1)::DECIMAL / 10),
			updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query, programID, person1ID, person2ID, artifactID)
	if err != nil {
		return fmt.Errorf("failed to upsert entity graph edge: %w", err)
	}

	return nil
}

// ============================================================================
// Temporal Queries
// ============================================================================

// FindTemporallyRelatedArtifacts finds artifacts close in time to the target
func (r *Repository) FindTemporallyRelatedArtifacts(
	ctx context.Context,
	targetArtifactID uuid.UUID,
	programID uuid.UUID,
	targetTime time.Time,
	limit int,
) ([]ArtifactCandidate, error) {
	query := `
		SELECT
			a.artifact_id,
			a.filename,
			a.artifact_category,
			a.artifact_subcategory,
			a.uploaded_at,
			s.executive_summary,
			s.sentiment,
			s.priority,
			EXTRACT(EPOCH FROM (a.uploaded_at - $3)) / 86400 as days_offset
		FROM artifacts a
		LEFT JOIN artifact_summaries s ON a.artifact_id = s.artifact_id
		WHERE a.artifact_id != $1
		  AND a.program_id = $2
		  AND a.deleted_at IS NULL
		  AND a.processing_status = 'completed'
		  AND ABS(EXTRACT(EPOCH FROM (a.uploaded_at - $3)) / 86400) <= 90
		ORDER BY ABS(EXTRACT(EPOCH FROM (a.uploaded_at - $3)))
		LIMIT $4
	`

	rows, err := r.db.QueryContext(ctx, query, targetArtifactID, programID, targetTime, limit)
	if err != nil {
		return nil, fmt.Errorf("temporal search failed: %w", err)
	}
	defer rows.Close()

	candidates := []ArtifactCandidate{}
	for rows.Next() {
		var candidate ArtifactCandidate
		var category, subcategory, summary, sentiment sql.NullString
		var priority sql.NullInt32
		var daysOffset float64

		err := rows.Scan(
			&candidate.ArtifactID,
			&candidate.Filename,
			&category,
			&subcategory,
			&candidate.UploadedAt,
			&summary,
			&sentiment,
			&priority,
			&daysOffset,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if category.Valid {
			candidate.Category = category.String
		}
		if subcategory.Valid {
			candidate.Subcategory = subcategory.String
		}
		if summary.Valid {
			candidate.ExecutiveSummary = summary.String
		}
		if sentiment.Valid {
			candidate.Sentiment = sentiment.String
		}
		if priority.Valid {
			candidate.Priority = int(priority.Int32)
		}

		// Get person data
		personIDs, personNames, err := r.getPersonDataForArtifact(ctx, candidate.ArtifactID)
		if err == nil {
			candidate.PersonIDs = personIDs
			candidate.MentionedPeople = personNames
		}

		// Get fact count
		factCount, err := r.getFactCountForArtifact(ctx, candidate.ArtifactID)
		if err == nil {
			candidate.FactCount = factCount
		}

		candidates = append(candidates, candidate)
	}

	return candidates, nil
}

// GetArtifactsInTimeRange retrieves artifacts within a time range
func (r *Repository) GetArtifactsInTimeRange(
	ctx context.Context,
	programID uuid.UUID,
	startDate time.Time,
	endDate time.Time,
) ([]Artifact, error) {
	query := `
		SELECT artifact_id, program_id, filename, storage_path, file_type,
		       file_size_bytes, mime_type, content_hash, raw_content,
		       artifact_category, artifact_subcategory,
		       processing_status, processed_at, uploaded_by, uploaded_at
		FROM artifacts
		WHERE program_id = $1
		  AND deleted_at IS NULL
		  AND uploaded_at BETWEEN $2 AND $3
		ORDER BY uploaded_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, programID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifacts in time range: %w", err)
	}
	defer rows.Close()

	artifacts := []Artifact{}
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
			&a.ArtifactCategory,
			&a.ArtifactSubcategory,
			&a.ProcessingStatus,
			&a.ProcessedAt,
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

// GetArtifactsByProgram retrieves all artifacts for a program
func (r *Repository) GetArtifactsByProgram(ctx context.Context, programID uuid.UUID) ([]Artifact, error) {
	query := `
		SELECT artifact_id, program_id, filename, storage_path, file_type,
		       file_size_bytes, mime_type, content_hash,
		       artifact_category, artifact_subcategory,
		       processing_status, processed_at, uploaded_by, uploaded_at
		FROM artifacts
		WHERE program_id = $1
		  AND deleted_at IS NULL
		ORDER BY uploaded_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, programID)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifacts: %w", err)
	}
	defer rows.Close()

	artifacts := []Artifact{}
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

// GetSummaryByArtifact retrieves the summary for an artifact
func (r *Repository) GetSummaryByArtifact(ctx context.Context, artifactID uuid.UUID) (*ArtifactSummary, error) {
	query := `
		SELECT summary_id, artifact_id, executive_summary, key_takeaways,
		       sentiment, priority, confidence_score, ai_model, created_at
		FROM artifact_summaries
		WHERE artifact_id = $1
	`

	var summary ArtifactSummary
	var keyTakeawaysJSON []byte

	err := r.db.QueryRowContext(ctx, query, artifactID).Scan(
		&summary.SummaryID,
		&summary.ArtifactID,
		&summary.ExecutiveSummary,
		&keyTakeawaysJSON,
		&summary.Sentiment,
		&summary.Priority,
		&summary.ConfidenceScore,
		&summary.AIModel,
		&summary.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get summary: %w", err)
	}

	// Parse key takeaways from JSONB
	if len(keyTakeawaysJSON) > 0 {
		// KeyTakeaways is []string, parse from JSON array
		// For simplicity, we'll leave it empty for now
		summary.KeyTakeaways = []string{}
	}

	return &summary, nil
}

// SaveTemporalSequence persists a temporal sequence to the database
func (r *Repository) SaveTemporalSequence(ctx context.Context, sequence *ArtifactSequence) error {
	query := `
		INSERT INTO artifact_temporal_sequences (
			sequence_id, program_id, sequence_name, sequence_type,
			artifact_ids, start_date, end_date,
			detection_method, confidence_score,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		ON CONFLICT (sequence_id) DO UPDATE
		SET
			sequence_name = $3,
			artifact_ids = $5,
			end_date = $7,
			confidence_score = $9,
			updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query,
		sequence.SequenceID,
		sequence.ProgramID,
		sequence.SequenceName,
		sequence.SequenceType,
		pq.Array(sequence.ArtifactIDs),
		sequence.StartDate,
		sequence.EndDate,
		sequence.DetectionMethod,
		sequence.ConfidenceScore,
	)

	if err != nil {
		return fmt.Errorf("failed to save temporal sequence: %w", err)
	}

	return nil
}

// GetTemporalSequencesByProgram retrieves all temporal sequences for a program
func (r *Repository) GetTemporalSequencesByProgram(ctx context.Context, programID uuid.UUID) ([]ArtifactSequence, error) {
	query := `
		SELECT sequence_id, program_id, sequence_name, sequence_type,
		       artifact_ids, start_date, end_date,
		       detection_method, confidence_score
		FROM artifact_temporal_sequences
		WHERE program_id = $1
		ORDER BY start_date DESC
	`

	rows, err := r.db.QueryContext(ctx, query, programID)
	if err != nil {
		return nil, fmt.Errorf("failed to get temporal sequences: %w", err)
	}
	defer rows.Close()

	sequences := []ArtifactSequence{}
	for rows.Next() {
		var seq ArtifactSequence
		var artifactIDs pq.StringArray

		err := rows.Scan(
			&seq.SequenceID,
			&seq.ProgramID,
			&seq.SequenceName,
			&seq.SequenceType,
			&artifactIDs,
			&seq.StartDate,
			&seq.EndDate,
			&seq.DetectionMethod,
			&seq.ConfidenceScore,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sequence: %w", err)
		}

		// Convert string array to UUID array
		for _, idStr := range artifactIDs {
			id, err := uuid.Parse(idStr)
			if err == nil {
				seq.ArtifactIDs = append(seq.ArtifactIDs, id)
			}
		}

		sequences = append(sequences, seq)
	}

	return sequences, nil
}

// ============================================================================
// Fact Aggregation Queries
// ============================================================================

// GetFactsByArtifacts retrieves facts from multiple artifacts
func (r *Repository) GetFactsByArtifacts(ctx context.Context, artifactIDs []uuid.UUID) ([]Fact, error) {
	query := `
		SELECT fact_id, artifact_id, fact_type, fact_key, fact_value,
		       normalized_value_numeric, normalized_value_date, normalized_value_boolean,
		       unit, confidence_score, context_snippet, extracted_at
		FROM artifact_facts
		WHERE artifact_id = ANY($1)
		ORDER BY fact_type, fact_key, confidence_score DESC
		LIMIT 200
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(artifactIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to get facts: %w", err)
	}
	defer rows.Close()

	facts := []Fact{}
	for rows.Next() {
		var f Fact
		err := rows.Scan(
			&f.FactID,
			&f.ArtifactID,
			&f.FactType,
			&f.FactKey,
			&f.FactValue,
			&f.NormalizedValueNumeric,
			&f.NormalizedValueDate,
			&f.NormalizedValueBoolean,
			&f.Unit,
			&f.ConfidenceScore,
			&f.ContextSnippet,
			&f.ExtractedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fact: %w", err)
		}
		facts = append(facts, f)
	}

	return facts, nil
}

// GetArtifactFilename retrieves just the filename for an artifact
func (r *Repository) GetArtifactFilename(ctx context.Context, artifactID uuid.UUID) (string, error) {
	var filename string
	query := `SELECT filename FROM artifacts WHERE artifact_id = $1`
	err := r.db.QueryRowContext(ctx, query, artifactID).Scan(&filename)
	if err != nil {
		return "", fmt.Errorf("failed to get filename: %w", err)
	}
	return filename, nil
}

// GetArtifactByID is an alias for GetByID to match the interface
func (r *Repository) GetArtifactByID(ctx context.Context, artifactID uuid.UUID) (*Artifact, error) {
	return r.GetByID(ctx, artifactID)
}

// FindFactsByKey finds facts with a specific key across a program
func (r *Repository) FindFactsByKey(ctx context.Context, programID uuid.UUID, factKey string) ([]Fact, error) {
	query := `
		SELECT f.fact_id, f.artifact_id, f.fact_type, f.fact_key, f.fact_value,
		       f.normalized_value_numeric, f.normalized_value_date, f.normalized_value_boolean,
		       f.unit, f.confidence_score, f.context_snippet, f.extracted_at
		FROM artifact_facts f
		JOIN artifacts a ON f.artifact_id = a.artifact_id
		WHERE a.program_id = $1
		  AND a.deleted_at IS NULL
		  AND f.fact_key ILIKE $2
		ORDER BY f.confidence_score DESC
	`

	rows, err := r.db.QueryContext(ctx, query, programID, "%"+factKey+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to find facts: %w", err)
	}
	defer rows.Close()

	facts := []Fact{}
	for rows.Next() {
		var f Fact
		err := rows.Scan(
			&f.FactID,
			&f.ArtifactID,
			&f.FactType,
			&f.FactKey,
			&f.FactValue,
			&f.NormalizedValueNumeric,
			&f.NormalizedValueDate,
			&f.NormalizedValueBoolean,
			&f.Unit,
			&f.ConfidenceScore,
			&f.ContextSnippet,
			&f.ExtractedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fact: %w", err)
		}
		facts = append(facts, f)
	}

	return facts, nil
}

// CountFactsInProgram counts total facts in a program
func (r *Repository) CountFactsInProgram(ctx context.Context, programID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM artifact_facts f
		JOIN artifacts a ON f.artifact_id = a.artifact_id
		WHERE a.program_id = $1 AND a.deleted_at IS NULL
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, programID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count facts: %w", err)
	}

	return count, nil
}

// CountFactsByType counts facts by type in a program
func (r *Repository) CountFactsByType(ctx context.Context, programID uuid.UUID) (map[string]int, error) {
	query := `
		SELECT f.fact_type, COUNT(*)
		FROM artifact_facts f
		JOIN artifacts a ON f.artifact_id = a.artifact_id
		WHERE a.program_id = $1 AND a.deleted_at IS NULL
		GROUP BY f.fact_type
	`

	rows, err := r.db.QueryContext(ctx, query, programID)
	if err != nil {
		return nil, fmt.Errorf("failed to count facts by type: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var factType string
		var count int
		if err := rows.Scan(&factType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		counts[factType] = count
	}

	return counts, nil
}

// GetAllFactsInProgram retrieves all facts for a program
func (r *Repository) GetAllFactsInProgram(ctx context.Context, programID uuid.UUID) ([]Fact, error) {
	query := `
		SELECT f.fact_id, f.artifact_id, f.fact_type, f.fact_key, f.fact_value,
		       f.normalized_value_numeric, f.normalized_value_date, f.normalized_value_boolean,
		       f.unit, f.confidence_score, f.context_snippet, f.extracted_at
		FROM artifact_facts f
		JOIN artifacts a ON f.artifact_id = a.artifact_id
		WHERE a.program_id = $1 AND a.deleted_at IS NULL
		ORDER BY f.extracted_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, programID)
	if err != nil {
		return nil, fmt.Errorf("failed to get facts: %w", err)
	}
	defer rows.Close()

	facts := []Fact{}
	for rows.Next() {
		var f Fact
		err := rows.Scan(
			&f.FactID,
			&f.ArtifactID,
			&f.FactType,
			&f.FactKey,
			&f.FactValue,
			&f.NormalizedValueNumeric,
			&f.NormalizedValueDate,
			&f.NormalizedValueBoolean,
			&f.Unit,
			&f.ConfidenceScore,
			&f.ContextSnippet,
			&f.ExtractedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fact: %w", err)
		}
		facts = append(facts, f)
	}

	return facts, nil
}

// ============================================================================
// Context Cache Queries
// ============================================================================

// GetContextCacheEntry retrieves a context cache entry by artifact ID
func (r *Repository) GetContextCacheEntry(ctx context.Context, artifactID uuid.UUID) (*ContextCacheEntry, error) {
	query := `
		SELECT cache_id, artifact_id, program_id, context_data, token_count,
		       artifacts_included, cache_version, created_at, expires_at
		FROM artifact_context_cache
		WHERE artifact_id = $1
		  AND expires_at > NOW()
	`

	var entry ContextCacheEntry
	var artifactIDs pq.StringArray

	err := r.db.QueryRowContext(ctx, query, artifactID).Scan(
		&entry.CacheID,
		&entry.ArtifactID,
		&entry.ProgramID,
		&entry.ContextData,
		&entry.TokenCount,
		&artifactIDs,
		&entry.CacheVersion,
		&entry.CreatedAt,
		&entry.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("cache entry not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cache entry: %w", err)
	}

	// Convert string array to UUID array
	for _, idStr := range artifactIDs {
		id, err := uuid.Parse(idStr)
		if err == nil {
			entry.ArtifactsIncluded = append(entry.ArtifactsIncluded, id)
		}
	}

	return &entry, nil
}

// UpsertContextCacheEntry creates or updates a context cache entry
func (r *Repository) UpsertContextCacheEntry(ctx context.Context, entry *ContextCacheEntry) error {
	// Get program ID from artifact
	var programID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT program_id FROM artifacts WHERE artifact_id = $1`, entry.ArtifactID).Scan(&programID)
	if err != nil {
		return fmt.Errorf("failed to get program ID: %w", err)
	}

	query := `
		INSERT INTO artifact_context_cache (
			cache_id, artifact_id, program_id, context_data, token_count,
			artifacts_included, cache_version, created_at, expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (artifact_id) DO UPDATE
		SET
			context_data = $4,
			token_count = $5,
			artifacts_included = $6,
			cache_version = $7,
			created_at = $8,
			expires_at = $9
	`

	_, err = r.db.ExecContext(ctx, query,
		entry.CacheID,
		entry.ArtifactID,
		programID,
		entry.ContextData,
		entry.TokenCount,
		pq.Array(entry.ArtifactsIncluded),
		entry.CacheVersion,
		entry.CreatedAt,
		entry.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert cache entry: %w", err)
	}

	return nil
}

// DeleteContextCache deletes a context cache entry
func (r *Repository) DeleteContextCache(ctx context.Context, artifactID uuid.UUID) error {
	query := `DELETE FROM artifact_context_cache WHERE artifact_id = $1`
	_, err := r.db.ExecContext(ctx, query, artifactID)
	if err != nil {
		return fmt.Errorf("failed to delete cache entry: %w", err)
	}
	return nil
}

// CleanupExpiredContextCache removes expired cache entries
func (r *Repository) CleanupExpiredContextCache(ctx context.Context) (int, error) {
	query := `DELETE FROM artifact_context_cache WHERE expires_at < NOW()`
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup cache: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(count), nil
}

// GetArtifactIDsByProgram retrieves all artifact IDs for a program
func (r *Repository) GetArtifactIDsByProgram(ctx context.Context, programID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT artifact_id
		FROM artifacts
		WHERE program_id = $1 AND deleted_at IS NULL
	`

	rows, err := r.db.QueryContext(ctx, query, programID)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact IDs: %w", err)
	}
	defer rows.Close()

	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan ID: %w", err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// GetRecentArtifactsWithoutCache retrieves recent artifacts that don't have cache entries
func (r *Repository) GetRecentArtifactsWithoutCache(ctx context.Context, programID uuid.UUID, limit int) ([]Artifact, error) {
	query := `
		SELECT a.artifact_id, a.program_id, a.filename, a.storage_path, a.file_type,
		       a.file_size_bytes, a.mime_type, a.content_hash,
		       a.artifact_category, a.artifact_subcategory,
		       a.processing_status, a.processed_at, a.uploaded_by, a.uploaded_at
		FROM artifacts a
		LEFT JOIN artifact_context_cache c ON a.artifact_id = c.artifact_id AND c.expires_at > NOW()
		WHERE a.program_id = $1
		  AND a.deleted_at IS NULL
		  AND a.processing_status = 'completed'
		  AND c.artifact_id IS NULL
		ORDER BY a.uploaded_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, programID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifacts: %w", err)
	}
	defer rows.Close()

	artifacts := []Artifact{}
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

// RefreshContextSummaryView refreshes the materialized view
func (r *Repository) RefreshContextSummaryView(ctx context.Context) error {
	query := `REFRESH MATERIALIZED VIEW CONCURRENTLY artifact_context_summary`
	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to refresh materialized view: %w", err)
	}
	return nil
}

// GetContextCacheStats returns statistics about the context cache
func (r *Repository) GetContextCacheStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total entries
	var totalEntries int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM artifact_context_cache WHERE expires_at > NOW()`).Scan(&totalEntries)
	if err == nil {
		stats["total_entries"] = totalEntries
	}

	// Average token count
	var avgTokens float64
	err = r.db.QueryRowContext(ctx, `SELECT AVG(token_count) FROM artifact_context_cache WHERE expires_at > NOW()`).Scan(&avgTokens)
	if err == nil {
		stats["avg_tokens"] = avgTokens
	}

	// Expired entries
	var expiredEntries int
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM artifact_context_cache WHERE expires_at <= NOW()`).Scan(&expiredEntries)
	if err == nil {
		stats["expired_entries"] = expiredEntries
	}

	return stats, nil
}

// ============================================================================
// Entity Graph Statistics
// ============================================================================

// CountUniquePeopleInProgram counts unique people in a program
func (r *Repository) CountUniquePeopleInProgram(ctx context.Context, programID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(DISTINCT ap.person_id)
		FROM artifact_persons ap
		JOIN artifacts a ON ap.artifact_id = a.artifact_id
		WHERE a.program_id = $1 AND a.deleted_at IS NULL
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, programID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count people: %w", err)
	}

	return count, nil
}

// CountEntityGraphEdges counts entity graph edges for a program
func (r *Repository) CountEntityGraphEdges(ctx context.Context, programID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM artifact_entity_graph WHERE program_id = $1`

	var count int
	err := r.db.QueryRowContext(ctx, query, programID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count edges: %w", err)
	}

	return count, nil
}

// GetAverageCoOccurrences calculates average co-occurrences in a program
func (r *Repository) GetAverageCoOccurrences(ctx context.Context, programID uuid.UUID) (float64, error) {
	query := `SELECT AVG(co_occurrence_count) FROM artifact_entity_graph WHERE program_id = $1`

	var avg sql.NullFloat64
	err := r.db.QueryRowContext(ctx, query, programID).Scan(&avg)
	if err != nil {
		return 0, fmt.Errorf("failed to get average: %w", err)
	}

	if !avg.Valid {
		return 0, nil
	}

	return avg.Float64, nil
}

// GetMostConnectedPerson finds the person with most relationships
func (r *Repository) GetMostConnectedPerson(ctx context.Context, programID uuid.UUID) (*Person, error) {
	query := `
		SELECT p.person_id, p.artifact_id, p.person_name, p.person_role, p.person_organization,
		       COUNT(*) as relationship_count
		FROM artifact_entity_graph eg
		JOIN artifact_persons p ON (p.person_id = eg.person_id_1 OR p.person_id = eg.person_id_2)
		WHERE eg.program_id = $1
		GROUP BY p.person_id, p.artifact_id, p.person_name, p.person_role, p.person_organization
		ORDER BY relationship_count DESC
		LIMIT 1
	`

	var person Person
	var relationshipCount int
	err := r.db.QueryRowContext(ctx, query, programID).Scan(
		&person.PersonID,
		&person.ArtifactID,
		&person.PersonName,
		&person.PersonRole,
		&person.PersonOrganization,
		&relationshipCount,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get most connected person: %w", err)
	}

	// Use mention count field to store relationship count
	person.MentionCount = relationshipCount

	return &person, nil
}

// GetKeyPeopleByProgram retrieves the most mentioned people in a program
func (r *Repository) GetKeyPeopleByProgram(ctx context.Context, programID uuid.UUID, limit int) ([]PersonContext, error) {
	query := `
		SELECT ap.person_id, ap.person_name, ap.person_role, ap.person_organization,
		       SUM(ap.mention_count) as total_mentions,
		       COUNT(DISTINCT ap.artifact_id) as artifact_count
		FROM artifact_persons ap
		JOIN artifacts a ON ap.artifact_id = a.artifact_id
		WHERE a.program_id = $1 AND a.deleted_at IS NULL
		GROUP BY ap.person_id, ap.person_name, ap.person_role, ap.person_organization
		ORDER BY total_mentions DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, programID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get key people: %w", err)
	}
	defer rows.Close()

	people := []PersonContext{}
	for rows.Next() {
		var person PersonContext
		var role, org sql.NullString

		err := rows.Scan(
			&person.PersonID,
			&person.Name,
			&role,
			&org,
			&person.MentionCount,
			&person.ArtifactCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan person: %w", err)
		}

		if role.Valid {
			person.Role = role.String
		}
		if org.Valid {
			person.Organization = org.String
		}

		people = append(people, person)
	}

	return people, nil
}

// FindPeopleByNamePattern finds people matching a name pattern
func (r *Repository) FindPeopleByNamePattern(ctx context.Context, programID uuid.UUID, namePattern string) ([]Person, error) {
	query := `
		SELECT DISTINCT ON (ap.person_name)
		       ap.person_id, ap.artifact_id, ap.person_name, ap.person_role, ap.person_organization,
		       ap.mention_count, ap.context_snippets, ap.confidence_score, ap.stakeholder_id, ap.extracted_at
		FROM artifact_persons ap
		JOIN artifacts a ON ap.artifact_id = a.artifact_id
		WHERE a.program_id = $1
		  AND a.deleted_at IS NULL
		  AND ap.person_name ILIKE $2
		ORDER BY ap.person_name, ap.mention_count DESC
		LIMIT 50
	`

	rows, err := r.db.QueryContext(ctx, query, programID, "%"+namePattern+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to find people: %w", err)
	}
	defer rows.Close()

	people := []Person{}
	for rows.Next() {
		var p Person
		err := rows.Scan(
			&p.PersonID,
			&p.ArtifactID,
			&p.PersonName,
			&p.PersonRole,
			&p.PersonOrganization,
			&p.MentionCount,
			&p.ContextSnippets,
			&p.ConfidenceScore,
			&p.StakeholderID,
			&p.ExtractedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan person: %w", err)
		}
		people = append(people, p)
	}

	return people, nil
}

// GetProgramEntityGraph retrieves the full entity graph for a program
func (r *Repository) GetProgramEntityGraph(ctx context.Context, programID uuid.UUID, minCoOccurrences int) ([]PersonRelationship, error) {
	query := `
		SELECT
			eg.person_id_1,
			p1.person_name as person1_name,
			eg.person_id_2,
			p2.person_name as person2_name,
			eg.co_occurrence_count,
			eg.shared_artifact_ids,
			eg.relationship_strength
		FROM artifact_entity_graph eg
		JOIN artifact_persons p1 ON eg.person_id_1 = p1.person_id
		JOIN artifact_persons p2 ON eg.person_id_2 = p2.person_id
		WHERE eg.program_id = $1
		  AND eg.co_occurrence_count >= $2
		ORDER BY eg.relationship_strength DESC
	`

	rows, err := r.db.QueryContext(ctx, query, programID, minCoOccurrences)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity graph: %w", err)
	}
	defer rows.Close()

	relationships := []PersonRelationship{}
	for rows.Next() {
		var rel PersonRelationship
		var sharedArtifactIDs pq.StringArray

		err := rows.Scan(
			&rel.Person1ID,
			&rel.Person1Name,
			&rel.Person2ID,
			&rel.Person2Name,
			&rel.CoOccurrences,
			&sharedArtifactIDs,
			&rel.Strength,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan relationship: %w", err)
		}

		// Convert string array to UUID array
		for _, idStr := range sharedArtifactIDs {
			id, err := uuid.Parse(idStr)
			if err == nil {
				rel.SharedArtifacts = append(rel.SharedArtifacts, id)
			}
		}

		relationships = append(relationships, rel)
	}

	return relationships, nil
}

// FindArtifactsWithEntityOverlap finds artifacts with entity overlap (used by entity graph)
func (r *Repository) FindArtifactsWithEntityOverlap(
	ctx context.Context,
	targetArtifactID uuid.UUID,
	programID uuid.UUID,
	personIDs []uuid.UUID,
	limit int,
) ([]ArtifactWithEntityOverlap, error) {
	// This is similar to FindArtifactsWithSharedEntities but returns a different type
	// For now, delegate to that method and convert
	candidates, err := r.FindArtifactsWithSharedEntities(ctx, targetArtifactID, programID, personIDs, limit)
	if err != nil {
		return nil, err
	}

	results := []ArtifactWithEntityOverlap{}
	for _, candidate := range candidates {
		artifact := &Artifact{
			ArtifactID:   candidate.ArtifactID,
			Filename:     candidate.Filename,
			UploadedAt:   candidate.UploadedAt,
		}

		overlap := ArtifactWithEntityOverlap{
			Artifact:          artifact,
			SharedPersonIDs:   candidate.SharedPersonIDs,
			SharedPersonNames: candidate.MentionedPeople,
			OverlapCount:      len(candidate.SharedPersonIDs),
		}

		// Calculate overlap ratio
		if len(personIDs) > 0 {
			overlap.OverlapRatio = float64(len(candidate.SharedPersonIDs)) / float64(len(personIDs))
		}

		results = append(results, overlap)
	}

	return results, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// getPersonDataForArtifact is a helper to get person IDs and names
func (r *Repository) getPersonDataForArtifact(ctx context.Context, artifactID uuid.UUID) ([]uuid.UUID, []string, error) {
	query := `
		SELECT person_id, person_name
		FROM artifact_persons
		WHERE artifact_id = $1
		ORDER BY mention_count DESC
	`

	rows, err := r.db.QueryContext(ctx, query, artifactID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	personIDs := []uuid.UUID{}
	personNames := []string{}

	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		personIDs = append(personIDs, id)
		personNames = append(personNames, name)
	}

	return personIDs, personNames, nil
}

// getTopicDataForArtifact is a helper to get topic names and IDs
func (r *Repository) getTopicDataForArtifact(ctx context.Context, artifactID uuid.UUID) ([]string, []uuid.UUID, error) {
	query := `
		SELECT topic_id, topic_name
		FROM artifact_topics
		WHERE artifact_id = $1
		ORDER BY confidence_score DESC
	`

	rows, err := r.db.QueryContext(ctx, query, artifactID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	topicNames := []string{}
	topicIDs := []uuid.UUID{}

	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		topicIDs = append(topicIDs, id)
		topicNames = append(topicNames, name)
	}

	return topicNames, topicIDs, nil
}

// getFactCountForArtifact is a helper to get fact count
func (r *Repository) getFactCountForArtifact(ctx context.Context, artifactID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM artifact_facts WHERE artifact_id = $1`
	err := r.db.QueryRowContext(ctx, query, artifactID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
