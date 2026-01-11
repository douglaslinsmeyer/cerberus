package artifacts

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Artifact represents a document uploaded to the system
type Artifact struct {
	ArtifactID         uuid.UUID      `json:"artifact_id"`
	ProgramID          uuid.UUID      `json:"program_id"`
	Filename           string         `json:"filename"`
	StoragePath        string         `json:"storage_path"`
	FileType           string         `json:"file_type"`
	FileSizeBytes      int64          `json:"file_size_bytes"`
	MimeType           string         `json:"mime_type"`
	RawContent         sql.NullString `json:"raw_content,omitempty"`
	ContentHash        string         `json:"content_hash"`
	ArtifactCategory   sql.NullString `json:"artifact_category,omitempty"`
	ArtifactSubcategory sql.NullString `json:"artifact_subcategory,omitempty"`
	ProcessingStatus   string         `json:"processing_status"`
	ProcessedAt        sql.NullTime   `json:"processed_at,omitempty"`
	AIModelVersion     sql.NullString `json:"ai_model_version,omitempty"`
	AIProcessingTimeMs sql.NullInt32  `json:"ai_processing_time_ms,omitempty"`
	UploadedBy         uuid.UUID      `json:"uploaded_by"`
	UploadedAt         time.Time      `json:"uploaded_at"`
	VersionNumber      int            `json:"version_number"`
	SupersededBy       uuid.NullUUID  `json:"superseded_by,omitempty"`
	DeletedAt          sql.NullTime   `json:"deleted_at,omitempty"`
}

// Topic represents an AI-extracted topic from an artifact
type Topic struct {
	TopicID         uuid.UUID `json:"topic_id"`
	ArtifactID      uuid.UUID `json:"artifact_id"`
	TopicName       string    `json:"topic_name"`
	ConfidenceScore float64   `json:"confidence_score"`
	ParentTopicID   uuid.NullUUID `json:"parent_topic_id,omitempty"`
	TopicLevel      int       `json:"topic_level"`
	ExtractedAt     time.Time `json:"extracted_at"`
}

// Person represents an AI-extracted person mention from an artifact
type Person struct {
	PersonID          uuid.UUID      `json:"person_id"`
	ArtifactID        uuid.UUID      `json:"artifact_id"`
	PersonName        string         `json:"person_name"`
	PersonRole        sql.NullString `json:"person_role,omitempty"`
	PersonOrganization sql.NullString `json:"person_organization,omitempty"`
	MentionCount      int            `json:"mention_count"`
	ContextSnippets   []byte         `json:"context_snippets"` // JSONB
	ConfidenceScore   sql.NullFloat64 `json:"confidence_score,omitempty"`
	StakeholderID     uuid.NullUUID  `json:"stakeholder_id,omitempty"`
	ExtractedAt       time.Time      `json:"extracted_at"`
}

// Fact represents an AI-extracted fact from an artifact
type Fact struct {
	FactID                  uuid.UUID       `json:"fact_id"`
	ArtifactID              uuid.UUID       `json:"artifact_id"`
	FactType                string          `json:"fact_type"`
	FactKey                 string          `json:"fact_key"`
	FactValue               string          `json:"fact_value"`
	NormalizedValueNumeric  sql.NullFloat64 `json:"normalized_value_numeric,omitempty"`
	NormalizedValueDate     sql.NullTime    `json:"normalized_value_date,omitempty"`
	NormalizedValueBoolean  sql.NullBool    `json:"normalized_value_boolean,omitempty"`
	Unit                    sql.NullString  `json:"unit,omitempty"`
	ConfidenceScore         sql.NullFloat64 `json:"confidence_score,omitempty"`
	ContextSnippet          sql.NullString  `json:"context_snippet,omitempty"`
	ExtractedAt             time.Time       `json:"extracted_at"`
}

// Insight represents an AI-generated insight from an artifact
type Insight struct {
	InsightID        uuid.UUID      `json:"insight_id"`
	ArtifactID       uuid.UUID      `json:"artifact_id"`
	InsightType      string         `json:"insight_type"`
	InsightCategory  sql.NullString `json:"insight_category,omitempty"`
	Title            string         `json:"title"`
	Description      string         `json:"description"`
	Severity         sql.NullString `json:"severity,omitempty"`
	SuggestedAction  sql.NullString `json:"suggested_action,omitempty"`
	ImpactedModules  []string       `json:"impacted_modules"`
	ConfidenceScore  sql.NullFloat64 `json:"confidence_score,omitempty"`
	UserRating       sql.NullInt32  `json:"user_rating,omitempty"`
	UserFeedback     sql.NullString `json:"user_feedback,omitempty"`
	IsDismissed      bool           `json:"is_dismissed"`
	ExtractedAt      time.Time      `json:"extracted_at"`
	DismissedAt      sql.NullTime   `json:"dismissed_at,omitempty"`
	DismissedBy      uuid.NullUUID  `json:"dismissed_by,omitempty"`
}

// ArtifactChunk represents a chunk of document text
type ArtifactChunk struct {
	ChunkID        uuid.UUID `json:"chunk_id"`
	ArtifactID     uuid.UUID `json:"artifact_id"`
	ChunkIndex     int       `json:"chunk_index"`
	ChunkText      string    `json:"chunk_text"`
	ChunkStartOffset sql.NullInt32 `json:"chunk_start_offset,omitempty"`
	ChunkEndOffset   sql.NullInt32 `json:"chunk_end_offset,omitempty"`
	TokenCount     sql.NullInt32 `json:"token_count,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// ArtifactEmbedding represents a vector embedding for semantic search
type ArtifactEmbedding struct {
	EmbeddingID    uuid.UUID `json:"embedding_id"`
	ArtifactID     uuid.UUID `json:"artifact_id"`
	ChunkID        uuid.UUID `json:"chunk_id"`
	Embedding      []float32 `json:"embedding"` // 1536-dimensional vector
	EmbeddingModel string    `json:"embedding_model"`
	CreatedAt      time.Time `json:"created_at"`
}

// ArtifactSummary represents an AI-generated summary
type ArtifactSummary struct {
	SummaryID        uuid.UUID `json:"summary_id"`
	ArtifactID       uuid.UUID `json:"artifact_id"`
	ExecutiveSummary string    `json:"executive_summary"`
	KeyTakeaways     []string  `json:"key_takeaways"`
	Sentiment        sql.NullString `json:"sentiment,omitempty"`
	Priority         sql.NullInt32  `json:"priority,omitempty"`
	ConfidenceScore  sql.NullFloat64 `json:"confidence_score,omitempty"`
	AIModel          sql.NullString `json:"ai_model,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// ArtifactWithMetadata combines artifact with its extracted metadata
type ArtifactWithMetadata struct {
	Artifact
	Summary  *ArtifactSummary `json:"summary,omitempty"`
	Topics   []Topic          `json:"topics,omitempty"`
	Persons  []Person         `json:"persons,omitempty"`
	Facts    []Fact           `json:"facts,omitempty"`
	Insights []Insight        `json:"insights,omitempty"`
}

// UploadRequest represents an artifact upload request
type UploadRequest struct {
	ProgramID uuid.UUID
	Filename  string
	MimeType  string
	Data      []byte
	UploadedBy uuid.UUID
}

// SearchRequest represents a semantic search request
type SearchRequest struct {
	ProgramID uuid.UUID
	Query     string
	Limit     int
}

// SearchResult represents a search result with relevance score
type SearchResult struct {
	Artifact   Artifact `json:"artifact"`
	Similarity float64  `json:"similarity"`
	Snippet    string   `json:"snippet"`
}
