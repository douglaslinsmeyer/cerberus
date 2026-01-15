package artifacts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cerberus/backend/internal/platform/ai"
	"github.com/google/uuid"
)

// AIAnalyzer handles AI-powered artifact analysis
type AIAnalyzer struct {
	client             *ai.Client
	prompts            *ai.PromptLibrary
	repo               RepositoryInterface
	contextGraphBuilder *ContextGraphBuilder
	useEnrichedContext bool
}

// NewAIAnalyzer creates a new AI analyzer
func NewAIAnalyzer(client *ai.Client, repo RepositoryInterface) *AIAnalyzer {
	return &AIAnalyzer{
		client:             client,
		prompts:            ai.NewPromptLibrary(),
		repo:               repo,
		useEnrichedContext: false, // Disabled by default
	}
}

// NewAIAnalyzerWithContext creates a new AI analyzer with enriched context support
func NewAIAnalyzerWithContext(client *ai.Client, repo RepositoryInterface, contextBuilder *ContextGraphBuilder) *AIAnalyzer {
	return &AIAnalyzer{
		client:              client,
		prompts:             ai.NewPromptLibrary(),
		repo:                repo,
		contextGraphBuilder: contextBuilder,
		useEnrichedContext:  true, // Enabled when context builder is provided
	}
}

// SetContextGraphBuilder enables enriched context and sets the context builder
func (a *AIAnalyzer) SetContextGraphBuilder(builder *ContextGraphBuilder) {
	a.contextGraphBuilder = builder
	a.useEnrichedContext = true
}

// EnableEnrichedContext enables or disables enriched context
func (a *AIAnalyzer) EnableEnrichedContext(enabled bool) {
	a.useEnrichedContext = enabled
}

// AnalysisResult contains all extracted metadata from AI analysis
type AnalysisResult struct {
	DocumentType           string
	DocumentTypeConfidence float64
	Summary                ArtifactSummary
	Topics                 []Topic
	Persons                []Person
	Facts                  []Fact
	Insights               []Insight
	ProcessingTime         time.Duration
	TokensUsed             int
	Cost                   float64
}

// AIExtractionResponse matches the JSON schema from the prompt
type AIExtractionResponse struct {
	DocumentType           string  `json:"document_type"`
	DocumentTypeConfidence float64 `json:"document_type_confidence"`
	Summary                string  `json:"summary"`
	KeyTopics              []struct {
		Topic      string  `json:"topic"`
		Confidence float64 `json:"confidence"`
	} `json:"key_topics"`
	PersonsMentioned []struct {
		Name         string  `json:"name"`
		Role         string  `json:"role"`
		Organization string  `json:"organization"`
		Context      string  `json:"context"`
		Confidence   float64 `json:"confidence"`
	} `json:"persons_mentioned"`
	Facts []struct {
		Type         string  `json:"type"`
		Key          string  `json:"key"`
		Value        string  `json:"value"`
		NumericValue float64 `json:"numeric_value,omitempty"`
		DateValue    string  `json:"date_value,omitempty"`
		Unit         string  `json:"unit,omitempty"`
		Confidence   float64 `json:"confidence"`
	} `json:"facts"`
	Insights []struct {
		Type            string   `json:"type"`
		Title           string   `json:"title"`
		Description     string   `json:"description"`
		Severity        string   `json:"severity"`
		SuggestedAction string   `json:"suggested_action"`
		ImpactedModules []string `json:"impacted_modules"`
		Confidence      float64  `json:"confidence"`
	} `json:"insights"`
	Sentiment string `json:"sentiment"`
	Priority  int    `json:"priority"`
}

// AnalyzeArtifact performs AI analysis on an artifact
func (a *AIAnalyzer) AnalyzeArtifact(ctx context.Context, artifact *Artifact, programContext *ai.ProgramContext) (*AnalysisResult, error) {
	startTime := time.Now()

	// Determine which prompt to use and whether to build enriched context
	var promptTmpl *ai.PromptTemplate
	var err error
	var enrichedContext *EnrichedContext

	if a.useEnrichedContext && a.contextGraphBuilder != nil {
		// Use context-aware prompt
		promptTmpl, err = a.prompts.Get("artifact_analysis_with_context_v2")
		if err != nil {
			// Fall back to v1 if v2 not available
			fmt.Printf("Warning: Failed to get v2 prompt, falling back to v1: %v\n", err)
			promptTmpl, err = a.prompts.Get("artifact_analysis_v1")
			if err != nil {
				return nil, fmt.Errorf("failed to get prompt template: %w", err)
			}
		} else {
			// Build enriched context
			fmt.Printf("Building enriched context for artifact %s...\n", artifact.ArtifactID)
			enrichedContext, err = a.contextGraphBuilder.BuildEnrichedContext(ctx, artifact, 4000)
			if err != nil {
				fmt.Printf("Warning: Failed to build enriched context: %v\n", err)
				// Continue without enriched context
			} else {
				fmt.Printf("Enriched context built: %d tokens, %d related artifacts\n",
					enrichedContext.EstimatedTokens, len(enrichedContext.RelatedArtifacts))
			}
		}
	} else {
		// Use standard prompt
		promptTmpl, err = a.prompts.Get("artifact_analysis_v1")
		if err != nil {
			return nil, fmt.Errorf("failed to get prompt template: %w", err)
		}
	}

	// Compile user prompt with variables
	vars := map[string]interface{}{
		"ProgramName":     programContext.ProgramName,
		"CompanyName":     programContext.CompanyName,
		"CustomTaxonomy":  programContext.CustomTaxonomy,
		"KeyStakeholders": programContext.KeyStakeholders,
		"ActiveRisks":     programContext.ActiveRisks,
		"Filename":        artifact.Filename,
		"ArtifactContent": artifact.RawContent.String,
	}

	// Add enriched context variables if available
	if enrichedContext != nil {
		vars["ContextSummary"] = a.formatContextSummary(enrichedContext)
		vars["RelatedArtifacts"] = a.formatRelatedArtifacts(enrichedContext)
		vars["EntityRelationships"] = a.formatEntityRelationships(enrichedContext)
		vars["DocumentTimeline"] = a.formatDocumentTimeline(enrichedContext)
		vars["AggregatedFacts"] = a.formatAggregatedFacts(enrichedContext)
	}

	userPrompt, err := promptTmpl.CompileUserPrompt(vars)
	if err != nil {
		return nil, fmt.Errorf("failed to compile prompt: %w", err)
	}

	// Prepare static context for caching
	staticContext := programContext.ToPromptString()

	// Call Claude API with caching
	resp, err := a.client.RequestWithContext(
		ctx,
		promptTmpl.Model,
		promptTmpl.SystemPrompt,
		staticContext,
		userPrompt,
		promptTmpl.MaxTokens,
	)

	if err != nil {
		return nil, fmt.Errorf("Claude API request failed: %w", err)
	}

	// Parse JSON response
	responseText := resp.GetExtractedText()

	// Strip markdown code blocks if present
	responseText = stripMarkdownCodeBlocks(responseText)

	// Log the response for debugging
	fmt.Printf("AI Response (first 500 chars): %s\n", truncateString(responseText, 500))

	var extraction AIExtractionResponse
	if err := json.Unmarshal([]byte(responseText), &extraction); err != nil {
		fmt.Printf("Failed to parse AI response. Full response length: %d\n", len(responseText))
		fmt.Printf("Full response: %s\n", responseText)
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// Convert to domain models
	result := &AnalysisResult{
		DocumentType:           extraction.DocumentType,
		DocumentTypeConfidence: extraction.DocumentTypeConfidence,
		ProcessingTime:         time.Since(startTime),
		TokensUsed:             resp.Usage.InputTokens + resp.Usage.OutputTokens,
		Cost:                   ai.NewCostCalculator().CalculateCost(resp.Model, &resp.Usage),
	}

	// Convert summary
	result.Summary = ArtifactSummary{
		SummaryID:        uuid.New(),
		ArtifactID:       artifact.ArtifactID,
		ExecutiveSummary: extraction.Summary,
		KeyTakeaways:     []string{}, // Could extract from summary
		Sentiment:        sql.NullString{String: extraction.Sentiment, Valid: extraction.Sentiment != ""},
		Priority:         sql.NullInt32{Int32: int32(extraction.Priority), Valid: extraction.Priority > 0},
		ConfidenceScore:  sql.NullFloat64{Float64: 0.9, Valid: true}, // Overall confidence
		AIModel:          sql.NullString{String: resp.Model, Valid: true},
		CreatedAt:        time.Now(),
	}

	// Convert topics
	for _, topic := range extraction.KeyTopics {
		result.Topics = append(result.Topics, Topic{
			TopicID:         uuid.New(),
			ArtifactID:      artifact.ArtifactID,
			TopicName:       topic.Topic,
			ConfidenceScore: topic.Confidence,
			TopicLevel:      1,
			ExtractedAt:     time.Now(),
		})
	}

	// Convert persons
	for _, person := range extraction.PersonsMentioned {
		contextSnippets, _ := json.Marshal([]string{person.Context})
		result.Persons = append(result.Persons, Person{
			PersonID:           uuid.New(),
			ArtifactID:         artifact.ArtifactID,
			PersonName:         person.Name,
			PersonRole:         sql.NullString{String: person.Role, Valid: person.Role != ""},
			PersonOrganization: sql.NullString{String: person.Organization, Valid: person.Organization != ""},
			MentionCount:       1,
			ContextSnippets:    contextSnippets,
			ConfidenceScore:    sql.NullFloat64{Float64: person.Confidence, Valid: true},
			ExtractedAt:        time.Now(),
		})
	}

	// Convert facts
	for _, fact := range extraction.Facts {
		f := Fact{
			FactID:          uuid.New(),
			ArtifactID:      artifact.ArtifactID,
			FactType:        fact.Type,
			FactKey:         fact.Key,
			FactValue:       fact.Value,
			Unit:            sql.NullString{String: fact.Unit, Valid: fact.Unit != ""},
			ConfidenceScore: sql.NullFloat64{Float64: fact.Confidence, Valid: true},
			ExtractedAt:     time.Now(),
		}

		// Add normalized values
		if fact.NumericValue != 0 {
			f.NormalizedValueNumeric = sql.NullFloat64{Float64: fact.NumericValue, Valid: true}
		}
		if fact.DateValue != "" {
			// Parse date
			if parsedDate, err := time.Parse("2006-01-02", fact.DateValue); err == nil {
				f.NormalizedValueDate = sql.NullTime{Time: parsedDate, Valid: true}
			}
		}

		result.Facts = append(result.Facts, f)
	}

	// Convert insights
	for _, insight := range extraction.Insights {
		result.Insights = append(result.Insights, Insight{
			InsightID:       uuid.New(),
			ArtifactID:      artifact.ArtifactID,
			InsightType:     insight.Type,
			Title:           insight.Title,
			Description:     insight.Description,
			Severity:        sql.NullString{String: insight.Severity, Valid: insight.Severity != ""},
			SuggestedAction: sql.NullString{String: insight.SuggestedAction, Valid: insight.SuggestedAction != ""},
			ImpactedModules: insight.ImpactedModules,
			ConfidenceScore: sql.NullFloat64{Float64: insight.Confidence, Valid: true},
			IsDismissed:     false,
			ExtractedAt:     time.Now(),
		})
	}

	return result, nil
}

// StoreAnalysisResults saves all analysis results to the database
func (a *AIAnalyzer) StoreAnalysisResults(ctx context.Context, artifactID uuid.UUID, result *AnalysisResult) error {
	// Save summary
	if err := a.repo.SaveSummary(ctx, &result.Summary); err != nil {
		return fmt.Errorf("failed to save summary: %w", err)
	}

	// Save topics
	if err := a.repo.SaveTopics(ctx, result.Topics); err != nil {
		return fmt.Errorf("failed to save topics: %w", err)
	}

	// Save persons
	if err := a.repo.SavePersons(ctx, result.Persons); err != nil {
		return fmt.Errorf("failed to save persons: %w", err)
	}

	// Save facts
	if err := a.repo.SaveFacts(ctx, result.Facts); err != nil {
		return fmt.Errorf("failed to save facts: %w", err)
	}

	// Save insights
	if err := a.repo.SaveInsights(ctx, result.Insights); err != nil {
		return fmt.Errorf("failed to save insights: %w", err)
	}

	// Update artifact category with document type
	if result.DocumentType != "" {
		query := `
			UPDATE artifacts
			SET artifact_category = $1
			WHERE artifact_id = $2
		`
		_, err := a.repo.ExecContext(ctx, query, result.DocumentType, artifactID)
		if err != nil {
			return fmt.Errorf("failed to update artifact category: %w", err)
		}
	}

	// Update artifact status to completed
	if err := a.repo.UpdateStatus(ctx, artifactID, "completed"); err != nil {
		return fmt.Errorf("failed to update artifact status: %w", err)
	}

	return nil
}

// ProcessArtifact performs complete analysis: analyze + store results
func (a *AIAnalyzer) ProcessArtifact(ctx context.Context, artifact *Artifact, programContext *ai.ProgramContext) error {
	// Update status to processing
	if err := a.repo.UpdateStatus(ctx, artifact.ArtifactID, "processing"); err != nil {
		return fmt.Errorf("failed to update status to processing: %w", err)
	}

	// Analyze artifact
	result, err := a.AnalyzeArtifact(ctx, artifact, programContext)
	if err != nil {
		// Mark as failed
		a.repo.UpdateStatus(ctx, artifact.ArtifactID, "failed")
		return fmt.Errorf("analysis failed: %w", err)
	}

	// Store results
	if err := a.StoreAnalysisResults(ctx, artifact.ArtifactID, result); err != nil {
		// Mark as failed
		a.repo.UpdateStatus(ctx, artifact.ArtifactID, "failed")
		return fmt.Errorf("failed to store results: %w", err)
	}

	return nil
}

// stripMarkdownCodeBlocks removes markdown code block wrappers from text
func stripMarkdownCodeBlocks(text string) string {
	// Remove ```json\n and ``` markers
	text = strings.TrimSpace(text)

	// Check if wrapped in code blocks
	if strings.HasPrefix(text, "```") {
		// Find the first newline after ```
		firstNewline := strings.Index(text, "\n")
		if firstNewline > 0 {
			text = text[firstNewline+1:]
		}

		// Remove trailing ```
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}

	return text
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ============================================================================
// Enriched Context Formatting Methods
// ============================================================================

// formatContextSummary creates a summary line about the available context
func (a *AIAnalyzer) formatContextSummary(ctx *EnrichedContext) string {
	if ctx == nil {
		return ""
	}

	summary := fmt.Sprintf("Context includes: %d related artifacts", len(ctx.RelatedArtifacts))

	if ctx.EntityGraph != nil && len(ctx.EntityGraph.KeyPeople) > 0 {
		summary += fmt.Sprintf(", %d key people", len(ctx.EntityGraph.KeyPeople))
	}

	if ctx.Timeline != nil {
		totalDocs := len(ctx.Timeline.PrecedingArtifacts) + len(ctx.Timeline.FollowingArtifacts)
		if totalDocs > 0 {
			summary += fmt.Sprintf(", timeline of %d documents", totalDocs)
		}
	}

	if ctx.AggregatedFacts != nil {
		totalFacts := len(ctx.AggregatedFacts.FinancialFacts) +
			len(ctx.AggregatedFacts.DateFacts) +
			len(ctx.AggregatedFacts.MetricFacts)
		if totalFacts > 0 {
			summary += fmt.Sprintf(", %d aggregated facts", totalFacts)
		}
		if len(ctx.AggregatedFacts.Conflicts) > 0 {
			summary += fmt.Sprintf(", %d fact conflicts", len(ctx.AggregatedFacts.Conflicts))
		}
	}

	return summary
}

// formatRelatedArtifacts formats the related artifacts for the prompt
func (a *AIAnalyzer) formatRelatedArtifacts(ctx *EnrichedContext) string {
	if ctx == nil || len(ctx.RelatedArtifacts) == 0 {
		return ""
	}

	var output strings.Builder

	for i, artifact := range ctx.RelatedArtifacts {
		output.WriteString(fmt.Sprintf("[%d] \"%s\"", i+1, artifact.Filename))

		if artifact.Category != "" {
			output.WriteString(fmt.Sprintf(" (%s)", artifact.Category))
		}

		// Add relative time
		relativeTime := formatRelativeTime(artifact.UploadedAt)
		output.WriteString(fmt.Sprintf(", uploaded %s", relativeTime))
		output.WriteString("\n")

		// Add summary if available
		if artifact.ExecutiveSummary != "" {
			summary := artifact.ExecutiveSummary
			if len(summary) > 200 {
				summary = summary[:197] + "..."
			}
			output.WriteString(fmt.Sprintf("    Summary: %s\n", summary))
		}

		// Add shared people if any
		if len(artifact.SharedPersonIDs) > 0 && len(artifact.MentionedPeople) > 0 {
			peopleList := ""
			maxPeople := 3
			for j, person := range artifact.MentionedPeople {
				if j >= maxPeople {
					peopleList += fmt.Sprintf(", +%d more", len(artifact.MentionedPeople)-maxPeople)
					break
				}
				if j > 0 {
					peopleList += ", "
				}
				peopleList += person
			}
			output.WriteString(fmt.Sprintf("    Shared People: %s\n", peopleList))
		}

		// Add topics if available
		if len(artifact.Topics) > 0 {
			topicList := ""
			maxTopics := 4
			for j, topic := range artifact.Topics {
				if j >= maxTopics {
					topicList += fmt.Sprintf(", +%d more", len(artifact.Topics)-maxTopics)
					break
				}
				if j > 0 {
					topicList += ", "
				}
				topicList += topic
			}
			output.WriteString(fmt.Sprintf("    Topics: %s\n", topicList))
		}

		// Add relevance score
		output.WriteString(fmt.Sprintf("    Relevance Score: %.2f\n", artifact.TotalScore))
		output.WriteString("\n")
	}

	return output.String()
}

// formatEntityRelationships formats the entity graph for the prompt
func (a *AIAnalyzer) formatEntityRelationships(ctx *EnrichedContext) string {
	if ctx == nil || ctx.EntityGraph == nil || len(ctx.EntityGraph.KeyPeople) == 0 {
		return ""
	}

	var output strings.Builder
	output.WriteString("Key people across related documents:\n\n")

	for _, person := range ctx.EntityGraph.KeyPeople {
		output.WriteString(fmt.Sprintf("- %s", person.Name))

		if person.Role != "" {
			output.WriteString(fmt.Sprintf(" (%s", person.Role))
			if person.Organization != "" {
				output.WriteString(fmt.Sprintf(", %s", person.Organization))
			}
			output.WriteString(")")
		}

		if person.Classification != "" {
			output.WriteString(fmt.Sprintf(" [%s]", person.Classification))
		}

		output.WriteString("\n")

		if person.MentionCount > 0 {
			output.WriteString(fmt.Sprintf("  → Mentioned %d times", person.MentionCount))
			if person.ArtifactCount > 0 {
				output.WriteString(fmt.Sprintf(" across %d documents", person.ArtifactCount))
			}
			output.WriteString("\n")
		}

		if len(person.CoOccursWith) > 0 {
			coOccurs := strings.Join(person.CoOccursWith, ", ")
			output.WriteString(fmt.Sprintf("  → Frequently appears with: %s\n", coOccurs))
		}

		if person.RecentContext != "" {
			output.WriteString(fmt.Sprintf("  → Recent context: %s\n", person.RecentContext))
		}

		output.WriteString("\n")
	}

	return output.String()
}

// formatDocumentTimeline formats the timeline for the prompt
func (a *AIAnalyzer) formatDocumentTimeline(ctx *EnrichedContext) string {
	if ctx == nil || ctx.Timeline == nil {
		return ""
	}

	if len(ctx.Timeline.PrecedingArtifacts) == 0 && len(ctx.Timeline.FollowingArtifacts) == 0 {
		return ""
	}

	var output strings.Builder

	if len(ctx.Timeline.PrecedingArtifacts) > 0 {
		output.WriteString("Before Current Artifact:\n")
		for _, entry := range ctx.Timeline.PrecedingArtifacts {
			output.WriteString(fmt.Sprintf("  [%s] %s", entry.RelativeTime, entry.Filename))
			if entry.Category != "" {
				output.WriteString(fmt.Sprintf(" (%s)", entry.Category))
			}
			output.WriteString("\n")

			if entry.Summary != "" {
				output.WriteString(fmt.Sprintf("    %s\n", entry.Summary))
			}
		}
		output.WriteString("\n")
	}

	output.WriteString("Current Artifact:\n")
	output.WriteString("  [Today] [ANALYZING THIS DOCUMENT]\n\n")

	if len(ctx.Timeline.FollowingArtifacts) > 0 {
		output.WriteString("After Current Artifact:\n")
		for _, entry := range ctx.Timeline.FollowingArtifacts {
			output.WriteString(fmt.Sprintf("  [%s] %s", entry.RelativeTime, entry.Filename))
			if entry.Category != "" {
				output.WriteString(fmt.Sprintf(" (%s)", entry.Category))
			}
			output.WriteString("\n")

			if entry.Summary != "" {
				output.WriteString(fmt.Sprintf("    %s\n", entry.Summary))
			}
		}
	}

	return output.String()
}

// formatAggregatedFacts formats the aggregated facts for the prompt
func (a *AIAnalyzer) formatAggregatedFacts(ctx *EnrichedContext) string {
	if ctx == nil || ctx.AggregatedFacts == nil {
		return ""
	}

	factsCtx := ctx.AggregatedFacts
	if len(factsCtx.FinancialFacts) == 0 && len(factsCtx.DateFacts) == 0 &&
		len(factsCtx.MetricFacts) == 0 && len(factsCtx.Conflicts) == 0 {
		return ""
	}

	var output strings.Builder

	if len(factsCtx.FinancialFacts) > 0 {
		output.WriteString("Financial Facts:\n")
		for _, fact := range factsCtx.FinancialFacts {
			sourceList := ""
			if len(fact.SourceArtifacts) > 3 {
				sourceList = fmt.Sprintf("%s, %s, +%d more",
					fact.SourceArtifacts[0], fact.SourceArtifacts[1],
					len(fact.SourceArtifacts)-2)
			} else {
				sourceList = strings.Join(fact.SourceArtifacts, ", ")
			}

			output.WriteString(fmt.Sprintf("- %s: %s (from %s, confidence: %.2f)\n",
				fact.FactKey, fact.FactValue, sourceList, fact.ConfidenceScore))
		}
		output.WriteString("\n")
	}

	if len(factsCtx.DateFacts) > 0 {
		output.WriteString("Date Facts:\n")
		for _, fact := range factsCtx.DateFacts {
			output.WriteString(fmt.Sprintf("- %s: %s (mentioned %d times)\n",
				fact.FactKey, fact.FactValue, fact.OccurrenceCount))
		}
		output.WriteString("\n")
	}

	if len(factsCtx.MetricFacts) > 0 {
		output.WriteString("Metric Facts:\n")
		for _, fact := range factsCtx.MetricFacts {
			output.WriteString(fmt.Sprintf("- %s: %s (confidence: %.2f)\n",
				fact.FactKey, fact.FactValue, fact.ConfidenceScore))
		}
		output.WriteString("\n")
	}

	if len(factsCtx.Conflicts) > 0 {
		output.WriteString("⚠️ CONFLICTS DETECTED:\n")
		for _, conflict := range factsCtx.Conflicts {
			output.WriteString(fmt.Sprintf("- \"%s\" has %d conflicting values:\n",
				conflict.FactKey, len(conflict.ConflictingValues)))

			for _, value := range conflict.ConflictingValues {
				output.WriteString(fmt.Sprintf("  • %s (from %s, confidence: %.2f)\n",
					value.Value, value.SourceFilename, value.ConfidenceScore))
			}
			output.WriteString("\n")
		}
	}

	return output.String()
}

// formatRelativeTime formats a timestamp as relative time
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)
	days := int(diff.Hours() / 24)
	weeks := days / 7
	months := days / 30

	if days == 0 {
		return "today"
	} else if days == 1 {
		return "yesterday"
	} else if days < 7 {
		return fmt.Sprintf("%d days ago", days)
	} else if weeks == 1 {
		return "1 week ago"
	} else if weeks < 4 {
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if months == 1 {
		return "1 month ago"
	} else if months < 12 {
		return fmt.Sprintf("%d months ago", months)
	} else {
		years := months / 12
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}
