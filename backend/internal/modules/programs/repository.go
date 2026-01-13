package programs

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cerberus/backend/internal/platform/db"
	"github.com/google/uuid"
)

// Repository handles program data persistence
type Repository struct {
	db *db.DB
}

// NewRepository creates a new programs repository
func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

// ListProgramsWithStats retrieves all programs with aggregated statistics
func (r *Repository) ListProgramsWithStats(ctx context.Context) ([]ProgramWithStats, error) {
	query := `
		SELECT
			p.program_id, p.program_name, p.program_code, p.description,
			p.start_date, p.end_date, p.status,
			COALESCE(p.internal_organization, p.program_name) as internal_organization,
			p.created_at, p.created_by, p.updated_at, p.updated_by,
			COALESCE(a.count, 0) as artifact_count,
			COALESCE(i.count, 0) as invoice_count,
			COALESCE(r.count, 0) as risk_count
		FROM programs p
		LEFT JOIN (
			SELECT program_id, COUNT(*) as count 
			FROM artifacts 
			WHERE deleted_at IS NULL 
			GROUP BY program_id
		) a ON p.program_id = a.program_id
		LEFT JOIN (
			SELECT program_id, COUNT(*) as count 
			FROM invoices 
			WHERE deleted_at IS NULL 
			GROUP BY program_id
		) i ON p.program_id = i.program_id
		LEFT JOIN (
			SELECT program_id, COUNT(*) as count 
			FROM risks 
			WHERE deleted_at IS NULL 
			GROUP BY program_id
		) r ON p.program_id = r.program_id
		WHERE p.deleted_at IS NULL
		ORDER BY p.updated_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query programs: %w", err)
	}
	defer rows.Close()

	var programs []ProgramWithStats
	for rows.Next() {
		var p ProgramWithStats
		err := rows.Scan(
			&p.ProgramID, &p.ProgramName, &p.ProgramCode, &p.Description,
			&p.StartDate, &p.EndDate, &p.Status,
			&p.InternalOrganization,
			&p.CreatedAt, &p.CreatedBy, &p.UpdatedAt, &p.UpdatedBy,
			&p.ArtifactCount, &p.InvoiceCount, &p.RiskCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan program: %w", err)
		}
		programs = append(programs, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating programs: %w", err)
	}

	return programs, nil
}

// GetProgramByID retrieves a single program by ID with statistics
func (r *Repository) GetProgramByID(ctx context.Context, programID uuid.UUID) (*ProgramWithStats, error) {
	query := `
		SELECT
			p.program_id, p.program_name, p.program_code, p.description,
			p.start_date, p.end_date, p.status,
			COALESCE(p.internal_organization, p.program_name) as internal_organization,
			p.created_at, p.created_by, p.updated_at, p.updated_by,
			COALESCE(a.count, 0) as artifact_count,
			COALESCE(i.count, 0) as invoice_count,
			COALESCE(r.count, 0) as risk_count
		FROM programs p
		LEFT JOIN (
			SELECT program_id, COUNT(*) as count 
			FROM artifacts 
			WHERE deleted_at IS NULL 
			GROUP BY program_id
		) a ON p.program_id = a.program_id
		LEFT JOIN (
			SELECT program_id, COUNT(*) as count 
			FROM invoices 
			WHERE deleted_at IS NULL 
			GROUP BY program_id
		) i ON p.program_id = i.program_id
		LEFT JOIN (
			SELECT program_id, COUNT(*) as count 
			FROM risks 
			WHERE deleted_at IS NULL 
			GROUP BY program_id
		) r ON p.program_id = r.program_id
		WHERE p.program_id = $1 AND p.deleted_at IS NULL
	`

	var p ProgramWithStats
	err := r.db.QueryRowContext(ctx, query, programID).Scan(
		&p.ProgramID, &p.ProgramName, &p.ProgramCode, &p.Description,
		&p.StartDate, &p.EndDate, &p.Status,
		&p.InternalOrganization,
		&p.CreatedAt, &p.CreatedBy, &p.UpdatedAt, &p.UpdatedBy,
		&p.ArtifactCount, &p.InvoiceCount, &p.RiskCount,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("program not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query program: %w", err)
	}

	return &p, nil
}

// CreateProgram inserts a new program into the database
func (r *Repository) CreateProgram(ctx context.Context, program *Program) error {
	query := `
		INSERT INTO programs (
			program_id, program_name, program_code, description,
			start_date, end_date, status, created_at, created_by, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(ctx, query,
		program.ProgramID, program.ProgramName, program.ProgramCode, program.Description,
		program.StartDate, program.EndDate, program.Status,
		program.CreatedAt, program.CreatedBy, program.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create program: %w", err)
	}

	return nil
}

// UpdateProgram updates an existing program
func (r *Repository) UpdateProgram(ctx context.Context, program *Program) error {
	query := `
		UPDATE programs
		SET
			program_name = COALESCE($2, program_name),
			description = COALESCE($3, description),
			start_date = COALESCE($4, start_date),
			end_date = COALESCE($5, end_date),
			status = COALESCE($6, status),
			internal_organization = COALESCE($7, internal_organization),
			updated_at = $8,
			updated_by = $9
		WHERE program_id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query,
		program.ProgramID, program.ProgramName, program.Description,
		program.StartDate, program.EndDate, program.Status,
		program.InternalOrganization,
		program.UpdatedAt, program.UpdatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to update program: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("program not found")
	}

	return nil
}

// ProgramCodeExists checks if a program code already exists
func (r *Repository) ProgramCodeExists(ctx context.Context, code string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM programs WHERE program_code = $1 AND deleted_at IS NULL)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, code).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check program code: %w", err)
	}

	return exists, nil
}
