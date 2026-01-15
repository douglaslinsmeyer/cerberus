// Package risk provides service layer for risk and issue management.
// This includes risk identification, assessment, mitigation tracking, and conversations.
package risk

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service handles business logic for risk management
type Service struct {
	repo RepositoryInterface
}

// NewService creates a new risk service
func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// NewServiceWithMocks creates a service with mock dependencies (useful for testing)
func NewServiceWithMocks(repo RepositoryInterface) *Service {
	return &Service{
		repo: repo,
	}
}

// CreateRisk creates a new risk in the risk register
func (s *Service) CreateRisk(ctx context.Context, req CreateRiskRequest) (uuid.UUID, error) {
	// Validate request
	if req.ProgramID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("program_id is required")
	}
	if req.CreatedBy == uuid.Nil {
		return uuid.Nil, fmt.Errorf("created_by is required")
	}
	if req.Title == "" {
		return uuid.Nil, fmt.Errorf("title is required")
	}
	if req.Description == "" {
		return uuid.Nil, fmt.Errorf("description is required")
	}

	// Validate probability
	if !isValidProbability(req.Probability) {
		return uuid.Nil, fmt.Errorf("invalid probability: must be very_low, low, medium, high, or very_high")
	}

	// Validate impact
	if !isValidImpact(req.Impact) {
		return uuid.Nil, fmt.Errorf("invalid impact: must be very_low, low, medium, high, or very_high")
	}

	// Validate category
	if !isValidCategory(req.Category) {
		return uuid.Nil, fmt.Errorf("invalid category: must be technical, financial, schedule, resource, or external")
	}

	// Create risk
	riskID := uuid.New()
	risk := &Risk{
		RiskID:      riskID,
		ProgramID:   req.ProgramID,
		Title:       req.Title,
		Description: req.Description,
		Probability: req.Probability,
		Impact:      req.Impact,
		Category:    req.Category,
		Status:      "open", // New risks start as open
		CreatedBy:   req.CreatedBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IdentifiedDate: time.Now(),
	}

	// Set optional fields
	if req.OwnerUserID != nil {
		risk.OwnerUserID = uuid.NullUUID{UUID: *req.OwnerUserID, Valid: true}
	}
	if req.OwnerName != "" {
		risk.OwnerName.String = req.OwnerName
		risk.OwnerName.Valid = true
	}
	if req.TargetResolutionDate != nil {
		risk.TargetResolutionDate.Time = *req.TargetResolutionDate
		risk.TargetResolutionDate.Valid = true
	}

	// Save to database
	err := s.repo.CreateRisk(ctx, risk)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create risk: %w", err)
	}

	return riskID, nil
}

// GetRisk retrieves a risk by ID
func (s *Service) GetRisk(ctx context.Context, riskID uuid.UUID) (*Risk, error) {
	if riskID == uuid.Nil {
		return nil, fmt.Errorf("risk_id is required")
	}

	risk, err := s.repo.GetRiskByID(ctx, riskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get risk: %w", err)
	}

	return risk, nil
}

// GetRiskWithContext retrieves a risk with all its related data
func (s *Service) GetRiskWithContext(ctx context.Context, riskID uuid.UUID) (*RiskWithMetadata, error) {
	if riskID == uuid.Nil {
		return nil, fmt.Errorf("risk_id is required")
	}

	riskWithMetadata, err := s.repo.GetRiskWithContext(ctx, riskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get risk with context: %w", err)
	}

	return riskWithMetadata, nil
}

// ListRisks retrieves risks for a program with optional filters
func (s *Service) ListRisks(ctx context.Context, filter RiskFilterRequest) ([]Risk, error) {
	if filter.ProgramID == uuid.Nil {
		return nil, fmt.Errorf("program_id is required")
	}

	// Validate filters
	if filter.Status != "" && !isValidStatus(filter.Status) {
		return nil, fmt.Errorf("invalid status: must be suggested, open, monitoring, mitigated, closed, or realized")
	}
	if filter.Category != "" && !isValidCategory(filter.Category) {
		return nil, fmt.Errorf("invalid category: must be technical, financial, schedule, resource, or external")
	}
	if filter.Severity != "" && !isValidSeverity(filter.Severity) {
		return nil, fmt.Errorf("invalid severity: must be low, medium, high, or critical")
	}

	// Set default pagination
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	risks, err := s.repo.ListRisks(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list risks: %w", err)
	}

	return risks, nil
}

// ListRisksWithSuggestions retrieves both risks and pending suggestions for a program
func (s *Service) ListRisksWithSuggestions(ctx context.Context, filter RiskFilterRequest, includeSuggestions bool) (*RiskListWithSuggestionsResponse, error) {
	if filter.ProgramID == uuid.Nil {
		return nil, fmt.Errorf("program_id is required")
	}

	// Validate filters
	if filter.Status != "" && !isValidStatus(filter.Status) {
		return nil, fmt.Errorf("invalid status: must be suggested, open, monitoring, mitigated, closed, or realized")
	}
	if filter.Category != "" && !isValidCategory(filter.Category) {
		return nil, fmt.Errorf("invalid category: must be technical, financial, schedule, resource, or external")
	}
	if filter.Severity != "" && !isValidSeverity(filter.Severity) {
		return nil, fmt.Errorf("invalid severity: must be low, medium, high, or critical")
	}

	// Set default pagination
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	response, err := s.repo.ListRisksWithSuggestions(ctx, filter, includeSuggestions)
	if err != nil {
		return nil, fmt.Errorf("failed to list risks with suggestions: %w", err)
	}

	return response, nil
}

// UpdateRiskStatus updates the status of a risk
func (s *Service) UpdateRiskStatus(ctx context.Context, riskID uuid.UUID, newStatus string) error {
	if riskID == uuid.Nil {
		return fmt.Errorf("risk_id is required")
	}
	if !isValidStatus(newStatus) {
		return fmt.Errorf("invalid status: must be suggested, open, monitoring, mitigated, closed, or realized")
	}

	// Get existing risk
	risk, err := s.repo.GetRiskByID(ctx, riskID)
	if err != nil {
		return fmt.Errorf("failed to get risk: %w", err)
	}

	// Update status
	risk.Status = newStatus

	// Set closed/realized dates based on status
	now := time.Now()
	if newStatus == "closed" && !risk.ClosedDate.Valid {
		risk.ClosedDate.Time = now
		risk.ClosedDate.Valid = true
	}
	if newStatus == "realized" && !risk.RealizedDate.Valid {
		risk.RealizedDate.Time = now
		risk.RealizedDate.Valid = true
	}

	// Save updated risk
	err = s.repo.UpdateRisk(ctx, risk)
	if err != nil {
		return fmt.Errorf("failed to update risk status: %w", err)
	}

	return nil
}

// UpdateRisk updates a risk with the provided changes
func (s *Service) UpdateRisk(ctx context.Context, riskID uuid.UUID, req UpdateRiskRequest) error {
	if riskID == uuid.Nil {
		return fmt.Errorf("risk_id is required")
	}

	// Get existing risk
	risk, err := s.repo.GetRiskByID(ctx, riskID)
	if err != nil {
		return fmt.Errorf("failed to get risk: %w", err)
	}

	// Apply updates
	if req.Title != nil {
		risk.Title = *req.Title
	}
	if req.Description != nil {
		risk.Description = *req.Description
	}
	if req.Probability != nil {
		if !isValidProbability(*req.Probability) {
			return fmt.Errorf("invalid probability")
		}
		risk.Probability = *req.Probability
	}
	if req.Impact != nil {
		if !isValidImpact(*req.Impact) {
			return fmt.Errorf("invalid impact")
		}
		risk.Impact = *req.Impact
	}
	if req.Category != nil {
		if !isValidCategory(*req.Category) {
			return fmt.Errorf("invalid category")
		}
		risk.Category = *req.Category
	}
	if req.Status != nil {
		if !isValidStatus(*req.Status) {
			return fmt.Errorf("invalid status")
		}
		risk.Status = *req.Status

		// Set closed/realized dates
		now := time.Now()
		if *req.Status == "closed" && !risk.ClosedDate.Valid {
			risk.ClosedDate.Time = now
			risk.ClosedDate.Valid = true
		}
		if *req.Status == "realized" && !risk.RealizedDate.Valid {
			risk.RealizedDate.Time = now
			risk.RealizedDate.Valid = true
		}
	}
	if req.OwnerUserID != nil {
		risk.OwnerUserID = uuid.NullUUID{UUID: *req.OwnerUserID, Valid: true}
	}
	if req.OwnerName != nil {
		risk.OwnerName.String = *req.OwnerName
		risk.OwnerName.Valid = *req.OwnerName != ""
	}
	if req.TargetResolutionDate != nil {
		risk.TargetResolutionDate.Time = *req.TargetResolutionDate
		risk.TargetResolutionDate.Valid = true
	}

	// Save updated risk
	err = s.repo.UpdateRisk(ctx, risk)
	if err != nil {
		return fmt.Errorf("failed to update risk: %w", err)
	}

	return nil
}

// DeleteRisk soft-deletes a risk
func (s *Service) DeleteRisk(ctx context.Context, riskID uuid.UUID) error {
	if riskID == uuid.Nil {
		return fmt.Errorf("risk_id is required")
	}

	err := s.repo.DeleteRisk(ctx, riskID)
	if err != nil {
		return fmt.Errorf("failed to delete risk: %w", err)
	}

	return nil
}

// ListSuggestions retrieves risk suggestions for a program
func (s *Service) ListSuggestions(ctx context.Context, programID uuid.UUID, includeProcessed bool) ([]RiskSuggestion, error) {
	if programID == uuid.Nil {
		return nil, fmt.Errorf("program_id is required")
	}

	suggestions, err := s.repo.ListSuggestions(ctx, programID, includeProcessed)
	if err != nil {
		return nil, fmt.Errorf("failed to list suggestions: %w", err)
	}

	return suggestions, nil
}

// ApproveSuggestion approves a risk suggestion and creates a new risk
func (s *Service) ApproveSuggestion(ctx context.Context, req ApproveSuggestionRequest) (uuid.UUID, error) {
	if req.SuggestionID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("suggestion_id is required")
	}
	if req.ApprovedBy == uuid.Nil {
		return uuid.Nil, fmt.Errorf("approved_by is required")
	}

	// Get suggestion
	suggestion, err := s.repo.GetSuggestionByID(ctx, req.SuggestionID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get suggestion: %w", err)
	}

	// Check if already processed
	if suggestion.IsApproved {
		return uuid.Nil, fmt.Errorf("suggestion already approved")
	}
	if suggestion.IsDismissed {
		return uuid.Nil, fmt.Errorf("suggestion already dismissed")
	}

	// Create risk from suggestion
	riskID := uuid.New()
	now := time.Now()

	// Use overrides if provided, otherwise use AI suggestions
	probability := suggestion.SuggestedProbability
	if req.OverrideProbability != nil {
		probability = *req.OverrideProbability
	}

	impact := suggestion.SuggestedImpact
	if req.OverrideImpact != nil {
		impact = *req.OverrideImpact
	}

	category := suggestion.SuggestedCategory
	if req.OverrideCategory != nil {
		category = *req.OverrideCategory
	}

	risk := &Risk{
		RiskID:         riskID,
		ProgramID:      suggestion.ProgramID,
		Title:          suggestion.Title,
		Description:    suggestion.Description,
		Probability:    probability,
		Impact:         impact,
		Category:       category,
		Status:         "open",
		CreatedBy:      req.ApprovedBy,
		CreatedAt:      now,
		UpdatedAt:      now,
		IdentifiedDate: now,
	}

	// Set AI metadata to track origin
	risk.AIConfidenceScore = suggestion.AIConfidenceScore
	risk.AIDetectedAt.Time = suggestion.AIDetectedAt
	risk.AIDetectedAt.Valid = true

	// Set optional fields
	if req.OwnerUserID != nil {
		risk.OwnerUserID = uuid.NullUUID{UUID: *req.OwnerUserID, Valid: true}
	}
	if req.TargetResolutionDate != nil {
		risk.TargetResolutionDate.Time = *req.TargetResolutionDate
		risk.TargetResolutionDate.Valid = true
	}

	// Create the risk
	err = s.repo.CreateRisk(ctx, risk)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create risk from suggestion: %w", err)
	}

	// Link artifacts from suggestion
	for _, artifactID := range suggestion.SourceArtifactIDs {
		link := &RiskArtifactLink{
			LinkID:     uuid.New(),
			RiskID:     riskID,
			ArtifactID: artifactID,
			LinkType:   "evidence",
			CreatedBy:  req.ApprovedBy,
			CreatedAt:  now,
		}
		link.Description.String = "Artifact referenced in AI risk detection"
		link.Description.Valid = true

		_ = s.repo.LinkArtifact(ctx, link) // Best effort
	}

	// Update suggestion as approved
	suggestion.IsApproved = true
	suggestion.ApprovedBy = uuid.NullUUID{UUID: req.ApprovedBy, Valid: true}
	suggestion.ApprovedAt.Time = now
	suggestion.ApprovedAt.Valid = true
	suggestion.CreatedRiskID = uuid.NullUUID{UUID: riskID, Valid: true}

	err = s.repo.UpdateSuggestion(ctx, suggestion)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to update suggestion: %w", err)
	}

	return riskID, nil
}

// DismissSuggestion dismisses a risk suggestion
func (s *Service) DismissSuggestion(ctx context.Context, req DismissSuggestionRequest) error {
	if req.SuggestionID == uuid.Nil {
		return fmt.Errorf("suggestion_id is required")
	}
	if req.DismissedBy == uuid.Nil {
		return fmt.Errorf("dismissed_by is required")
	}
	if req.Reason == "" {
		return fmt.Errorf("dismissal reason is required")
	}

	// Get suggestion
	suggestion, err := s.repo.GetSuggestionByID(ctx, req.SuggestionID)
	if err != nil {
		return fmt.Errorf("failed to get suggestion: %w", err)
	}

	// Check if already processed
	if suggestion.IsApproved {
		return fmt.Errorf("suggestion already approved, cannot dismiss")
	}
	if suggestion.IsDismissed {
		return fmt.Errorf("suggestion already dismissed")
	}

	// Update suggestion as dismissed
	now := time.Now()
	suggestion.IsDismissed = true
	suggestion.DismissedBy = uuid.NullUUID{UUID: req.DismissedBy, Valid: true}
	suggestion.DismissedAt.Time = now
	suggestion.DismissedAt.Valid = true
	suggestion.DismissalReason.String = req.Reason
	suggestion.DismissalReason.Valid = true

	err = s.repo.UpdateSuggestion(ctx, suggestion)
	if err != nil {
		return fmt.Errorf("failed to update suggestion: %w", err)
	}

	return nil
}

// AcceptEnrichment marks an enrichment link as relevant for a specific risk
func (s *Service) AcceptEnrichment(ctx context.Context, riskID uuid.UUID, enrichmentID uuid.UUID, reviewedBy uuid.UUID) error {
	if riskID == uuid.Nil {
		return fmt.Errorf("risk_id is required")
	}
	if enrichmentID == uuid.Nil {
		return fmt.Errorf("enrichment_id is required")
	}
	if reviewedBy == uuid.Nil {
		return fmt.Errorf("reviewed_by is required")
	}

	err := s.repo.UpdateEnrichmentRelevance(ctx, riskID, enrichmentID, true, reviewedBy)
	if err != nil {
		return fmt.Errorf("failed to accept enrichment: %w", err)
	}

	return nil
}

// RejectEnrichment marks an enrichment link as not relevant for a specific risk
func (s *Service) RejectEnrichment(ctx context.Context, riskID uuid.UUID, enrichmentID uuid.UUID, reviewedBy uuid.UUID) error {
	if riskID == uuid.Nil {
		return fmt.Errorf("risk_id is required")
	}
	if enrichmentID == uuid.Nil {
		return fmt.Errorf("enrichment_id is required")
	}
	if reviewedBy == uuid.Nil {
		return fmt.Errorf("reviewed_by is required")
	}

	err := s.repo.UpdateEnrichmentRelevance(ctx, riskID, enrichmentID, false, reviewedBy)
	if err != nil {
		return fmt.Errorf("failed to reject enrichment: %w", err)
	}

	return nil
}

// GetEnrichments retrieves enrichments for a risk
func (s *Service) GetEnrichments(ctx context.Context, riskID uuid.UUID) ([]RiskEnrichmentWithMetadata, error) {
	if riskID == uuid.Nil {
		return nil, fmt.Errorf("risk_id is required")
	}

	enrichments, err := s.repo.GetEnrichmentsByRisk(ctx, riskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get enrichments: %w", err)
	}

	return enrichments, nil
}

// AddMitigation adds a mitigation action to a risk
func (s *Service) AddMitigation(ctx context.Context, req CreateMitigationRequest) (uuid.UUID, error) {
	if req.RiskID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("risk_id is required")
	}
	if req.CreatedBy == uuid.Nil {
		return uuid.Nil, fmt.Errorf("created_by is required")
	}
	if req.ActionDescription == "" {
		return uuid.Nil, fmt.Errorf("action_description is required")
	}

	// Validate strategy
	if !isValidStrategy(req.Strategy) {
		return uuid.Nil, fmt.Errorf("invalid strategy: must be avoid, transfer, mitigate, or accept")
	}

	// Verify risk exists
	_, err := s.repo.GetRiskByID(ctx, req.RiskID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("risk not found: %w", err)
	}

	// Create mitigation
	mitigationID := uuid.New()
	now := time.Now()

	mitigation := &RiskMitigation{
		MitigationID:      mitigationID,
		RiskID:            req.RiskID,
		Strategy:          req.Strategy,
		ActionDescription: req.ActionDescription,
		Status:            "planned",
		Currency:          req.Currency,
		CreatedBy:         req.CreatedBy,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Set optional fields
	if req.ExpectedProbabilityReduction != "" {
		if !isValidProbability(req.ExpectedProbabilityReduction) {
			return uuid.Nil, fmt.Errorf("invalid expected_probability_reduction")
		}
		mitigation.ExpectedProbabilityReduction.String = req.ExpectedProbabilityReduction
		mitigation.ExpectedProbabilityReduction.Valid = true
	}
	if req.ExpectedImpactReduction != "" {
		if !isValidImpact(req.ExpectedImpactReduction) {
			return uuid.Nil, fmt.Errorf("invalid expected_impact_reduction")
		}
		mitigation.ExpectedImpactReduction.String = req.ExpectedImpactReduction
		mitigation.ExpectedImpactReduction.Valid = true
	}
	if req.AssignedTo != nil {
		mitigation.AssignedTo = uuid.NullUUID{UUID: *req.AssignedTo, Valid: true}
	}
	if req.TargetCompletionDate != nil {
		mitigation.TargetCompletionDate.Time = *req.TargetCompletionDate
		mitigation.TargetCompletionDate.Valid = true
	}
	if req.EstimatedCost != nil {
		mitigation.EstimatedCost.Float64 = *req.EstimatedCost
		mitigation.EstimatedCost.Valid = true
	}

	// Save to database
	err = s.repo.CreateMitigation(ctx, mitigation)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create mitigation: %w", err)
	}

	return mitigationID, nil
}

// UpdateMitigation updates a mitigation action
func (s *Service) UpdateMitigation(ctx context.Context, mitigationID uuid.UUID, req UpdateMitigationRequest) error {
	if mitigationID == uuid.Nil {
		return fmt.Errorf("mitigation_id is required")
	}

	// Get existing mitigation
	mitigation, err := s.repo.GetMitigationByID(ctx, mitigationID)
	if err != nil {
		return fmt.Errorf("failed to get mitigation: %w", err)
	}

	// Apply updates
	if req.Status != nil {
		if !isValidMitigationStatus(*req.Status) {
			return fmt.Errorf("invalid status: must be planned, in_progress, completed, or abandoned")
		}
		mitigation.Status = *req.Status

		// Set completion date if status is completed
		if *req.Status == "completed" && !mitigation.ActualCompletionDate.Valid {
			mitigation.ActualCompletionDate.Time = time.Now()
			mitigation.ActualCompletionDate.Valid = true
		}
	}
	if req.EffectivenessRating != nil {
		if *req.EffectivenessRating < 1 || *req.EffectivenessRating > 5 {
			return fmt.Errorf("effectiveness_rating must be between 1 and 5")
		}
		mitigation.EffectivenessRating.Int32 = int32(*req.EffectivenessRating)
		mitigation.EffectivenessRating.Valid = true
	}
	if req.ActualCompletionDate != nil {
		mitigation.ActualCompletionDate.Time = *req.ActualCompletionDate
		mitigation.ActualCompletionDate.Valid = true
	}
	if req.ActualCost != nil {
		mitigation.ActualCost.Float64 = *req.ActualCost
		mitigation.ActualCost.Valid = true
	}

	// Save updated mitigation
	err = s.repo.UpdateMitigation(ctx, mitigation)
	if err != nil {
		return fmt.Errorf("failed to update mitigation: %w", err)
	}

	return nil
}

// DeleteMitigation soft-deletes a mitigation
func (s *Service) DeleteMitigation(ctx context.Context, mitigationID uuid.UUID) error {
	if mitigationID == uuid.Nil {
		return fmt.Errorf("mitigation_id is required")
	}

	err := s.repo.DeleteMitigation(ctx, mitigationID)
	if err != nil {
		return fmt.Errorf("failed to delete mitigation: %w", err)
	}

	return nil
}

// LinkArtifact links an artifact to a risk
func (s *Service) LinkArtifact(ctx context.Context, req LinkArtifactRequest) (uuid.UUID, error) {
	if req.RiskID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("risk_id is required")
	}
	if req.ArtifactID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("artifact_id is required")
	}
	if req.CreatedBy == uuid.Nil {
		return uuid.Nil, fmt.Errorf("created_by is required")
	}
	if !isValidLinkType(req.LinkType) {
		return uuid.Nil, fmt.Errorf("invalid link_type: must be evidence, impact_analysis, mitigation_plan, or related")
	}

	// Verify risk exists
	_, err := s.repo.GetRiskByID(ctx, req.RiskID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("risk not found: %w", err)
	}

	// Create link
	linkID := uuid.New()
	link := &RiskArtifactLink{
		LinkID:     linkID,
		RiskID:     req.RiskID,
		ArtifactID: req.ArtifactID,
		LinkType:   req.LinkType,
		CreatedBy:  req.CreatedBy,
		CreatedAt:  time.Now(),
	}

	if req.Description != "" {
		link.Description.String = req.Description
		link.Description.Valid = true
	}

	err = s.repo.LinkArtifact(ctx, link)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to link artifact: %w", err)
	}

	return linkID, nil
}

// UnlinkArtifact removes a link between a risk and an artifact
func (s *Service) UnlinkArtifact(ctx context.Context, linkID uuid.UUID) error {
	if linkID == uuid.Nil {
		return fmt.Errorf("link_id is required")
	}

	err := s.repo.UnlinkArtifact(ctx, linkID)
	if err != nil {
		return fmt.Errorf("failed to unlink artifact: %w", err)
	}

	return nil
}

// GetLinkedArtifacts retrieves all artifacts linked to a risk
func (s *Service) GetLinkedArtifacts(ctx context.Context, riskID uuid.UUID) ([]RiskArtifactLink, error) {
	if riskID == uuid.Nil {
		return nil, fmt.Errorf("risk_id is required")
	}

	links, err := s.repo.GetLinkedArtifacts(ctx, riskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get linked artifacts: %w", err)
	}

	return links, nil
}

// GetRisksByArtifact retrieves all risks associated with an artifact
func (s *Service) GetRisksByArtifact(ctx context.Context, artifactID uuid.UUID) ([]Risk, error) {
	if artifactID == uuid.Nil {
		return nil, fmt.Errorf("artifact_id is required")
	}

	risks, err := s.repo.GetRisksByArtifact(ctx, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get risks by artifact: %w", err)
	}

	return risks, nil
}

// Validation helpers

func isValidProbability(probability string) bool {
	validValues := map[string]bool{
		"very_low": true, "low": true, "medium": true, "high": true, "very_high": true,
	}
	return validValues[probability]
}

func isValidImpact(impact string) bool {
	validValues := map[string]bool{
		"very_low": true, "low": true, "medium": true, "high": true, "very_high": true,
	}
	return validValues[impact]
}

func isValidSeverity(severity string) bool {
	validValues := map[string]bool{
		"low": true, "medium": true, "high": true, "critical": true,
	}
	return validValues[severity]
}

func isValidCategory(category string) bool {
	validValues := map[string]bool{
		"technical": true, "financial": true, "schedule": true, "resource": true, "external": true,
	}
	return validValues[category]
}

func isValidStatus(status string) bool {
	validValues := map[string]bool{
		"suggested": true, "open": true, "monitoring": true, "mitigated": true, "closed": true, "realized": true,
	}
	return validValues[status]
}

func isValidStrategy(strategy string) bool {
	validValues := map[string]bool{
		"avoid": true, "transfer": true, "mitigate": true, "accept": true,
	}
	return validValues[strategy]
}

func isValidMitigationStatus(status string) bool {
	validValues := map[string]bool{
		"planned": true, "in_progress": true, "completed": true, "abandoned": true,
	}
	return validValues[status]
}

func isValidLinkType(linkType string) bool {
	validValues := map[string]bool{
		"evidence": true, "impact_analysis": true, "mitigation_plan": true, "related": true,
	}
	return validValues[linkType]
}
