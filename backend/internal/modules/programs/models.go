package programs

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Program represents a program in the system
type Program struct {
	ProgramID            uuid.UUID      `json:"program_id"`
	ProgramName          string         `json:"program_name"`
	ProgramCode          string         `json:"program_code"`
	Description          sql.NullString `json:"description,omitempty"`
	StartDate            sql.NullTime   `json:"start_date,omitempty"`
	EndDate              sql.NullTime   `json:"end_date,omitempty"`
	Status               string         `json:"status"`
	InternalOrganization string         `json:"internal_organization"` // NEW: Simple field for AI context
	Configuration        ProgramConfig  `json:"configuration"`
	CreatedAt            time.Time      `json:"created_at"`
	CreatedBy            uuid.UUID      `json:"created_by"`
	UpdatedAt            time.Time      `json:"updated_at"`
	UpdatedBy            uuid.NullUUID  `json:"updated_by,omitempty"`
	DeletedAt            sql.NullTime   `json:"deleted_at,omitempty"`
}

// ProgramWithStats represents a program with aggregated statistics
type ProgramWithStats struct {
	Program
	ArtifactCount int `json:"artifact_count"`
	InvoiceCount  int `json:"invoice_count"`
	RiskCount     int `json:"risk_count"`
}

// CreateProgramRequest represents a request to create a new program
type CreateProgramRequest struct {
	ProgramName string  `json:"program_name"`
	ProgramCode string  `json:"program_code"`
	Description *string `json:"description,omitempty"`
	StartDate   *string `json:"start_date,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
	Status      string  `json:"status"`
}

// UpdateProgramRequest represents a request to update an existing program
type UpdateProgramRequest struct {
	ProgramName          *string `json:"program_name,omitempty"`
	Description          *string `json:"description,omitempty"`
	StartDate            *string `json:"start_date,omitempty"`
	EndDate              *string `json:"end_date,omitempty"`
	Status               *string `json:"status,omitempty"`
	InternalOrganization *string `json:"internal_organization,omitempty"`
}

// ListProgramsResponse represents the response for listing programs
type ListProgramsResponse struct {
	Programs []ProgramWithStats `json:"programs"`
	Total    int                `json:"total"`
}
