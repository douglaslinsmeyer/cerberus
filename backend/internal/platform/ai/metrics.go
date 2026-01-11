package ai

import (
	"context"
	"fmt"

	"github.com/cerberus/backend/internal/platform/db"
	"github.com/google/uuid"
)

// DBMetricsTracker implements MetricsTracker using database storage
type DBMetricsTracker struct {
	db *db.DB
}

// NewDBMetricsTracker creates a new database-backed metrics tracker
func NewDBMetricsTracker(database *db.DB) *DBMetricsTracker {
	return &DBMetricsTracker{db: database}
}

// Track stores AI usage metrics in the database
func (t *DBMetricsTracker) Track(ctx context.Context, metrics *Metrics) error {
	query := `
		INSERT INTO ai_usage (
			program_id, module, job_type, model,
			tokens_input, tokens_output, tokens_cached, tokens_total,
			cost_usd, duration_ms, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	programID := uuid.Nil
	if metrics.ProgramID != "" {
		parsed, err := uuid.Parse(metrics.ProgramID)
		if err == nil {
			programID = parsed
		}
	}

	_, err := t.db.ExecContext(ctx, query,
		programID,
		metrics.Module,
		metrics.JobType,
		metrics.Model,
		metrics.InputTokens,
		metrics.OutputTokens,
		metrics.CachedTokens,
		metrics.TotalTokens,
		metrics.Cost,
		metrics.Duration.Milliseconds(),
		metrics.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to track metrics: %w", err)
	}

	return nil
}

// GetDailyCost retrieves total AI cost for a program today
func (t *DBMetricsTracker) GetDailyCost(ctx context.Context, programID uuid.UUID) (float64, error) {
	var cost float64
	query := `
		SELECT COALESCE(SUM(cost_usd), 0)
		FROM ai_usage
		WHERE program_id = $1
		  AND created_at >= CURRENT_DATE
	`

	err := t.db.QueryRowContext(ctx, query, programID).Scan(&cost)
	if err != nil {
		return 0, fmt.Errorf("failed to get daily cost: %w", err)
	}

	return cost, nil
}

// GetMonthlyCost retrieves total AI cost for a program this month
func (t *DBMetricsTracker) GetMonthlyCost(ctx context.Context, programID uuid.UUID) (float64, error) {
	var cost float64
	query := `
		SELECT COALESCE(SUM(cost_usd), 0)
		FROM ai_usage
		WHERE program_id = $1
		  AND created_at >= date_trunc('month', CURRENT_DATE)
	`

	err := t.db.QueryRowContext(ctx, query, programID).Scan(&cost)
	if err != nil {
		return 0, fmt.Errorf("failed to get monthly cost: %w", err)
	}

	return cost, nil
}

// GetUsageByModule retrieves AI usage broken down by module
func (t *DBMetricsTracker) GetUsageByModule(ctx context.Context, programID uuid.UUID, days int) (map[string]float64, error) {
	query := `
		SELECT module, COALESCE(SUM(cost_usd), 0) as total_cost
		FROM ai_usage
		WHERE program_id = $1
		  AND created_at >= CURRENT_DATE - $2 * INTERVAL '1 day'
		GROUP BY module
		ORDER BY total_cost DESC
	`

	rows, err := t.db.QueryContext(ctx, query, programID, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage by module: %w", err)
	}
	defer rows.Close()

	usage := make(map[string]float64)
	for rows.Next() {
		var module string
		var cost float64
		if err := rows.Scan(&module, &cost); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		usage[module] = cost
	}

	return usage, nil
}
