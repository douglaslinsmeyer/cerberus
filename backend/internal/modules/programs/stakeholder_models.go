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
