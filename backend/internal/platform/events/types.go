package events

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents different types of events in the system
type EventType string

const (
	// Artifact events
	ArtifactUploaded         EventType = "artifact.uploaded"
	ArtifactAnalyzed         EventType = "artifact.analyzed"
	ArtifactMetadataExtracted EventType = "artifact.metadata_extracted"
	ArtifactEmbeddingsCreated EventType = "artifact.embeddings_created"

	// Financial events
	InvoiceProcessed        EventType = "financial.invoice_processed"
	VarianceDetected        EventType = "financial.variance_detected"
	BudgetThresholdExceeded EventType = "financial.budget_exceeded"

	// Risk events
	RiskIdentified EventType = "risk.identified"
	RiskEscalated  EventType = "risk.escalated"
	IssueCreated   EventType = "risk.issue_created"

	// Decision events
	DecisionExtracted EventType = "decision.extracted"
	DecisionApproved  EventType = "decision.approved"

	// Change events
	ChangeProposed EventType = "change.proposed"
	ChangeApproved EventType = "change.approved"
)

// Event represents a system event
type Event struct {
	ID            uuid.UUID              `json:"id"`
	Type          EventType              `json:"type"`
	ProgramID     uuid.UUID              `json:"program_id"`
	Timestamp     time.Time              `json:"timestamp"`
	Source        string                 `json:"source"` // Module name
	Payload       map[string]interface{} `json:"payload"`
	CorrelationID uuid.UUID              `json:"correlation_id"`
	Metadata      EventMetadata          `json:"metadata"`
}

// EventMetadata contains additional event metadata
type EventMetadata struct {
	AIGenerated  bool     `json:"ai_generated"`
	Confidence   float64  `json:"confidence,omitempty"`
	ArtifactRefs []uuid.UUID `json:"artifact_refs,omitempty"`
}

// EventHandler is a function that handles an event
type EventHandler func(ctx context.Context, event *Event) error

// NewEvent creates a new event
func NewEvent(eventType EventType, programID uuid.UUID, source string, payload map[string]interface{}) *Event {
	return &Event{
		ID:            uuid.New(),
		Type:          eventType,
		ProgramID:     programID,
		Timestamp:     time.Now(),
		Source:        source,
		Payload:       payload,
		CorrelationID: uuid.New(),
		Metadata:      EventMetadata{},
	}
}

// WithCorrelationID sets the correlation ID for tracing related events
func (e *Event) WithCorrelationID(correlationID uuid.UUID) *Event {
	e.CorrelationID = correlationID
	return e
}

// WithAIMetadata adds AI-related metadata
func (e *Event) WithAIMetadata(confidence float64, artifactRefs []uuid.UUID) *Event {
	e.Metadata.AIGenerated = true
	e.Metadata.Confidence = confidence
	e.Metadata.ArtifactRefs = artifactRefs
	return e
}
