package ai

import (
	"bytes"
	"fmt"
	"text/template"
)

// PromptTemplate represents a versioned AI prompt template
type PromptTemplate struct {
	ID              string
	Version         string
	Module          string
	Purpose         string
	SystemPrompt    string
	UserPromptTmpl  string
	Model           string
	Temperature     float64
	MaxTokens       int
	RequiredContext []string
}

// PromptLibrary manages AI prompt templates
type PromptLibrary struct {
	templates map[string]*PromptTemplate
}

// NewPromptLibrary creates a new prompt library with default templates
func NewPromptLibrary() *PromptLibrary {
	lib := &PromptLibrary{
		templates: make(map[string]*PromptTemplate),
	}

	// Register default templates
	lib.registerArtifactAnalysisPrompt()

	return lib
}

// Get retrieves a prompt template by ID
func (lib *PromptLibrary) Get(id string) (*PromptTemplate, error) {
	tmpl, ok := lib.templates[id]
	if !ok {
		return nil, fmt.Errorf("prompt template not found: %s", id)
	}
	return tmpl, nil
}

// Register adds a new prompt template
func (lib *PromptLibrary) Register(tmpl *PromptTemplate) {
	lib.templates[tmpl.ID] = tmpl
}

// CompileUserPrompt compiles the user prompt template with variables
func (pt *PromptTemplate) CompileUserPrompt(vars map[string]interface{}) (string, error) {
	tmpl, err := template.New("prompt").Parse(pt.UserPromptTmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// registerArtifactAnalysisPrompt registers the artifact analysis prompt template
func (lib *PromptLibrary) registerArtifactAnalysisPrompt() {
	lib.Register(&PromptTemplate{
		ID:          "artifact_analysis_v1",
		Version:     "1.0",
		Module:      "artifacts",
		Purpose:     "Extract structured metadata from artifacts",
		Model:       ModelSonnet4,
		Temperature: 0.0,
		MaxTokens:   4096,
		RequiredContext: []string{
			"ProgramName",
			"ArtifactContent",
		},

		SystemPrompt: `You are an expert document analyst for enterprise program management.

Your role is to analyze uploaded artifacts and extract structured metadata to build program knowledge.

Extract with high accuracy:
1. Document type classification (invoice, contract, meeting notes, report, email, memo, etc.)
2. Executive summary (2-3 sentences maximum)
3. Key topics/themes from the document
4. People mentioned (names, roles, organizations, context)
5. Important facts (dates, amounts, metrics, commitments, deadlines)
6. Decisions or action items
7. Risk indicators or concerns
8. Financial data (budgets, costs, invoices)
9. Sentiment (positive, neutral, concern, negative)
10. Priority level (1-5, where 5 is most critical)

Be thorough but concise. Provide confidence scores (0.0-1.0) for all extractions where you're uncertain.`,

		UserPromptTmpl: `Program Context:
- Program: {{.ProgramName}}
{{if .CompanyName}}- Internal Organization: {{.CompanyName}}{{end}}
{{if .ActiveRisks}}- Active Risks: {{.ActiveRisks}}{{end}}

Task: Analyze this artifact and extract structured metadata.

IMPORTANT Classification Instructions:
{{if .CompanyName}}- When extracting people, if their organization matches or contains "{{.CompanyName}}", classify them as INTERNAL
- If their organization is different from "{{.CompanyName}}", classify them as EXTERNAL
{{end}}- For invoices, extract the exact legal entity name from the invoice (e.g., "Infor (US), LLC")
- Extract person names and organizations exactly as they appear in documents

Output as JSON matching this exact schema:
{
  "document_type": "invoice",
  "document_type_confidence": 0.98,
  "summary": "2-3 sentence executive summary",
  "key_topics": [
    {"topic": "budget", "confidence": 0.95},
    {"topic": "risk", "confidence": 0.88}
  ],
  "persons_mentioned": [
    {
      "name": "John Smith",
      "role": "Program Director",
      "organization": "Acme Corp",
      "context": "Mentioned as decision maker for budget approval",
      "confidence": 0.92
    }
  ],
  "facts": [
    {
      "type": "amount",
      "key": "Budget Increase",
      "value": "$500,000",
      "numeric_value": 500000,
      "unit": "USD",
      "confidence": 0.95
    },
    {
      "type": "date",
      "key": "Deadline",
      "value": "March 31, 2026",
      "date_value": "2026-03-31",
      "confidence": 0.98
    }
  ],
  "insights": [
    {
      "type": "risk",
      "title": "Budget overrun risk",
      "description": "Q2 expenses tracking 15% over budget",
      "severity": "high",
      "suggested_action": "Review vendor contracts and adjust Phase 2 scope",
      "impacted_modules": ["financial", "risk"],
      "confidence": 0.85
    }
  ],
  "sentiment": "concern",
  "priority": 4
}

Artifact Filename: {{.Filename}}
Artifact Content:
{{.ArtifactContent}}`,
	})
}

// ProgramContext represents context about a program for AI prompts
type ProgramContext struct {
	ProgramName     string
	ProgramCode     string
	CompanyName     string
	CustomTaxonomy  string
	KeyStakeholders string
	KnownVendors    string
	ActiveRisks     string
	BudgetStatus    string
	HealthScore     int
}

// ToPromptString converts program context to a formatted string for prompts
func (pc *ProgramContext) ToPromptString() string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("Program: %s", pc.ProgramName))
	if pc.ProgramCode != "" {
		buf.WriteString(fmt.Sprintf(" (%s)", pc.ProgramCode))
	}
	buf.WriteString("\n")

	if pc.CompanyName != "" {
		buf.WriteString(fmt.Sprintf("Company: %s\n", pc.CompanyName))
	}

	if pc.KnownVendors != "" {
		buf.WriteString(fmt.Sprintf("Known Vendors: %s\n", pc.KnownVendors))
	}

	if pc.HealthScore > 0 {
		buf.WriteString(fmt.Sprintf("Health Score: %d\n", pc.HealthScore))
	}

	if pc.BudgetStatus != "" {
		buf.WriteString(fmt.Sprintf("Budget Status: %s\n", pc.BudgetStatus))
	}

	if pc.ActiveRisks != "" {
		buf.WriteString(fmt.Sprintf("Active Risks: %s\n", pc.ActiveRisks))
	}

	return buf.String()
}
