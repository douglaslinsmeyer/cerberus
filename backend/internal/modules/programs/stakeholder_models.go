package programs

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Stakeholder represents a program stakeholder
type Stakeholder struct {
	StakeholderID   uuid.UUID       `json:"stakeholder_id"`
	ProgramID       uuid.UUID       `json:"program_id"`
	PersonName      string          `json:"person_name"`
	StakeholderType string          `json:"stakeholder_type"`
	IsInternal      bool            `json:"is_internal"`
	Email           sql.NullString  `json:"email,omitempty"`
	Role            sql.NullString  `json:"role,omitempty"`
	Organization    sql.NullString  `json:"organization,omitempty"`
	EngagementLevel sql.NullString  `json:"engagement_level,omitempty"`
	Department      sql.NullString  `json:"department,omitempty"`
	Influence       sql.NullString  `json:"influence,omitempty"`
	Notes           sql.NullString  `json:"notes,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       sql.NullTime    `json:"deleted_at,omitempty"`
}

// StakeholderFilter represents filter parameters for stakeholders
type StakeholderFilter struct {
	ProgramID       uuid.UUID
	Role            string
	Organization    string
	StakeholderType string
	IsInternal      *bool
	EngagementLevel string
	Limit           int
	Offset          int
}

// CreateStakeholderRequest represents a request to create a stakeholder
type CreateStakeholderRequest struct {
	PersonName      string  `json:"person_name"`
	StakeholderType string  `json:"stakeholder_type"`
	IsInternal      bool    `json:"is_internal"`
	Email           *string `json:"email,omitempty"`
	Role            *string `json:"role,omitempty"`
	Organization    *string `json:"organization,omitempty"`
	EngagementLevel *string `json:"engagement_level,omitempty"`
	Department      *string `json:"department,omitempty"`
	Influence       *string `json:"influence,omitempty"`
	Notes           *string `json:"notes,omitempty"`
}

// UpdateStakeholderRequest represents a request to update a stakeholder
type UpdateStakeholderRequest struct {
	PersonName      *string `json:"person_name,omitempty"`
	StakeholderType *string `json:"stakeholder_type,omitempty"`
	IsInternal      *bool   `json:"is_internal,omitempty"`
	Email           *string `json:"email,omitempty"`
	Role            *string `json:"role,omitempty"`
	Organization    *string `json:"organization,omitempty"`
	EngagementLevel *string `json:"engagement_level,omitempty"`
	Department      *string `json:"department,omitempty"`
	Influence       *string `json:"influence,omitempty"`
	Notes           *string `json:"notes,omitempty"`
}

// PersonSuggestion represents a person mention that could become a stakeholder
type PersonSuggestion struct {
	PersonID                 uuid.UUID             `json:"person_id"`
	PersonName               string                `json:"person_name"`
	PersonRole               sql.NullString        `json:"person_role,omitempty"`
	PersonOrganization       sql.NullString        `json:"person_organization,omitempty"`
	ConfidenceScore          sql.NullFloat64       `json:"confidence_score,omitempty"`
	ArtifactCount            int                   `json:"artifact_count"`
	TotalMentions            int                   `json:"total_mentions"`
	LastMentioned            time.Time             `json:"last_mentioned"`
	SuggestedStakeholderID   uuid.NullUUID         `json:"suggested_stakeholder_id,omitempty"`
	SimilarityScore          float64               `json:"similarity_score,omitempty"`
	Artifacts                []SuggestionArtifact  `json:"artifacts"`
	SuggestedStakeholderType string                `json:"suggested_stakeholder_type"`
	SuggestedIsInternal      bool                  `json:"suggested_is_internal"`
}

// SuggestionArtifact represents an artifact where a person is mentioned
type SuggestionArtifact struct {
	ArtifactID   uuid.UUID `json:"artifact_id"`
	Filename     string    `json:"filename"`
	MentionCount int       `json:"mention_count"`
}

// LinkPersonRequest represents a request to link a person to a stakeholder
type LinkPersonRequest struct {
	StakeholderID uuid.UUID `json:"stakeholder_id"`
}

// PersonMergeGroup represents a group of similar person mentions
type PersonMergeGroup struct {
	GroupID               uuid.UUID      `json:"group_id"`
	ProgramID             uuid.UUID      `json:"program_id"`
	SuggestedName         string         `json:"suggested_name"`
	Status                string         `json:"status"`
	HasRoleConflicts      bool           `json:"has_role_conflicts"`
	HasOrgConflicts       bool           `json:"has_org_conflicts"`
	ResolvedName          sql.NullString `json:"resolved_name,omitempty"`
	ResolvedRole          sql.NullString `json:"resolved_role,omitempty"`
	ResolvedOrganization  sql.NullString `json:"resolved_organization,omitempty"`
	MergedStakeholderID   uuid.NullUUID  `json:"merged_stakeholder_id,omitempty"`
	MergedAt              sql.NullTime   `json:"merged_at,omitempty"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

// PersonMergeGroupMember represents a person in a merge group
type PersonMergeGroupMember struct {
	MemberID        uuid.UUID `json:"member_id"`
	GroupID         uuid.UUID `json:"group_id"`
	PersonID        uuid.UUID `json:"person_id"`
	SimilarityScore float64   `json:"similarity_score"`
	MatchingMethod  string    `json:"matching_method"`
	AddedAt         time.Time `json:"added_at"`
}

// GroupedSuggestion represents a grouped set of person mentions for review
type GroupedSuggestion struct {
	GroupID            uuid.UUID          `json:"group_id"`
	SuggestedName      string             `json:"suggested_name"`
	Status             string             `json:"status"`
	HasRoleConflicts   bool               `json:"has_role_conflicts"`
	HasOrgConflicts    bool               `json:"has_org_conflicts"`
	TotalPersons       int                `json:"total_persons"`
	TotalArtifacts     int                `json:"total_artifacts"`
	TotalMentions      int                `json:"total_mentions"`
	AverageConfidence  float64            `json:"average_confidence"`
	LastMentioned      time.Time          `json:"last_mentioned"`
	RoleOptions        []ConflictOption   `json:"role_options,omitempty"`
	OrgOptions         []ConflictOption   `json:"org_options,omitempty"`
	Members            []GroupMember      `json:"members"`
	AllContexts        []ContextMention   `json:"all_contexts"`
}

// ConflictOption represents a value choice when resolving conflicts
type ConflictOption struct {
	Value      string  `json:"value"`
	Count      int     `json:"count"`
	Confidence float64 `json:"confidence"`
}

// GroupMember represents a person within a grouped suggestion
type GroupMember struct {
	PersonID           uuid.UUID      `json:"person_id"`
	PersonName         string         `json:"person_name"`
	PersonRole         sql.NullString `json:"person_role,omitempty"`
	PersonOrganization sql.NullString `json:"person_organization,omitempty"`
	ConfidenceScore    float64        `json:"confidence_score"`
	SimilarityScore    float64        `json:"similarity_score"`
	ArtifactCount      int            `json:"artifact_count"`
	MentionCount       int            `json:"mention_count"`
}

// ContextMention represents a single mention of a person in a document
type ContextMention struct {
	ArtifactID   uuid.UUID `json:"artifact_id"`
	ArtifactName string    `json:"artifact_name"`
	UploadedAt   time.Time `json:"uploaded_at"`
	Snippet      string    `json:"snippet"`
	PersonName   string    `json:"person_name"`
}

// ConfirmMergeGroupRequest represents a request to confirm a merge group
type ConfirmMergeGroupRequest struct {
	SelectedName         string  `json:"selected_name"`
	SelectedRole         *string `json:"selected_role,omitempty"`
	SelectedOrganization *string `json:"selected_organization,omitempty"`
	CreateStakeholder    bool    `json:"create_stakeholder"`
}

// ModifyGroupMembersRequest represents a request to add/remove persons from a group
type ModifyGroupMembersRequest struct {
	AddPersonIDs    []uuid.UUID `json:"add_person_ids,omitempty"`
	RemovePersonIDs []uuid.UUID `json:"remove_person_ids,omitempty"`
}

// LinkedArtifact represents an artifact where a stakeholder is mentioned
type LinkedArtifact struct {
	ArtifactID   uuid.UUID `json:"artifact_id"`
	Filename     string    `json:"filename"`
	UploadedAt   time.Time `json:"uploaded_at"`
	MentionCount int       `json:"mention_count"`
}
