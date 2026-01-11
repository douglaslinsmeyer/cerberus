package artifacts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cerberus/backend/internal/platform/ai"
	"github.com/google/uuid"
)

// AIAnalyzer handles AI-powered artifact analysis
type AIAnalyzer struct {
	client  *ai.Client
	prompts *ai.PromptLibrary
	repo    RepositoryInterface
}

// NewAIAnalyzer creates a new AI analyzer
func NewAIAnalyzer(client *ai.Client, repo RepositoryInterface) *AIAnalyzer {
	return &AIAnalyzer{
		client:  client,
		prompts: ai.NewPromptLibrary(),
		repo:    repo,
	}
}

// AnalysisResult contains all extracted metadata from AI analysis
type AnalysisResult struct {
	Summary         ArtifactSummary
	Topics          []Topic
	Persons         []Person
	Facts           []Fact
	Insights        []Insight
	ProcessingTime  time.Duration
	TokensUsed      int
	Cost            float64
}

// AIExtractionResponse matches the JSON schema from the prompt
type AIExtractionResponse struct {
	Summary          string `json:"summary"`
	KeyTopics        []struct {
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

	// Get prompt template
	promptTmpl, err := a.prompts.Get("artifact_analysis_v1")
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt template: %w", err)
	}

	// Compile user prompt with variables
	vars := map[string]interface{}{
		"ProgramName":     programContext.ProgramName,
		"CustomTaxonomy":  programContext.CustomTaxonomy,
		"KeyStakeholders": programContext.KeyStakeholders,
		"ActiveRisks":     programContext.ActiveRisks,
		"Filename":        artifact.Filename,
		"ArtifactContent": artifact.RawContent.String,
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
	var extraction AIExtractionResponse
	if err := json.Unmarshal([]byte(responseText), &extraction); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// Convert to domain models
	result := &AnalysisResult{
		ProcessingTime: time.Since(startTime),
		TokensUsed:     resp.Usage.InputTokens + resp.Usage.OutputTokens,
		Cost:           ai.NewCostCalculator().CalculateCost(resp.Model, &resp.Usage),
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
