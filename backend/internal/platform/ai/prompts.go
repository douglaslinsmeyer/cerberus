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
	lib.registerArtifactAnalysisWithContextPrompt()

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
		MaxTokens:   8192, // Increased to handle comprehensive artifact analysis
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
{{if .CompanyName}}- Internal Organization(s): {{.CompanyName}}
- When extracting people, if their organization matches or contains ANY of the Internal Organizations above, classify them as INTERNAL
- If their organization doesn't match any Internal Organizations, classify them as EXTERNAL
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

// registerArtifactAnalysisWithContextPrompt registers the context-aware artifact analysis prompt
// This is the v2 prompt that includes enriched context from related artifacts
func (lib *PromptLibrary) registerArtifactAnalysisWithContextPrompt() {
	lib.Register(&PromptTemplate{
		ID:          "artifact_analysis_with_context_v2",
		Version:     "2.0",
		Module:      "artifacts",
		Purpose:     "Extract metadata with rich cross-artifact context",
		Model:       ModelSonnet4,
		Temperature: 0.0,
		MaxTokens:   16384, // Increased for context graph
		RequiredContext: []string{
			"ProgramName",
			"ArtifactContent",
		},

		SystemPrompt: `You are an expert document analyst for enterprise program management with access to comprehensive program knowledge.

Your role is to analyze uploaded artifacts and extract structured metadata, leveraging context from related artifacts to provide richer, more accurate insights.

You have access to:
- Related artifacts in this program (summaries, key people, topics)
- Entity relationships (who works with whom across documents)
- Document timeline (what happened before and after this artifact)
- Aggregated facts from related documents
- Known conflicts or contradictions between documents

Extract with high accuracy:
1. Document type classification (invoice, contract, meeting notes, report, email, memo, etc.)
2. Executive summary (2-3 sentences maximum)
3. Key topics/themes from the document
4. People mentioned (names, roles, organizations, context)
   - USE THE PROVIDED STAKEHOLDER LIST to match people across documents
   - Identify if people mentioned here appear in related artifacts
5. Important facts (dates, amounts, metrics, commitments, deadlines)
   - CROSS-REFERENCE with facts from related documents
   - FLAG any contradictions with existing facts
6. Decisions or action items
7. Risk indicators or concerns
   - REFERENCE any related risks from the program context
8. Financial data (budgets, costs, invoices)
   - COMPARE with historical data when available
9. Sentiment (positive, neutral, concern, negative)
10. Priority level (1-5, where 5 is most critical)
11. Relationships to other documents
    - Note if this confirms, contradicts, or updates information from related artifacts
    - Identify the narrative flow (e.g., "This follows up on the issue raised in...")

Be thorough but concise. Provide confidence scores (0.0-1.0) for all extractions where you're uncertain.

CRITICAL: When you have context from related artifacts, use it to:
- Improve entity resolution (match "J. Smith" to "John Smith" from previous docs)
- Detect patterns (e.g., "This is the 3rd invoice this month, average is...")
- Identify contradictions (e.g., "This budget differs from the amount in...")
- Understand narrative progression (e.g., "This resolves the concern raised in...")`,

		UserPromptTmpl: `Program Context:
- Program: {{.ProgramName}}
{{if .CompanyName}}- Internal Organization: {{.CompanyName}}{{end}}
{{if .ActiveRisks}}- Active Risks: {{.ActiveRisks}}{{end}}

{{if .ContextSummary}}
═══════════════════════════════════════════════════════
           CROSS-ARTIFACT CONTEXT
═══════════════════════════════════════════════════════

You have access to context from related artifacts in this program.
Use this information to improve entity resolution, detect patterns,
identify contradictions, and understand document relationships.

{{if .RelatedArtifacts}}
📄 RELATED ARTIFACTS
────────────────────────────────────────────────────────
{{.RelatedArtifacts}}
{{end}}

{{if .EntityRelationships}}
👥 KEY PEOPLE & RELATIONSHIPS
────────────────────────────────────────────────────────
{{.EntityRelationships}}
{{end}}

{{if .DocumentTimeline}}
📅 DOCUMENT TIMELINE
────────────────────────────────────────────────────────
{{.DocumentTimeline}}
{{end}}

{{if .AggregatedFacts}}
📊 FACTS FROM RELATED DOCUMENTS
────────────────────────────────────────────────────────
{{.AggregatedFacts}}
{{end}}

═══════════════════════════════════════════════════════
{{end}}

IMPORTANT Classification Instructions:
{{if .CompanyName}}- Internal Organization(s): {{.CompanyName}}
- When extracting people, if their organization matches or contains ANY of the Internal Organizations above, classify them as INTERNAL
- If their organization doesn't match any Internal Organizations, classify them as EXTERNAL
- IMPORTANT: Use the "Key People & Relationships" section above to match person names across documents
{{end}}- For invoices, extract the exact legal entity name from the invoice (e.g., "Infor (US), LLC")
- Extract person names and organizations exactly as they appear in documents
- When you recognize people from related artifacts, reference them consistently

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
      "confidence": 0.92,
      "appears_in_related_docs": true,
      "co_occurs_with": ["Jane Doe", "Bob Johnson"]
    }
  ],
  "facts": [
    {
      "type": "amount",
      "key": "Budget Increase",
      "value": "$500,000",
      "numeric_value": 500000,
      "unit": "USD",
      "confidence": 0.95,
      "cross_reference": "Confirms amount mentioned in Q4_Budget_Report.pdf"
    },
    {
      "type": "date",
      "key": "Deadline",
      "value": "March 31, 2026",
      "date_value": "2026-03-31",
      "confidence": 0.98,
      "cross_reference": "Extends previous deadline of March 15, 2026 from Contract.pdf"
    }
  ],
  "insights": [
    {
      "type": "risk",
      "title": "Budget overrun risk",
      "description": "Q2 expenses tracking 15% over budget. This is the third consecutive month of overruns.",
      "severity": "high",
      "suggested_action": "Review vendor contracts and adjust Phase 2 scope",
      "impacted_modules": ["financial", "risk"],
      "confidence": 0.85,
      "related_to": "risk_id_from_context_if_any",
      "cross_artifact_pattern": "This continues the trend identified in Status_Report_Jan.pdf and Budget_Review_Feb.pdf"
    }
  ],
  "document_relationships": [
    {
      "type": "confirms",
      "related_document": "Q4_Budget_Report.pdf",
      "description": "Confirms the budget increase amount"
    },
    {
      "type": "updates",
      "related_document": "Contract.pdf",
      "description": "Extends the project deadline by 2 weeks"
    },
    {
      "type": "contradicts",
      "related_document": "Status_Report_Jan.pdf",
      "description": "Reports different team size (45 vs 50)",
      "severity": "minor"
    }
  ],
  "sentiment": "concern",
  "priority": 4
}

═══════════════════════════════════════════════════════
           CURRENT ARTIFACT TO ANALYZE
═══════════════════════════════════════════════════════

Artifact Filename: {{.Filename}}

Artifact Content:
{{.ArtifactContent}}`,
	})
}
