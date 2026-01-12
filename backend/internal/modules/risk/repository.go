package risk

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cerberus/backend/internal/platform/db"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// RepositoryInterface defines methods for risk data access
type RepositoryInterface interface {
	// Risk CRUD
	CreateRisk(ctx context.Context, risk *Risk) error
	GetRiskByID(ctx context.Context, riskID uuid.UUID) (*Risk, error)
	ListRisks(ctx context.Context, filter RiskFilterRequest) ([]Risk, error)
	UpdateRisk(ctx context.Context, risk *Risk) error
	DeleteRisk(ctx context.Context, riskID uuid.UUID) error

	// Risk Suggestions
	CreateSuggestion(ctx context.Context, suggestion *RiskSuggestion) error
	GetSuggestionByID(ctx context.Context, suggestionID uuid.UUID) (*RiskSuggestion, error)
	ListSuggestions(ctx context.Context, programID uuid.UUID, includeProcessed bool) ([]RiskSuggestion, error)
	UpdateSuggestion(ctx context.Context, suggestion *RiskSuggestion) error

	// Risk Mitigations
	CreateMitigation(ctx context.Context, mitigation *RiskMitigation) error
	GetMitigationByID(ctx context.Context, mitigationID uuid.UUID) (*RiskMitigation, error)
	ListMitigationsByRisk(ctx context.Context, riskID uuid.UUID) ([]RiskMitigation, error)
	UpdateMitigation(ctx context.Context, mitigation *RiskMitigation) error
	DeleteMitigation(ctx context.Context, mitigationID uuid.UUID) error

	// Artifact Linking
	LinkArtifact(ctx context.Context, link *RiskArtifactLink) error
	UnlinkArtifact(ctx context.Context, linkID uuid.UUID) error
	GetLinkedArtifacts(ctx context.Context, riskID uuid.UUID) ([]RiskArtifactLink, error)
	GetRisksByArtifact(ctx context.Context, artifactID uuid.UUID) ([]Risk, error)

	// Conversation Threads
	CreateThread(ctx context.Context, thread *ConversationThread) error
	GetThreadByID(ctx context.Context, threadID uuid.UUID) (*ConversationThread, error)
	ListThreadsByRisk(ctx context.Context, riskID uuid.UUID) ([]ConversationThread, error)
	ResolveThread(ctx context.Context, threadID, resolvedBy uuid.UUID) error
	DeleteThread(ctx context.Context, threadID uuid.UUID) error

	// Conversation Messages
	CreateMessage(ctx context.Context, message *ConversationMessage) error
	GetThreadMessages(ctx context.Context, threadID uuid.UUID) ([]ConversationMessage, error)
	DeleteMessage(ctx context.Context, messageID uuid.UUID) error

	// Composed Queries
	GetRiskWithContext(ctx context.Context, riskID uuid.UUID) (*RiskWithMetadata, error)
	GetThreadWithMessages(ctx context.Context, threadID uuid.UUID) (*ThreadWithMessages, error)

	// Direct DB access
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

// Repository handles database operations for risk module
type Repository struct {
	db *db.DB
}

// NewRepository creates a new risk repository
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

// CreateRisk inserts a new risk
func (r *Repository) CreateRisk(ctx context.Context, risk *Risk) error {
	query := `
		INSERT INTO risks (
			risk_id, program_id, title, description, probability, impact,
			category, status, owner_user_id, owner_name, identified_date,
			target_resolution_date, ai_confidence_score, ai_detected_at,
			created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	_, err := r.db.ExecContext(ctx, query,
		risk.RiskID,
		risk.ProgramID,
		risk.Title,
		risk.Description,
		risk.Probability,
		risk.Impact,
		risk.Category,
		risk.Status,
		risk.OwnerUserID,
		risk.OwnerName,
		risk.IdentifiedDate,
		risk.TargetResolutionDate,
		risk.AIConfidenceScore,
		risk.AIDetectedAt,
		risk.CreatedBy,
		risk.CreatedAt,
		risk.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create risk: %w", err)
	}

	return nil
}

// GetRiskByID retrieves a risk by ID
func (r *Repository) GetRiskByID(ctx context.Context, riskID uuid.UUID) (*Risk, error) {
	query := `
		SELECT risk_id, program_id, title, description, probability, impact,
		       severity, category, status, owner_user_id, owner_name,
		       identified_date, target_resolution_date, closed_date, realized_date,
		       ai_confidence_score, ai_detected_at, created_by, created_at,
		       updated_at, deleted_at
		FROM risks
		WHERE risk_id = $1 AND deleted_at IS NULL
	`

	var risk Risk
	err := r.db.QueryRowContext(ctx, query, riskID).Scan(
		&risk.RiskID,
		&risk.ProgramID,
		&risk.Title,
		&risk.Description,
		&risk.Probability,
		&risk.Impact,
		&risk.Severity,
		&risk.Category,
		&risk.Status,
		&risk.OwnerUserID,
		&risk.OwnerName,
		&risk.IdentifiedDate,
		&risk.TargetResolutionDate,
		&risk.ClosedDate,
		&risk.RealizedDate,
		&risk.AIConfidenceScore,
		&risk.AIDetectedAt,
		&risk.CreatedBy,
		&risk.CreatedAt,
		&risk.UpdatedAt,
		&risk.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("risk not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get risk: %w", err)
	}

	return &risk, nil
}

// ListRisks retrieves risks with optional filters
func (r *Repository) ListRisks(ctx context.Context, filter RiskFilterRequest) ([]Risk, error) {
	query := `
		SELECT risk_id, program_id, title, description, probability, impact,
		       severity, category, status, owner_user_id, owner_name,
		       identified_date, target_resolution_date, created_by, created_at, updated_at
		FROM risks
		WHERE program_id = $1 AND deleted_at IS NULL
	`

	args := []interface{}{filter.ProgramID}
	argCount := 1

	// Add filters
	if filter.Status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
	}

	if filter.Category != "" {
		argCount++
		query += fmt.Sprintf(" AND category = $%d", argCount)
		args = append(args, filter.Category)
	}

	if filter.Severity != "" {
		argCount++
		query += fmt.Sprintf(" AND severity = $%d", argCount)
		args = append(args, filter.Severity)
	}

	if filter.OwnerUserID != nil {
		argCount++
		query += fmt.Sprintf(" AND owner_user_id = $%d", argCount)
		args = append(args, filter.OwnerUserID)
	}

	query += " ORDER BY severity DESC, created_at DESC"

	// Add pagination
	if filter.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list risks: %w", err)
	}
	defer rows.Close()

	var risks []Risk
	for rows.Next() {
		var risk Risk
		err := rows.Scan(
			&risk.RiskID,
			&risk.ProgramID,
			&risk.Title,
			&risk.Description,
			&risk.Probability,
			&risk.Impact,
			&risk.Severity,
			&risk.Category,
			&risk.Status,
			&risk.OwnerUserID,
			&risk.OwnerName,
			&risk.IdentifiedDate,
			&risk.TargetResolutionDate,
			&risk.CreatedBy,
			&risk.CreatedAt,
			&risk.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan risk: %w", err)
		}
		risks = append(risks, risk)
	}

	return risks, nil
}

// UpdateRisk updates an existing risk
func (r *Repository) UpdateRisk(ctx context.Context, risk *Risk) error {
	query := `
		UPDATE risks
		SET title = $1, description = $2, probability = $3, impact = $4,
		    category = $5, status = $6, owner_user_id = $7, owner_name = $8,
		    target_resolution_date = $9, closed_date = $10, realized_date = $11
		WHERE risk_id = $12 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query,
		risk.Title,
		risk.Description,
		risk.Probability,
		risk.Impact,
		risk.Category,
		risk.Status,
		risk.OwnerUserID,
		risk.OwnerName,
		risk.TargetResolutionDate,
		risk.ClosedDate,
		risk.RealizedDate,
		risk.RiskID,
	)

	if err != nil {
		return fmt.Errorf("failed to update risk: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("risk not found or already deleted")
	}

	return nil
}

// DeleteRisk soft-deletes a risk
func (r *Repository) DeleteRisk(ctx context.Context, riskID uuid.UUID) error {
	query := `
		UPDATE risks
		SET deleted_at = NOW()
		WHERE risk_id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, riskID)
	if err != nil {
		return fmt.Errorf("failed to delete risk: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("risk not found or already deleted")
	}

	return nil
}

// CreateSuggestion inserts a new risk suggestion
func (r *Repository) CreateSuggestion(ctx context.Context, suggestion *RiskSuggestion) error {
	query := `
		INSERT INTO risk_suggestions (
			suggestion_id, program_id, title, description, rationale,
			suggested_probability, suggested_impact, suggested_category,
			source_type, source_artifact_ids, source_insight_id, source_variance_id,
			ai_confidence_score, ai_detected_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.db.ExecContext(ctx, query,
		suggestion.SuggestionID,
		suggestion.ProgramID,
		suggestion.Title,
		suggestion.Description,
		suggestion.Rationale,
		suggestion.SuggestedProbability,
		suggestion.SuggestedImpact,
		suggestion.SuggestedCategory,
		suggestion.SourceType,
		pq.Array(suggestion.SourceArtifactIDs),
		suggestion.SourceInsightID,
		suggestion.SourceVarianceID,
		suggestion.AIConfidenceScore,
		suggestion.AIDetectedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create risk suggestion: %w", err)
	}

	return nil
}

// GetSuggestionByID retrieves a risk suggestion by ID
func (r *Repository) GetSuggestionByID(ctx context.Context, suggestionID uuid.UUID) (*RiskSuggestion, error) {
	query := `
		SELECT suggestion_id, program_id, title, description, rationale,
		       suggested_probability, suggested_impact, suggested_severity, suggested_category,
		       source_type, source_artifact_ids, source_insight_id, source_variance_id,
		       ai_confidence_score, ai_detected_at, is_approved, is_dismissed,
		       approved_by, approved_at, dismissed_by, dismissed_at, dismissal_reason, created_risk_id
		FROM risk_suggestions
		WHERE suggestion_id = $1
	`

	var suggestion RiskSuggestion
	err := r.db.QueryRowContext(ctx, query, suggestionID).Scan(
		&suggestion.SuggestionID,
		&suggestion.ProgramID,
		&suggestion.Title,
		&suggestion.Description,
		&suggestion.Rationale,
		&suggestion.SuggestedProbability,
		&suggestion.SuggestedImpact,
		&suggestion.SuggestedSeverity,
		&suggestion.SuggestedCategory,
		&suggestion.SourceType,
		pq.Array(&suggestion.SourceArtifactIDs),
		&suggestion.SourceInsightID,
		&suggestion.SourceVarianceID,
		&suggestion.AIConfidenceScore,
		&suggestion.AIDetectedAt,
		&suggestion.IsApproved,
		&suggestion.IsDismissed,
		&suggestion.ApprovedBy,
		&suggestion.ApprovedAt,
		&suggestion.DismissedBy,
		&suggestion.DismissedAt,
		&suggestion.DismissalReason,
		&suggestion.CreatedRiskID,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("risk suggestion not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get risk suggestion: %w", err)
	}

	return &suggestion, nil
}

// ListSuggestions retrieves risk suggestions for a program
func (r *Repository) ListSuggestions(ctx context.Context, programID uuid.UUID, includeProcessed bool) ([]RiskSuggestion, error) {
	query := `
		SELECT suggestion_id, program_id, title, description, rationale,
		       suggested_probability, suggested_impact, suggested_severity, suggested_category,
		       source_type, source_artifact_ids, ai_confidence_score, ai_detected_at,
		       is_approved, is_dismissed, created_risk_id
		FROM risk_suggestions
		WHERE program_id = $1
	`

	if !includeProcessed {
		query += " AND is_approved = FALSE AND is_dismissed = FALSE"
	}

	query += " ORDER BY suggested_severity DESC, ai_detected_at DESC"

	rows, err := r.db.QueryContext(ctx, query, programID)
	if err != nil {
		return nil, fmt.Errorf("failed to list risk suggestions: %w", err)
	}
	defer rows.Close()

	var suggestions []RiskSuggestion
	for rows.Next() {
		var suggestion RiskSuggestion
		err := rows.Scan(
			&suggestion.SuggestionID,
			&suggestion.ProgramID,
			&suggestion.Title,
			&suggestion.Description,
			&suggestion.Rationale,
			&suggestion.SuggestedProbability,
			&suggestion.SuggestedImpact,
			&suggestion.SuggestedSeverity,
			&suggestion.SuggestedCategory,
			&suggestion.SourceType,
			pq.Array(&suggestion.SourceArtifactIDs),
			&suggestion.AIConfidenceScore,
			&suggestion.AIDetectedAt,
			&suggestion.IsApproved,
			&suggestion.IsDismissed,
			&suggestion.CreatedRiskID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan risk suggestion: %w", err)
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// UpdateSuggestion updates a risk suggestion
func (r *Repository) UpdateSuggestion(ctx context.Context, suggestion *RiskSuggestion) error {
	query := `
		UPDATE risk_suggestions
		SET is_approved = $1, is_dismissed = $2, approved_by = $3, approved_at = $4,
		    dismissed_by = $5, dismissed_at = $6, dismissal_reason = $7, created_risk_id = $8
		WHERE suggestion_id = $9
	`

	result, err := r.db.ExecContext(ctx, query,
		suggestion.IsApproved,
		suggestion.IsDismissed,
		suggestion.ApprovedBy,
		suggestion.ApprovedAt,
		suggestion.DismissedBy,
		suggestion.DismissedAt,
		suggestion.DismissalReason,
		suggestion.CreatedRiskID,
		suggestion.SuggestionID,
	)

	if err != nil {
		return fmt.Errorf("failed to update risk suggestion: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("risk suggestion not found")
	}

	return nil
}

// CreateMitigation inserts a new mitigation
func (r *Repository) CreateMitigation(ctx context.Context, mitigation *RiskMitigation) error {
	query := `
		INSERT INTO risk_mitigations (
			mitigation_id, risk_id, strategy, action_description,
			expected_probability_reduction, expected_impact_reduction,
			status, assigned_to, target_completion_date,
			estimated_cost, currency, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.db.ExecContext(ctx, query,
		mitigation.MitigationID,
		mitigation.RiskID,
		mitigation.Strategy,
		mitigation.ActionDescription,
		mitigation.ExpectedProbabilityReduction,
		mitigation.ExpectedImpactReduction,
		mitigation.Status,
		mitigation.AssignedTo,
		mitigation.TargetCompletionDate,
		mitigation.EstimatedCost,
		mitigation.Currency,
		mitigation.CreatedBy,
		mitigation.CreatedAt,
		mitigation.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create mitigation: %w", err)
	}

	return nil
}

// GetMitigationByID retrieves a mitigation by ID
func (r *Repository) GetMitigationByID(ctx context.Context, mitigationID uuid.UUID) (*RiskMitigation, error) {
	query := `
		SELECT mitigation_id, risk_id, strategy, action_description,
		       expected_probability_reduction, expected_impact_reduction,
		       effectiveness_rating, status, assigned_to, target_completion_date,
		       actual_completion_date, estimated_cost, actual_cost, currency,
		       created_by, created_at, updated_at, deleted_at
		FROM risk_mitigations
		WHERE mitigation_id = $1 AND deleted_at IS NULL
	`

	var mitigation RiskMitigation
	err := r.db.QueryRowContext(ctx, query, mitigationID).Scan(
		&mitigation.MitigationID,
		&mitigation.RiskID,
		&mitigation.Strategy,
		&mitigation.ActionDescription,
		&mitigation.ExpectedProbabilityReduction,
		&mitigation.ExpectedImpactReduction,
		&mitigation.EffectivenessRating,
		&mitigation.Status,
		&mitigation.AssignedTo,
		&mitigation.TargetCompletionDate,
		&mitigation.ActualCompletionDate,
		&mitigation.EstimatedCost,
		&mitigation.ActualCost,
		&mitigation.Currency,
		&mitigation.CreatedBy,
		&mitigation.CreatedAt,
		&mitigation.UpdatedAt,
		&mitigation.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("mitigation not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get mitigation: %w", err)
	}

	return &mitigation, nil
}

// ListMitigationsByRisk retrieves all mitigations for a risk
func (r *Repository) ListMitigationsByRisk(ctx context.Context, riskID uuid.UUID) ([]RiskMitigation, error) {
	query := `
		SELECT mitigation_id, risk_id, strategy, action_description,
		       expected_probability_reduction, expected_impact_reduction,
		       effectiveness_rating, status, assigned_to, target_completion_date,
		       actual_completion_date, estimated_cost, actual_cost, currency,
		       created_by, created_at, updated_at
		FROM risk_mitigations
		WHERE risk_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, riskID)
	if err != nil {
		return nil, fmt.Errorf("failed to list mitigations: %w", err)
	}
	defer rows.Close()

	var mitigations []RiskMitigation
	for rows.Next() {
		var mitigation RiskMitigation
		err := rows.Scan(
			&mitigation.MitigationID,
			&mitigation.RiskID,
			&mitigation.Strategy,
			&mitigation.ActionDescription,
			&mitigation.ExpectedProbabilityReduction,
			&mitigation.ExpectedImpactReduction,
			&mitigation.EffectivenessRating,
			&mitigation.Status,
			&mitigation.AssignedTo,
			&mitigation.TargetCompletionDate,
			&mitigation.ActualCompletionDate,
			&mitigation.EstimatedCost,
			&mitigation.ActualCost,
			&mitigation.Currency,
			&mitigation.CreatedBy,
			&mitigation.CreatedAt,
			&mitigation.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mitigation: %w", err)
		}
		mitigations = append(mitigations, mitigation)
	}

	return mitigations, nil
}

// UpdateMitigation updates an existing mitigation
func (r *Repository) UpdateMitigation(ctx context.Context, mitigation *RiskMitigation) error {
	query := `
		UPDATE risk_mitigations
		SET status = $1, effectiveness_rating = $2, actual_completion_date = $3, actual_cost = $4
		WHERE mitigation_id = $5 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query,
		mitigation.Status,
		mitigation.EffectivenessRating,
		mitigation.ActualCompletionDate,
		mitigation.ActualCost,
		mitigation.MitigationID,
	)

	if err != nil {
		return fmt.Errorf("failed to update mitigation: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("mitigation not found or already deleted")
	}

	return nil
}

// DeleteMitigation soft-deletes a mitigation
func (r *Repository) DeleteMitigation(ctx context.Context, mitigationID uuid.UUID) error {
	query := `
		UPDATE risk_mitigations
		SET deleted_at = NOW()
		WHERE mitigation_id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, mitigationID)
	if err != nil {
		return fmt.Errorf("failed to delete mitigation: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("mitigation not found or already deleted")
	}

	return nil
}

// LinkArtifact creates a link between a risk and an artifact
func (r *Repository) LinkArtifact(ctx context.Context, link *RiskArtifactLink) error {
	query := `
		INSERT INTO risk_artifact_links (
			link_id, risk_id, artifact_id, link_type, description, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (risk_id, artifact_id, link_type) DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query,
		link.LinkID,
		link.RiskID,
		link.ArtifactID,
		link.LinkType,
		link.Description,
		link.CreatedBy,
		link.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to link artifact: %w", err)
	}

	return nil
}

// UnlinkArtifact removes a link between a risk and an artifact
func (r *Repository) UnlinkArtifact(ctx context.Context, linkID uuid.UUID) error {
	query := `DELETE FROM risk_artifact_links WHERE link_id = $1`

	result, err := r.db.ExecContext(ctx, query, linkID)
	if err != nil {
		return fmt.Errorf("failed to unlink artifact: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("link not found")
	}

	return nil
}

// GetLinkedArtifacts retrieves all artifacts linked to a risk
func (r *Repository) GetLinkedArtifacts(ctx context.Context, riskID uuid.UUID) ([]RiskArtifactLink, error) {
	query := `
		SELECT link_id, risk_id, artifact_id, link_type, description, created_by, created_at
		FROM risk_artifact_links
		WHERE risk_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, riskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get linked artifacts: %w", err)
	}
	defer rows.Close()

	var links []RiskArtifactLink
	for rows.Next() {
		var link RiskArtifactLink
		err := rows.Scan(
			&link.LinkID,
			&link.RiskID,
			&link.ArtifactID,
			&link.LinkType,
			&link.Description,
			&link.CreatedBy,
			&link.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan link: %w", err)
		}
		links = append(links, link)
	}

	return links, nil
}

// GetRisksByArtifact retrieves all risks linked to an artifact
func (r *Repository) GetRisksByArtifact(ctx context.Context, artifactID uuid.UUID) ([]Risk, error) {
	query := `
		SELECT r.risk_id, r.program_id, r.title, r.description, r.probability,
		       r.impact, r.severity, r.category, r.status, r.owner_user_id,
		       r.owner_name, r.identified_date, r.target_resolution_date,
		       r.created_by, r.created_at, r.updated_at
		FROM risks r
		INNER JOIN risk_artifact_links ral ON r.risk_id = ral.risk_id
		WHERE ral.artifact_id = $1 AND r.deleted_at IS NULL
		ORDER BY r.severity DESC, r.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get risks by artifact: %w", err)
	}
	defer rows.Close()

	var risks []Risk
	for rows.Next() {
		var risk Risk
		err := rows.Scan(
			&risk.RiskID,
			&risk.ProgramID,
			&risk.Title,
			&risk.Description,
			&risk.Probability,
			&risk.Impact,
			&risk.Severity,
			&risk.Category,
			&risk.Status,
			&risk.OwnerUserID,
			&risk.OwnerName,
			&risk.IdentifiedDate,
			&risk.TargetResolutionDate,
			&risk.CreatedBy,
			&risk.CreatedAt,
			&risk.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan risk: %w", err)
		}
		risks = append(risks, risk)
	}

	return risks, nil
}

// CreateThread creates a new conversation thread
func (r *Repository) CreateThread(ctx context.Context, thread *ConversationThread) error {
	query := `
		INSERT INTO conversation_threads (
			thread_id, risk_id, title, thread_type, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		thread.ThreadID,
		thread.RiskID,
		thread.Title,
		thread.ThreadType,
		thread.CreatedBy,
		thread.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create thread: %w", err)
	}

	return nil
}

// GetThreadByID retrieves a thread by ID
func (r *Repository) GetThreadByID(ctx context.Context, threadID uuid.UUID) (*ConversationThread, error) {
	query := `
		SELECT thread_id, risk_id, title, thread_type, is_resolved, resolved_at,
		       resolved_by, message_count, last_message_at, created_by, created_at, deleted_at
		FROM conversation_threads
		WHERE thread_id = $1 AND deleted_at IS NULL
	`

	var thread ConversationThread
	err := r.db.QueryRowContext(ctx, query, threadID).Scan(
		&thread.ThreadID,
		&thread.RiskID,
		&thread.Title,
		&thread.ThreadType,
		&thread.IsResolved,
		&thread.ResolvedAt,
		&thread.ResolvedBy,
		&thread.MessageCount,
		&thread.LastMessageAt,
		&thread.CreatedBy,
		&thread.CreatedAt,
		&thread.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("thread not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	return &thread, nil
}

// ListThreadsByRisk retrieves all threads for a risk
func (r *Repository) ListThreadsByRisk(ctx context.Context, riskID uuid.UUID) ([]ConversationThread, error) {
	query := `
		SELECT thread_id, risk_id, title, thread_type, is_resolved, resolved_at,
		       resolved_by, message_count, last_message_at, created_by, created_at
		FROM conversation_threads
		WHERE risk_id = $1 AND deleted_at IS NULL
		ORDER BY is_resolved ASC, last_message_at DESC NULLS LAST, created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, riskID)
	if err != nil {
		return nil, fmt.Errorf("failed to list threads: %w", err)
	}
	defer rows.Close()

	var threads []ConversationThread
	for rows.Next() {
		var thread ConversationThread
		err := rows.Scan(
			&thread.ThreadID,
			&thread.RiskID,
			&thread.Title,
			&thread.ThreadType,
			&thread.IsResolved,
			&thread.ResolvedAt,
			&thread.ResolvedBy,
			&thread.MessageCount,
			&thread.LastMessageAt,
			&thread.CreatedBy,
			&thread.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan thread: %w", err)
		}
		threads = append(threads, thread)
	}

	return threads, nil
}

// ResolveThread marks a thread as resolved
func (r *Repository) ResolveThread(ctx context.Context, threadID, resolvedBy uuid.UUID) error {
	query := `
		UPDATE conversation_threads
		SET is_resolved = TRUE, resolved_at = NOW(), resolved_by = $1
		WHERE thread_id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, resolvedBy, threadID)
	if err != nil {
		return fmt.Errorf("failed to resolve thread: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("thread not found or already deleted")
	}

	return nil
}

// DeleteThread soft-deletes a thread
func (r *Repository) DeleteThread(ctx context.Context, threadID uuid.UUID) error {
	query := `
		UPDATE conversation_threads
		SET deleted_at = NOW()
		WHERE thread_id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, threadID)
	if err != nil {
		return fmt.Errorf("failed to delete thread: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("thread not found or already deleted")
	}

	return nil
}

// CreateMessage creates a new message in a thread
func (r *Repository) CreateMessage(ctx context.Context, message *ConversationMessage) error {
	query := `
		INSERT INTO conversation_messages (
			message_id, thread_id, message_text, message_format,
			mentioned_user_ids, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		message.MessageID,
		message.ThreadID,
		message.MessageText,
		message.MessageFormat,
		pq.Array(message.MentionedUserIDs),
		message.CreatedBy,
		message.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	return nil
}

// GetThreadMessages retrieves all messages for a thread
func (r *Repository) GetThreadMessages(ctx context.Context, threadID uuid.UUID) ([]ConversationMessage, error) {
	query := `
		SELECT message_id, thread_id, message_text, message_format,
		       mentioned_user_ids, created_by, created_at, edited_at, deleted_at
		FROM conversation_messages
		WHERE thread_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []ConversationMessage
	for rows.Next() {
		var message ConversationMessage
		err := rows.Scan(
			&message.MessageID,
			&message.ThreadID,
			&message.MessageText,
			&message.MessageFormat,
			pq.Array(&message.MentionedUserIDs),
			&message.CreatedBy,
			&message.CreatedAt,
			&message.EditedAt,
			&message.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, message)
	}

	return messages, nil
}

// DeleteMessage soft-deletes a message
func (r *Repository) DeleteMessage(ctx context.Context, messageID uuid.UUID) error {
	query := `
		UPDATE conversation_messages
		SET deleted_at = NOW()
		WHERE message_id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, messageID)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("message not found or already deleted")
	}

	return nil
}

// GetRiskWithContext retrieves a risk with all its related entities
func (r *Repository) GetRiskWithContext(ctx context.Context, riskID uuid.UUID) (*RiskWithMetadata, error) {
	// Get risk
	risk, err := r.GetRiskByID(ctx, riskID)
	if err != nil {
		return nil, err
	}

	result := &RiskWithMetadata{
		Risk: *risk,
	}

	// Get mitigations
	mitigations, err := r.ListMitigationsByRisk(ctx, riskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get mitigations: %w", err)
	}
	result.Mitigations = mitigations

	// Get linked artifacts
	links, err := r.GetLinkedArtifacts(ctx, riskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get linked artifacts: %w", err)
	}
	result.LinkedArtifacts = links

	// Get threads
	threads, err := r.ListThreadsByRisk(ctx, riskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get threads: %w", err)
	}
	result.Threads = threads

	return result, nil
}

// GetThreadWithMessages retrieves a thread with all its messages
func (r *Repository) GetThreadWithMessages(ctx context.Context, threadID uuid.UUID) (*ThreadWithMessages, error) {
	// Get thread
	thread, err := r.GetThreadByID(ctx, threadID)
	if err != nil {
		return nil, err
	}

	result := &ThreadWithMessages{
		ConversationThread: *thread,
	}

	// Get messages
	messages, err := r.GetThreadMessages(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	result.Messages = messages

	return result, nil
}
