package risk

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Risk represents a risk in the risk register
type Risk struct {
	RiskID                uuid.UUID      `json:"risk_id"`
	ProgramID             uuid.UUID      `json:"program_id"`
	Title                 string         `json:"title"`
	Description           string         `json:"description"`
	Probability           string         `json:"probability"`
	Impact                string         `json:"impact"`
	Severity              string         `json:"severity"`
	Category              string         `json:"category"`
	Status                string         `json:"status"`
	OwnerUserID           uuid.NullUUID  `json:"owner_user_id,omitempty"`
	OwnerName             sql.NullString `json:"owner_name,omitempty"`
	IdentifiedDate        time.Time      `json:"identified_date"`
	TargetResolutionDate  sql.NullTime   `json:"target_resolution_date,omitempty"`
	ClosedDate            sql.NullTime   `json:"closed_date,omitempty"`
	RealizedDate          sql.NullTime   `json:"realized_date,omitempty"`
	AIConfidenceScore     sql.NullFloat64 `json:"ai_confidence_score,omitempty"`
	AIDetectedAt          sql.NullTime   `json:"ai_detected_at,omitempty"`
	CreatedBy             uuid.UUID      `json:"created_by"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             sql.NullTime   `json:"deleted_at,omitempty"`
}

// RiskSuggestion represents an AI-detected risk awaiting user approval
type RiskSuggestion struct {
	SuggestionID         uuid.UUID       `json:"suggestion_id"`
	ProgramID            uuid.UUID       `json:"program_id"`
	Title                string          `json:"title"`
	Description          string          `json:"description"`
	Rationale            string          `json:"rationale"`
	SuggestedProbability string          `json:"suggested_probability"`
	SuggestedImpact      string          `json:"suggested_impact"`
	SuggestedSeverity    string          `json:"suggested_severity"`
	SuggestedCategory    string          `json:"suggested_category"`
	SourceType           string          `json:"source_type"`
	SourceArtifactIDs    []uuid.UUID     `json:"source_artifact_ids"`
	SourceInsightID      uuid.NullUUID   `json:"source_insight_id,omitempty"`
	SourceVarianceID     uuid.NullUUID   `json:"source_variance_id,omitempty"`
	AIConfidenceScore    sql.NullFloat64 `json:"ai_confidence_score,omitempty"`
	AIDetectedAt         time.Time       `json:"ai_detected_at"`
	IsApproved           bool            `json:"is_approved"`
	IsDismissed          bool            `json:"is_dismissed"`
	ApprovedBy           uuid.NullUUID   `json:"approved_by,omitempty"`
	ApprovedAt           sql.NullTime    `json:"approved_at,omitempty"`
	DismissedBy          uuid.NullUUID   `json:"dismissed_by,omitempty"`
	DismissedAt          sql.NullTime    `json:"dismissed_at,omitempty"`
	DismissalReason      sql.NullString  `json:"dismissal_reason,omitempty"`
	CreatedRiskID        uuid.NullUUID   `json:"created_risk_id,omitempty"`
}

// RiskMitigation represents a mitigation action for a risk
type RiskMitigation struct {
	MitigationID                uuid.UUID       `json:"mitigation_id"`
	RiskID                      uuid.UUID       `json:"risk_id"`
	Strategy                    string          `json:"strategy"`
	ActionDescription           string          `json:"action_description"`
	ExpectedProbabilityReduction sql.NullString  `json:"expected_probability_reduction,omitempty"`
	ExpectedImpactReduction     sql.NullString  `json:"expected_impact_reduction,omitempty"`
	EffectivenessRating         sql.NullInt32   `json:"effectiveness_rating,omitempty"`
	Status                      string          `json:"status"`
	AssignedTo                  uuid.NullUUID   `json:"assigned_to,omitempty"`
	TargetCompletionDate        sql.NullTime    `json:"target_completion_date,omitempty"`
	ActualCompletionDate        sql.NullTime    `json:"actual_completion_date,omitempty"`
	EstimatedCost               sql.NullFloat64 `json:"estimated_cost,omitempty"`
	ActualCost                  sql.NullFloat64 `json:"actual_cost,omitempty"`
	Currency                    string          `json:"currency"`
	CreatedBy                   uuid.UUID       `json:"created_by"`
	CreatedAt                   time.Time       `json:"created_at"`
	UpdatedAt                   time.Time       `json:"updated_at"`
	DeletedAt                   sql.NullTime    `json:"deleted_at,omitempty"`
}

// RiskArtifactLink represents a link between a risk and an artifact
type RiskArtifactLink struct {
	LinkID      uuid.UUID      `json:"link_id"`
	RiskID      uuid.UUID      `json:"risk_id"`
	ArtifactID  uuid.UUID      `json:"artifact_id"`
	LinkType    string         `json:"link_type"`
	Description sql.NullString `json:"description,omitempty"`
	CreatedBy   uuid.UUID      `json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
}

// ConversationThread represents a discussion thread about a risk
type ConversationThread struct {
	ThreadID       uuid.UUID      `json:"thread_id"`
	RiskID         uuid.UUID      `json:"risk_id"`
	Title          string         `json:"title"`
	ThreadType     string         `json:"thread_type"`
	IsResolved     bool           `json:"is_resolved"`
	ResolvedAt     sql.NullTime   `json:"resolved_at,omitempty"`
	ResolvedBy     uuid.NullUUID  `json:"resolved_by,omitempty"`
	MessageCount   int            `json:"message_count"`
	LastMessageAt  sql.NullTime   `json:"last_message_at,omitempty"`
	CreatedBy      uuid.UUID      `json:"created_by"`
	CreatedAt      time.Time      `json:"created_at"`
	DeletedAt      sql.NullTime   `json:"deleted_at,omitempty"`
}

// ConversationMessage represents a message in a conversation thread
type ConversationMessage struct {
	MessageID        uuid.UUID     `json:"message_id"`
	ThreadID         uuid.UUID     `json:"thread_id"`
	MessageText      string        `json:"message_text"`
	MessageFormat    string        `json:"message_format"`
	MentionedUserIDs []uuid.UUID   `json:"mentioned_user_ids"`
	CreatedBy        uuid.UUID     `json:"created_by"`
	CreatedAt        time.Time     `json:"created_at"`
	EditedAt         sql.NullTime  `json:"edited_at,omitempty"`
	DeletedAt        sql.NullTime  `json:"deleted_at,omitempty"`
}

// RiskWithMetadata combines risk with its related entities
type RiskWithMetadata struct {
	Risk
	Mitigations    []RiskMitigation     `json:"mitigations,omitempty"`
	LinkedArtifacts []RiskArtifactLink  `json:"linked_artifacts,omitempty"`
	Threads        []ConversationThread `json:"threads,omitempty"`
}

// ThreadWithMessages combines thread with its messages
type ThreadWithMessages struct {
	ConversationThread
	Messages []ConversationMessage `json:"messages,omitempty"`
}

// CreateRiskRequest represents a request to create a new risk
type CreateRiskRequest struct {
	ProgramID            uuid.UUID  `json:"program_id"`
	Title                string     `json:"title"`
	Description          string     `json:"description"`
	Probability          string     `json:"probability"`
	Impact               string     `json:"impact"`
	Category             string     `json:"category"`
	OwnerUserID          *uuid.UUID `json:"owner_user_id,omitempty"`
	OwnerName            string     `json:"owner_name,omitempty"`
	TargetResolutionDate *time.Time `json:"target_resolution_date,omitempty"`
	CreatedBy            uuid.UUID  `json:"created_by"`
}

// UpdateRiskRequest represents a request to update a risk
type UpdateRiskRequest struct {
	Title                *string    `json:"title,omitempty"`
	Description          *string    `json:"description,omitempty"`
	Probability          *string    `json:"probability,omitempty"`
	Impact               *string    `json:"impact,omitempty"`
	Category             *string    `json:"category,omitempty"`
	Status               *string    `json:"status,omitempty"`
	OwnerUserID          *uuid.UUID `json:"owner_user_id,omitempty"`
	OwnerName            *string    `json:"owner_name,omitempty"`
	TargetResolutionDate *time.Time `json:"target_resolution_date,omitempty"`
}

// CreateMitigationRequest represents a request to create a mitigation
type CreateMitigationRequest struct {
	RiskID                       uuid.UUID  `json:"risk_id"`
	Strategy                     string     `json:"strategy"`
	ActionDescription            string     `json:"action_description"`
	ExpectedProbabilityReduction string     `json:"expected_probability_reduction,omitempty"`
	ExpectedImpactReduction      string     `json:"expected_impact_reduction,omitempty"`
	AssignedTo                   *uuid.UUID `json:"assigned_to,omitempty"`
	TargetCompletionDate         *time.Time `json:"target_completion_date,omitempty"`
	EstimatedCost                *float64   `json:"estimated_cost,omitempty"`
	Currency                     string     `json:"currency"`
	CreatedBy                    uuid.UUID  `json:"created_by"`
}

// UpdateMitigationRequest represents a request to update a mitigation
type UpdateMitigationRequest struct {
	Status               *string    `json:"status,omitempty"`
	EffectivenessRating  *int       `json:"effectiveness_rating,omitempty"`
	ActualCompletionDate *time.Time `json:"actual_completion_date,omitempty"`
	ActualCost           *float64   `json:"actual_cost,omitempty"`
}

// LinkArtifactRequest represents a request to link an artifact to a risk
type LinkArtifactRequest struct {
	RiskID      uuid.UUID `json:"risk_id"`
	ArtifactID  uuid.UUID `json:"artifact_id"`
	LinkType    string    `json:"link_type"`
	Description string    `json:"description,omitempty"`
	CreatedBy   uuid.UUID `json:"created_by"`
}

// CreateThreadRequest represents a request to create a conversation thread
type CreateThreadRequest struct {
	RiskID     uuid.UUID `json:"risk_id"`
	Title      string    `json:"title"`
	ThreadType string    `json:"thread_type"`
	CreatedBy  uuid.UUID `json:"created_by"`
}

// CreateMessageRequest represents a request to create a message
type CreateMessageRequest struct {
	ThreadID      uuid.UUID   `json:"thread_id"`
	MessageText   string      `json:"message_text"`
	MessageFormat string      `json:"message_format"`
	CreatedBy     uuid.UUID   `json:"created_by"`
}

// RiskFilterRequest represents filters for listing risks
type RiskFilterRequest struct {
	ProgramID   uuid.UUID
	Status      string
	Category    string
	Severity    string
	OwnerUserID *uuid.UUID
	Limit       int
	Offset      int
}

// ApproveSuggestionRequest represents a request to approve a risk suggestion
type ApproveSuggestionRequest struct {
	SuggestionID         uuid.UUID  `json:"suggestion_id"`
	ApprovedBy           uuid.UUID  `json:"approved_by"`
	OwnerUserID          *uuid.UUID `json:"owner_user_id,omitempty"`
	TargetResolutionDate *time.Time `json:"target_resolution_date,omitempty"`
	// Allow overriding AI suggestions
	OverrideProbability *string `json:"override_probability,omitempty"`
	OverrideImpact      *string `json:"override_impact,omitempty"`
	OverrideCategory    *string `json:"override_category,omitempty"`
}

// DismissSuggestionRequest represents a request to dismiss a risk suggestion
type DismissSuggestionRequest struct {
	SuggestionID uuid.UUID `json:"suggestion_id"`
	DismissedBy  uuid.UUID `json:"dismissed_by"`
	Reason       string    `json:"reason"`
}
