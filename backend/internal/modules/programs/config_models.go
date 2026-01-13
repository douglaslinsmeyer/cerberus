package programs

// ProgramConfig represents the JSONB configuration for a program
type ProgramConfig struct {
	Company  CompanyConfig   `json:"company"`
	Taxonomy TaxonomyConfig  `json:"taxonomy"`
	Vendors  []VendorConfig  `json:"vendors"`
}

// CompanyConfig represents company information
type CompanyConfig struct {
	Name          string   `json:"name"`
	FullLegalName string   `json:"full_legal_name"`
	Aliases       []string `json:"aliases,omitempty"`
}

// TaxonomyConfig represents program taxonomy configuration
type TaxonomyConfig struct {
	RiskCategories  []string `json:"risk_categories"`
	SpendCategories []string `json:"spend_categories"`
	ProjectPhases   []string `json:"project_phases"`
}

// VendorConfig represents a configured vendor
type VendorConfig struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

// UpdateConfigRequest represents a request to update program configuration
type UpdateConfigRequest struct {
	Company  *CompanyConfig  `json:"company,omitempty"`
	Taxonomy *TaxonomyConfig `json:"taxonomy,omitempty"`
	Vendors  *[]VendorConfig `json:"vendors,omitempty"`
}
