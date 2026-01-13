package programs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cerberus/backend/internal/platform/db"
	"github.com/google/uuid"
)

// ConfigService handles program configuration operations
type ConfigService struct {
	db *db.DB
}

// NewConfigService creates a new config service
func NewConfigService(database *db.DB) *ConfigService {
	return &ConfigService{db: database}
}

// GetProgram retrieves a program by ID with its configuration
func (s *ConfigService) GetProgram(ctx context.Context, programID uuid.UUID) (*Program, error) {
	query := `
		SELECT program_id, program_name, program_code, description,
		       start_date, end_date, status,
		       COALESCE(internal_organization, program_name) as internal_organization,
		       configuration,
		       created_at, created_by, updated_at, updated_by, deleted_at
		FROM programs
		WHERE program_id = $1 AND deleted_at IS NULL
	`

	var p Program
	var configJSON []byte

	err := s.db.QueryRowContext(ctx, query, programID).Scan(
		&p.ProgramID,
		&p.ProgramName,
		&p.ProgramCode,
		&p.Description,
		&p.StartDate,
		&p.EndDate,
		&p.Status,
		&p.InternalOrganization,
		&configJSON,
		&p.CreatedAt,
		&p.CreatedBy,
		&p.UpdatedAt,
		&p.UpdatedBy,
		&p.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("program not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get program: %w", err)
	}

	// Parse configuration JSON
	if err := json.Unmarshal(configJSON, &p.Configuration); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	return &p, nil
}

// GetProgramConfig retrieves just the configuration for a program
func (s *ConfigService) GetProgramConfig(ctx context.Context, programID uuid.UUID) (*ProgramConfig, error) {
	query := `
		SELECT configuration
		FROM programs
		WHERE program_id = $1 AND deleted_at IS NULL
	`

	var configJSON []byte
	err := s.db.QueryRowContext(ctx, query, programID).Scan(&configJSON)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("program not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get program config: %w", err)
	}

	var config ProgramConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	return &config, nil
}

// UpdateProgramConfig updates the JSONB configuration for a program
func (s *ConfigService) UpdateProgramConfig(ctx context.Context, programID uuid.UUID, req UpdateConfigRequest) error {
	// First, get current config
	currentConfig, err := s.GetProgramConfig(ctx, programID)
	if err != nil {
		return fmt.Errorf("failed to get current config: %w", err)
	}

	// Apply updates (merge with existing config)
	if req.Company != nil {
		currentConfig.Company = *req.Company
	}
	if req.Taxonomy != nil {
		currentConfig.Taxonomy = *req.Taxonomy
	}
	if req.Vendors != nil {
		currentConfig.Vendors = *req.Vendors
	}

	// Serialize to JSON
	configJSON, err := json.Marshal(currentConfig)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	// Update in database
	query := `
		UPDATE programs
		SET configuration = $1, updated_at = NOW()
		WHERE program_id = $2 AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, configJSON, programID)
	if err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("program not found or already deleted")
	}

	return nil
}

// FormatCompanyForAI formats company configuration for AI context
func (s *ConfigService) FormatCompanyForAI(config *ProgramConfig) string {
	if config.Company.Name == "" {
		return ""
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("Company: %s", config.Company.Name))

	if config.Company.FullLegalName != "" && config.Company.FullLegalName != config.Company.Name {
		parts = append(parts, fmt.Sprintf("(Legal Name: %s)", config.Company.FullLegalName))
	}

	if len(config.Company.Aliases) > 0 {
		parts = append(parts, fmt.Sprintf("Aliases: %s", strings.Join(config.Company.Aliases, ", ")))
	}

	return strings.Join(parts, " ")
}

// FormatTaxonomyForAI formats taxonomy configuration for AI context
func (s *ConfigService) FormatTaxonomyForAI(config *ProgramConfig) string {
	var parts []string

	if len(config.Taxonomy.RiskCategories) > 0 {
		parts = append(parts, fmt.Sprintf("Risk Categories: %s", strings.Join(config.Taxonomy.RiskCategories, ", ")))
	}

	if len(config.Taxonomy.SpendCategories) > 0 {
		parts = append(parts, fmt.Sprintf("Spend Categories: %s", strings.Join(config.Taxonomy.SpendCategories, ", ")))
	}

	if len(config.Taxonomy.ProjectPhases) > 0 {
		parts = append(parts, fmt.Sprintf("Project Phases: %s", strings.Join(config.Taxonomy.ProjectPhases, ", ")))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "\n")
}

// FormatVendorsForAI formats vendor list for AI context
func (s *ConfigService) FormatVendorsForAI(config *ProgramConfig) string {
	if len(config.Vendors) == 0 {
		return ""
	}

	var vendorNames []string
	for _, vendor := range config.Vendors {
		vendorNames = append(vendorNames, fmt.Sprintf("%s (%s)", vendor.Name, vendor.Type))
	}

	return strings.Join(vendorNames, ", ")
}

// GetVendorNames extracts a list of vendor names from configuration
func (s *ConfigService) GetVendorNames(config *ProgramConfig) []string {
	var names []string
	for _, vendor := range config.Vendors {
		names = append(names, vendor.Name)
	}
	return names
}

// IsKnownVendor checks if a vendor name matches any configured vendor
func (s *ConfigService) IsKnownVendor(config *ProgramConfig, vendorName string) bool {
	lowerName := strings.ToLower(vendorName)
	for _, vendor := range config.Vendors {
		if strings.ToLower(vendor.Name) == lowerName {
			return true
		}
	}
	return false
}

// AddVendor adds a new vendor to the configuration
func (s *ConfigService) AddVendor(ctx context.Context, programID uuid.UUID, vendor VendorConfig) error {
	config, err := s.GetProgramConfig(ctx, programID)
	if err != nil {
		return err
	}

	// Check if vendor already exists
	for _, v := range config.Vendors {
		if strings.EqualFold(v.Name, vendor.Name) {
			return fmt.Errorf("vendor already exists: %s", vendor.Name)
		}
	}

	// Add new vendor
	config.Vendors = append(config.Vendors, vendor)

	// Update configuration
	return s.UpdateProgramConfig(ctx, programID, UpdateConfigRequest{
		Vendors: &config.Vendors,
	})
}

// RemoveVendor removes a vendor from the configuration
func (s *ConfigService) RemoveVendor(ctx context.Context, programID uuid.UUID, vendorName string) error {
	config, err := s.GetProgramConfig(ctx, programID)
	if err != nil {
		return err
	}

	// Filter out the vendor
	var updatedVendors []VendorConfig
	found := false
	for _, v := range config.Vendors {
		if !strings.EqualFold(v.Name, vendorName) {
			updatedVendors = append(updatedVendors, v)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("vendor not found: %s", vendorName)
	}

	// Update configuration
	return s.UpdateProgramConfig(ctx, programID, UpdateConfigRequest{
		Vendors: &updatedVendors,
	})
}

// ValidateConfig validates program configuration
func (s *ConfigService) ValidateConfig(config *ProgramConfig) error {
	// Validate company
	if config.Company.Name == "" {
		return fmt.Errorf("company name is required")
	}

	// Validate vendor types
	validVendorTypes := map[string]bool{
		"software_vendor": true,
		"consulting":      true,
		"contractor":      true,
		"partner":         true,
		"supplier":        true,
		"other":           true,
	}

	for _, vendor := range config.Vendors {
		if vendor.Name == "" {
			return fmt.Errorf("vendor name is required")
		}
		if vendor.Type != "" && !validVendorTypes[vendor.Type] {
			return fmt.Errorf("invalid vendor type: %s (must be one of: software_vendor, consulting, contractor, partner, supplier, other)", vendor.Type)
		}
	}

	return nil
}

// GetDefaultConfig returns a default program configuration
func (s *ConfigService) GetDefaultConfig(programName string) *ProgramConfig {
	return &ProgramConfig{
		Company: CompanyConfig{
			Name:          programName,
			FullLegalName: programName,
			Aliases:       []string{},
		},
		Taxonomy: TaxonomyConfig{
			RiskCategories:  []string{"technical", "financial", "schedule", "resource", "external"},
			SpendCategories: []string{"labor", "materials", "software", "travel", "other"},
			ProjectPhases:   []string{},
		},
		Vendors: []VendorConfig{},
	}
}
