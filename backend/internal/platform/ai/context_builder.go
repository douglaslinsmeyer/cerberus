package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/cerberus/backend/internal/modules/programs"
	"github.com/google/uuid"
)

// ContextBuilder assembles AI context from database-driven program configuration
type ContextBuilder struct {
	configService         *programs.ConfigService
	stakeholderRepository *programs.StakeholderRepository
}

// NewContextBuilder creates a new context builder
func NewContextBuilder(
	configService *programs.ConfigService,
	stakeholderRepository *programs.StakeholderRepository,
) *ContextBuilder {
	return &ContextBuilder{
		configService:         configService,
		stakeholderRepository: stakeholderRepository,
	}
}

// BuildContext assembles complete program context from database
func (cb *ContextBuilder) BuildContext(ctx context.Context, programID uuid.UUID) (*ProgramContext, error) {
	// Get program with configuration
	program, err := cb.configService.GetProgram(ctx, programID)
	if err != nil {
		// Fallback to default context if program not found
		return cb.getDefaultContext(programID), fmt.Errorf("failed to load program, using defaults: %w", err)
	}

	// Get internal stakeholders
	internalStakeholders, err := cb.stakeholderRepository.ListInternal(ctx, programID)
	if err != nil {
		// Log error but continue with empty stakeholder list
		internalStakeholders = []programs.Stakeholder{}
	}

	// Get external stakeholders
	externalStakeholders, err := cb.stakeholderRepository.ListExternal(ctx, programID)
	if err != nil {
		// Log error but continue with empty stakeholder list
		externalStakeholders = []programs.Stakeholder{}
	}

	// Format context
	programContext := &ProgramContext{
		ProgramName:     program.ProgramName,
		ProgramCode:     program.ProgramCode,
		CompanyName:     program.Configuration.Company.Name,
		CustomTaxonomy:  cb.formatTaxonomy(&program.Configuration),
		KeyStakeholders: cb.formatStakeholders(internalStakeholders, externalStakeholders),
		KnownVendors:    cb.formatVendors(&program.Configuration),
		HealthScore:     0, // TODO: Integrate with health score calculation when available
		BudgetStatus:    "", // TODO: Integrate with budget status when available
		ActiveRisks:     "", // TODO: Integrate with risk module when available
	}

	return programContext, nil
}

// BuildContextOrDefault builds context with fallback to defaults on error
func (cb *ContextBuilder) BuildContextOrDefault(ctx context.Context, programID uuid.UUID) *ProgramContext {
	programContext, err := cb.BuildContext(ctx, programID)
	if err != nil {
		// Return default context on error
		return cb.getDefaultContext(programID)
	}
	return programContext
}

// formatTaxonomy formats taxonomy configuration for AI prompts
func (cb *ContextBuilder) formatTaxonomy(config *programs.ProgramConfig) string {
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

	return strings.Join(parts, " | ")
}

// formatStakeholders formats stakeholder lists for AI prompts
func (cb *ContextBuilder) formatStakeholders(internal []programs.Stakeholder, external []programs.Stakeholder) string {
	var parts []string

	// Format internal stakeholders (key team members)
	if len(internal) > 0 {
		internalNames := []string{}
		for _, s := range internal {
			name := s.PersonName
			if s.Role.Valid && s.Role.String != "" {
				name += fmt.Sprintf(" (%s)", s.Role.String)
			}
			// Prioritize key stakeholders
			if s.EngagementLevel.Valid && (s.EngagementLevel.String == "key" || s.EngagementLevel.String == "primary") {
				internalNames = append(internalNames, name)
			}
		}
		if len(internalNames) > 0 {
			parts = append(parts, fmt.Sprintf("Internal: %s", strings.Join(internalNames, ", ")))
		}
	}

	// Format external stakeholders (vendors, partners)
	if len(external) > 0 {
		externalNames := []string{}
		for _, s := range external {
			name := s.PersonName
			if s.Organization.Valid && s.Organization.String != "" {
				name += fmt.Sprintf(" (%s)", s.Organization.String)
			}
			// Prioritize key external stakeholders
			if s.EngagementLevel.Valid && (s.EngagementLevel.String == "key" || s.EngagementLevel.String == "primary") {
				externalNames = append(externalNames, name)
			}
		}
		if len(externalNames) > 0 {
			parts = append(parts, fmt.Sprintf("External: %s", strings.Join(externalNames, ", ")))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " | ")
}

// formatVendors formats vendor list for AI prompts
func (cb *ContextBuilder) formatVendors(config *programs.ProgramConfig) string {
	if len(config.Vendors) == 0 {
		return ""
	}

	var vendorNames []string
	for _, vendor := range config.Vendors {
		vendorNames = append(vendorNames, vendor.Name)
	}

	return strings.Join(vendorNames, ", ")
}

// getDefaultContext returns a default context when database lookup fails
func (cb *ContextBuilder) getDefaultContext(programID uuid.UUID) *ProgramContext {
	return &ProgramContext{
		ProgramName:     "Default Program",
		ProgramCode:     programID.String()[:8],
		CompanyName:     "Company",
		CustomTaxonomy:  "Risk Categories: technical, financial, schedule, resource, external | Spend Categories: labor, materials, software, travel, other",
		KeyStakeholders: "",
		KnownVendors:    "",
		HealthScore:     0,
		BudgetStatus:    "",
		ActiveRisks:     "",
	}
}

// GetVendorList returns a list of known vendor names for a program
func (cb *ContextBuilder) GetVendorList(ctx context.Context, programID uuid.UUID) ([]string, error) {
	config, err := cb.configService.GetProgramConfig(ctx, programID)
	if err != nil {
		return []string{}, fmt.Errorf("failed to get program config: %w", err)
	}

	return cb.configService.GetVendorNames(config), nil
}

// IsKnownVendor checks if a vendor name is recognized for the program
func (cb *ContextBuilder) IsKnownVendor(ctx context.Context, programID uuid.UUID, vendorName string) (bool, error) {
	config, err := cb.configService.GetProgramConfig(ctx, programID)
	if err != nil {
		return false, fmt.Errorf("failed to get program config: %w", err)
	}

	return cb.configService.IsKnownVendor(config, vendorName), nil
}

// GetStakeholderByName attempts to find a stakeholder by name (for auto-linking)
func (cb *ContextBuilder) GetStakeholderByName(ctx context.Context, programID uuid.UUID, personName string) (uuid.UUID, error) {
	return cb.stakeholderRepository.AutoLinkByName(ctx, programID, personName)
}
