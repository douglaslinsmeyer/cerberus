package programs

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/cerberus/backend/internal/platform/db"
	"github.com/google/uuid"
)

// StakeholderRepository handles database operations for stakeholders
type StakeholderRepository struct {
	db *db.DB
}

// NewStakeholderRepository creates a new stakeholder repository
func NewStakeholderRepository(database *db.DB) *StakeholderRepository {
	return &StakeholderRepository{db: database}
}

// Create inserts a new stakeholder
func (r *StakeholderRepository) Create(ctx context.Context, stakeholder *Stakeholder) error {
	query := `
		INSERT INTO program_stakeholders (
			stakeholder_id, program_id, person_name, email, role, organization,
			stakeholder_type, is_internal, engagement_level, department, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		stakeholder.StakeholderID,
		stakeholder.ProgramID,
		stakeholder.PersonName,
		stakeholder.Email,
		stakeholder.Role,
		stakeholder.Organization,
		stakeholder.StakeholderType,
		stakeholder.IsInternal,
		stakeholder.EngagementLevel,
		stakeholder.Department,
		stakeholder.Notes,
	)

	if err != nil {
		return fmt.Errorf("failed to create stakeholder: %w", err)
	}

	return nil
}

// GetByID retrieves a stakeholder by ID
func (r *StakeholderRepository) GetByID(ctx context.Context, stakeholderID uuid.UUID) (*Stakeholder, error) {
	query := `
		SELECT stakeholder_id, program_id, person_name, email, role, organization,
		       stakeholder_type, is_internal, engagement_level, department, notes,
		       created_at, updated_at, deleted_at
		FROM program_stakeholders
		WHERE stakeholder_id = $1 AND deleted_at IS NULL
	`

	var s Stakeholder
	err := r.db.QueryRowContext(ctx, query, stakeholderID).Scan(
		&s.StakeholderID,
		&s.ProgramID,
		&s.PersonName,
		&s.Email,
		&s.Role,
		&s.Organization,
		&s.StakeholderType,
		&s.IsInternal,
		&s.EngagementLevel,
		&s.Department,
		&s.Notes,
		&s.CreatedAt,
		&s.UpdatedAt,
		&s.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("stakeholder not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get stakeholder: %w", err)
	}

	return &s, nil
}

// ListByProgram retrieves all stakeholders for a program with optional filtering
func (r *StakeholderRepository) ListByProgram(ctx context.Context, filter StakeholderFilter) ([]Stakeholder, error) {
	// Build query with filters
	query := `
		SELECT stakeholder_id, program_id, person_name, email, role, organization,
		       stakeholder_type, is_internal, engagement_level, department, notes,
		       created_at, updated_at, deleted_at
		FROM program_stakeholders
		WHERE program_id = $1 AND deleted_at IS NULL
	`
	args := []interface{}{filter.ProgramID}
	argPos := 2

	// Add optional filters
	if filter.StakeholderType != "" {
		query += fmt.Sprintf(" AND stakeholder_type = $%d", argPos)
		args = append(args, filter.StakeholderType)
		argPos++
	}

	if filter.IsInternal != nil {
		query += fmt.Sprintf(" AND is_internal = $%d", argPos)
		args = append(args, *filter.IsInternal)
		argPos++
	}

	if filter.EngagementLevel != "" {
		query += fmt.Sprintf(" AND engagement_level = $%d", argPos)
		args = append(args, filter.EngagementLevel)
		argPos++
	}

	// Add ordering and pagination
	query += " ORDER BY person_name ASC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filter.Limit)
		argPos++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list stakeholders: %w", err)
	}
	defer rows.Close()

	var stakeholders []Stakeholder
	for rows.Next() {
		var s Stakeholder
		err := rows.Scan(
			&s.StakeholderID,
			&s.ProgramID,
			&s.PersonName,
			&s.Email,
			&s.Role,
			&s.Organization,
			&s.StakeholderType,
			&s.IsInternal,
			&s.EngagementLevel,
			&s.Department,
			&s.Notes,
			&s.CreatedAt,
			&s.UpdatedAt,
			&s.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stakeholder: %w", err)
		}
		stakeholders = append(stakeholders, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating stakeholders: %w", err)
	}

	return stakeholders, nil
}

// ListInternal retrieves internal stakeholders for a program
func (r *StakeholderRepository) ListInternal(ctx context.Context, programID uuid.UUID) ([]Stakeholder, error) {
	filter := StakeholderFilter{
		ProgramID:  programID,
		IsInternal: boolPtr(true),
		Limit:      100, // Reasonable limit for internal stakeholders
	}
	return r.ListByProgram(ctx, filter)
}

// ListExternal retrieves external stakeholders for a program
func (r *StakeholderRepository) ListExternal(ctx context.Context, programID uuid.UUID) ([]Stakeholder, error) {
	filter := StakeholderFilter{
		ProgramID:  programID,
		IsInternal: boolPtr(false),
		Limit:      100, // Reasonable limit for external stakeholders
	}
	return r.ListByProgram(ctx, filter)
}

// Update modifies an existing stakeholder
func (r *StakeholderRepository) Update(ctx context.Context, stakeholderID uuid.UUID, req UpdateStakeholderRequest) error {
	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.PersonName != nil {
		updates = append(updates, fmt.Sprintf("person_name = $%d", argPos))
		args = append(args, *req.PersonName)
		argPos++
	}
	if req.Email != nil {
		updates = append(updates, fmt.Sprintf("email = $%d", argPos))
		args = append(args, sqlNullString(*req.Email))
		argPos++
	}
	if req.Role != nil {
		updates = append(updates, fmt.Sprintf("role = $%d", argPos))
		args = append(args, sqlNullString(*req.Role))
		argPos++
	}
	if req.Organization != nil {
		updates = append(updates, fmt.Sprintf("organization = $%d", argPos))
		args = append(args, sqlNullString(*req.Organization))
		argPos++
	}
	if req.StakeholderType != nil {
		updates = append(updates, fmt.Sprintf("stakeholder_type = $%d", argPos))
		args = append(args, *req.StakeholderType)
		argPos++
	}
	if req.IsInternal != nil {
		updates = append(updates, fmt.Sprintf("is_internal = $%d", argPos))
		args = append(args, *req.IsInternal)
		argPos++
	}
	if req.EngagementLevel != nil {
		updates = append(updates, fmt.Sprintf("engagement_level = $%d", argPos))
		args = append(args, sqlNullString(*req.EngagementLevel))
		argPos++
	}
	if req.Department != nil {
		updates = append(updates, fmt.Sprintf("department = $%d", argPos))
		args = append(args, sqlNullString(*req.Department))
		argPos++
	}
	if req.Notes != nil {
		updates = append(updates, fmt.Sprintf("notes = $%d", argPos))
		args = append(args, sqlNullString(*req.Notes))
		argPos++
	}

	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	// Add updated_at
	updates = append(updates, fmt.Sprintf("updated_at = $%d", argPos))
	args = append(args, "NOW()")
	argPos++

	// Add stakeholder ID
	args = append(args, stakeholderID)

	query := fmt.Sprintf(`
		UPDATE program_stakeholders
		SET %s
		WHERE stakeholder_id = $%d AND deleted_at IS NULL
	`, strings.Join(updates, ", "), argPos)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update stakeholder: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("stakeholder not found or already deleted")
	}

	return nil
}

// Delete soft-deletes a stakeholder
func (r *StakeholderRepository) Delete(ctx context.Context, stakeholderID uuid.UUID) error {
	query := `
		UPDATE program_stakeholders
		SET deleted_at = NOW()
		WHERE stakeholder_id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, stakeholderID)
	if err != nil {
		return fmt.Errorf("failed to delete stakeholder: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("stakeholder not found or already deleted")
	}

	return nil
}

// AutoLinkByName attempts to find a stakeholder by fuzzy name matching
// This is used to automatically link extracted person names to known stakeholders
func (r *StakeholderRepository) AutoLinkByName(ctx context.Context, programID uuid.UUID, personName string) (uuid.UUID, error) {
	// Try exact match first
	query := `
		SELECT stakeholder_id
		FROM program_stakeholders
		WHERE program_id = $1
		  AND LOWER(person_name) = LOWER($2)
		  AND deleted_at IS NULL
		LIMIT 1
	`

	var stakeholderID uuid.UUID
	err := r.db.QueryRowContext(ctx, query, programID, personName).Scan(&stakeholderID)
	if err == nil {
		return stakeholderID, nil
	}

	if err != sql.ErrNoRows {
		return uuid.Nil, fmt.Errorf("failed to query stakeholder: %w", err)
	}

	// Try fuzzy match using pg_trgm similarity
	query = `
		SELECT stakeholder_id
		FROM program_stakeholders
		WHERE program_id = $1
		  AND deleted_at IS NULL
		  AND similarity(person_name, $2) > 0.6
		ORDER BY similarity(person_name, $2) DESC
		LIMIT 1
	`

	err = r.db.QueryRowContext(ctx, query, programID, personName).Scan(&stakeholderID)
	if err == sql.ErrNoRows {
		return uuid.Nil, fmt.Errorf("no matching stakeholder found for: %s", personName)
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to fuzzy match stakeholder: %w", err)
	}

	return stakeholderID, nil
}

// Helper functions

func boolPtr(b bool) *bool {
	return &b
}

func sqlNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
