package programs

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service handles program business logic
type Service struct {
	repo *Repository
}

// NewService creates a new programs service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ListPrograms retrieves all programs with statistics
func (s *Service) ListPrograms(ctx context.Context) ([]ProgramWithStats, error) {
	return s.repo.ListProgramsWithStats(ctx)
}

// GetProgram retrieves a single program by ID
func (s *Service) GetProgram(ctx context.Context, programID uuid.UUID) (*ProgramWithStats, error) {
	return s.repo.GetProgramByID(ctx, programID)
}

// CreateProgram creates a new program with validation
func (s *Service) CreateProgram(ctx context.Context, req CreateProgramRequest, userID uuid.UUID) (uuid.UUID, error) {
	// Validate request
	if err := s.validateCreateRequest(ctx, req); err != nil {
		return uuid.Nil, err
	}

	// Parse dates if provided
	var startDate, endDate sql.NullTime
	if req.StartDate != nil && *req.StartDate != "" {
		parsedStart, err := time.Parse("2006-01-02", *req.StartDate)
		if err != nil {
			return uuid.Nil, fmt.Errorf("invalid start_date format, use YYYY-MM-DD")
		}
		startDate = sql.NullTime{Time: parsedStart, Valid: true}
	}

	if req.EndDate != nil && *req.EndDate != "" {
		parsedEnd, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			return uuid.Nil, fmt.Errorf("invalid end_date format, use YYYY-MM-DD")
		}
		endDate = sql.NullTime{Time: parsedEnd, Valid: true}
	}

	// Validate date range
	if startDate.Valid && endDate.Valid && endDate.Time.Before(startDate.Time) {
		return uuid.Nil, fmt.Errorf("end_date must be after start_date")
	}

	// Create program
	program := &Program{
		ProgramID:   uuid.New(),
		ProgramName: strings.TrimSpace(req.ProgramName),
		ProgramCode: strings.TrimSpace(strings.ToUpper(req.ProgramCode)),
		Status:      req.Status,
		CreatedAt:   time.Now(),
		CreatedBy:   userID,
		UpdatedAt:   time.Now(),
	}

	if req.Description != nil && *req.Description != "" {
		program.Description = sql.NullString{String: strings.TrimSpace(*req.Description), Valid: true}
	}

	program.StartDate = startDate
	program.EndDate = endDate

	if err := s.repo.CreateProgram(ctx, program); err != nil {
		return uuid.Nil, err
	}

	return program.ProgramID, nil
}

// UpdateProgram updates an existing program with validation
func (s *Service) UpdateProgram(ctx context.Context, programID uuid.UUID, req UpdateProgramRequest) error {
	// Get existing program
	existing, err := s.repo.GetProgramByID(ctx, programID)
	if err != nil {
		return err
	}

	// Validate update request
	if err := s.validateUpdateRequest(req); err != nil {
		return err
	}

	// Build update
	program := &Program{
		ProgramID:   programID,
		ProgramName: existing.ProgramName,
		Description: existing.Description,
		StartDate:   existing.StartDate,
		EndDate:     existing.EndDate,
		Status:      existing.Status,
		UpdatedAt:   time.Now(),
		UpdatedBy:   uuid.NullUUID{UUID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Valid: true}, // TODO: Get from JWT
	}

	// Apply updates
	if req.ProgramName != nil && *req.ProgramName != "" {
		program.ProgramName = strings.TrimSpace(*req.ProgramName)
	}

	if req.Description != nil {
		if *req.Description == "" {
			program.Description = sql.NullString{Valid: false}
		} else {
			program.Description = sql.NullString{String: strings.TrimSpace(*req.Description), Valid: true}
		}
	}

	if req.StartDate != nil {
		if *req.StartDate == "" {
			program.StartDate = sql.NullTime{Valid: false}
		} else {
			parsedStart, err := time.Parse("2006-01-02", *req.StartDate)
			if err != nil {
				return fmt.Errorf("invalid start_date format, use YYYY-MM-DD")
			}
			program.StartDate = sql.NullTime{Time: parsedStart, Valid: true}
		}
	}

	if req.EndDate != nil {
		if *req.EndDate == "" {
			program.EndDate = sql.NullTime{Valid: false}
		} else {
			parsedEnd, err := time.Parse("2006-01-02", *req.EndDate)
			if err != nil {
				return fmt.Errorf("invalid end_date format, use YYYY-MM-DD")
			}
			program.EndDate = sql.NullTime{Time: parsedEnd, Valid: true}
		}
	}

	// Validate date range after updates
	if program.StartDate.Valid && program.EndDate.Valid && program.EndDate.Time.Before(program.StartDate.Time) {
		return fmt.Errorf("end_date must be after start_date")
	}

	if req.Status != nil && *req.Status != "" {
		program.Status = *req.Status
	}

	return s.repo.UpdateProgram(ctx, program)
}

// validateCreateRequest validates a create program request
func (s *Service) validateCreateRequest(ctx context.Context, req CreateProgramRequest) error {
	// Validate program name
	if strings.TrimSpace(req.ProgramName) == "" {
		return fmt.Errorf("program_name is required")
	}
	if len(req.ProgramName) > 255 {
		return fmt.Errorf("program_name must be 255 characters or less")
	}

	// Validate program code
	if strings.TrimSpace(req.ProgramCode) == "" {
		return fmt.Errorf("program_code is required")
	}
	if len(req.ProgramCode) > 50 {
		return fmt.Errorf("program_code must be 50 characters or less")
	}

	// Validate program code format (alphanumeric, hyphens, underscores)
	codePattern := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	if !codePattern.MatchString(req.ProgramCode) {
		return fmt.Errorf("program_code must contain only alphanumeric characters, hyphens, and underscores")
	}

	// Check if program code already exists
	exists, err := s.repo.ProgramCodeExists(ctx, strings.ToUpper(req.ProgramCode))
	if err != nil {
		return fmt.Errorf("failed to check program code: %w", err)
	}
	if exists {
		return fmt.Errorf("program code already exists")
	}

	// Validate status
	validStatuses := map[string]bool{
		"active":    true,
		"planning":  true,
		"on-hold":   true,
		"completed": true,
		"archived":  true,
	}
	if !validStatuses[req.Status] {
		return fmt.Errorf("status must be one of: active, planning, on-hold, completed, archived")
	}

	return nil
}

// validateUpdateRequest validates an update program request
func (s *Service) validateUpdateRequest(req UpdateProgramRequest) error {
	// Validate program name if provided
	if req.ProgramName != nil {
		if len(*req.ProgramName) > 255 {
			return fmt.Errorf("program_name must be 255 characters or less")
		}
	}

	// Validate status if provided
	if req.Status != nil && *req.Status != "" {
		validStatuses := map[string]bool{
			"active":    true,
			"planning":  true,
			"on-hold":   true,
			"completed": true,
			"archived":  true,
		}
		if !validStatuses[*req.Status] {
			return fmt.Errorf("status must be one of: active, planning, on-hold, completed, archived")
		}
	}

	return nil
}
