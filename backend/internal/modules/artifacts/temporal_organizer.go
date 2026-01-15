package artifacts

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

// TemporalOrganizer builds document timelines and detects temporal sequences
type TemporalOrganizer struct {
	repo RepositoryInterface
}

// NewTemporalOrganizer creates a new temporal organizer
func NewTemporalOrganizer(repo RepositoryInterface) *TemporalOrganizer {
	return &TemporalOrganizer{
		repo: repo,
	}
}

// BuildTimeline builds a timeline of artifacts around the target artifact
func (to *TemporalOrganizer) BuildTimeline(
	ctx context.Context,
	programID uuid.UUID,
	targetArtifact *Artifact,
) (*TimelineContext, error) {
	// Find artifacts within 90 days before and after
	timeWindow := 90 * 24 * time.Hour
	startDate := targetArtifact.UploadedAt.Add(-timeWindow)
	endDate := targetArtifact.UploadedAt.Add(timeWindow)

	// Get artifacts in this time window
	artifacts, err := to.repo.GetArtifactsInTimeRange(ctx, programID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifacts in time range: %w", err)
	}

	// Separate into preceding and following
	preceding := []TimelineEntry{}
	following := []TimelineEntry{}

	for _, artifact := range artifacts {
		// Skip the target artifact itself
		if artifact.ArtifactID == targetArtifact.ArtifactID {
			continue
		}

		// Get summary for this artifact
		summary, err := to.repo.GetSummaryByArtifact(ctx, artifact.ArtifactID)
		summaryText := ""
		if err == nil && summary != nil && summary.ExecutiveSummary != "" {
			summaryText = summary.ExecutiveSummary
			// Truncate if too long
			if len(summaryText) > 150 {
				summaryText = summaryText[:147] + "..."
			}
		}

		entry := TimelineEntry{
			ArtifactID: artifact.ArtifactID,
			Filename:   artifact.Filename,
			UploadedAt: artifact.UploadedAt,
			Summary:    summaryText,
			RelativeTime: to.formatRelativeTime(artifact.UploadedAt, targetArtifact.UploadedAt),
		}

		if artifact.ArtifactCategory.Valid {
			entry.Category = artifact.ArtifactCategory.String
		}

		if artifact.UploadedAt.Before(targetArtifact.UploadedAt) {
			entry.RelationToTarget = "before"
			preceding = append(preceding, entry)
		} else if artifact.UploadedAt.After(targetArtifact.UploadedAt) {
			entry.RelationToTarget = "after"
			following = append(following, entry)
		} else {
			// Same day
			entry.RelationToTarget = "same_day"
			preceding = append(preceding, entry)
		}
	}

	// Sort preceding (most recent first)
	sort.Slice(preceding, func(i, j int) bool {
		return preceding[i].UploadedAt.After(preceding[j].UploadedAt)
	})

	// Sort following (earliest first)
	sort.Slice(following, func(i, j int) bool {
		return following[i].UploadedAt.Before(following[j].UploadedAt)
	})

	// Limit to reasonable counts (5 preceding, 3 following)
	if len(preceding) > 5 {
		preceding = preceding[:5]
	}
	if len(following) > 3 {
		following = following[:3]
	}

	// Estimate tokens
	tokensPerEntry := 80 // filename + category + date + summary
	estimatedTokens := (len(preceding) + len(following)) * tokensPerEntry

	return &TimelineContext{
		PrecedingArtifacts: preceding,
		FollowingArtifacts: following,
		EstimatedTokens:    estimatedTokens,
	}, nil
}

// formatRelativeTime formats time difference as human-readable string
func (to *TemporalOrganizer) formatRelativeTime(artifactTime, targetTime time.Time) string {
	diff := artifactTime.Sub(targetTime)
	absDiff := math.Abs(diff.Seconds())

	days := int(absDiff / (24 * 3600))
	hours := int(absDiff/3600) % 24

	if absDiff < 3600 { // Less than 1 hour
		minutes := int(absDiff / 60)
		if minutes < 5 {
			return "same time"
		}
		return fmt.Sprintf("%d minutes", minutes)
	} else if days == 0 {
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	} else if days == 1 {
		return "1 day"
	} else if days < 7 {
		return fmt.Sprintf("%d days", days)
	} else if days < 14 {
		return "1 week"
	} else if days < 30 {
		weeks := days / 7
		return fmt.Sprintf("%d weeks", weeks)
	} else if days < 60 {
		return "1 month"
	} else {
		months := days / 30
		return fmt.Sprintf("%d months", months)
	}
}

// DetectSequences automatically detects temporal sequences in a program
// This uses heuristics like:
// - Similar document types uploaded in succession
// - Documents with shared topics/people uploaded over time
// - Regular intervals (weekly meetings, monthly reports)
func (to *TemporalOrganizer) DetectSequences(
	ctx context.Context,
	programID uuid.UUID,
) ([]ArtifactSequence, error) {
	// Get all artifacts in program, sorted by date
	artifacts, err := to.repo.GetArtifactsByProgram(ctx, programID)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifacts: %w", err)
	}

	if len(artifacts) < 3 {
		// Need at least 3 artifacts to detect patterns
		return []ArtifactSequence{}, nil
	}

	// Sort by upload date
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].UploadedAt.Before(artifacts[j].UploadedAt)
	})

	sequences := []ArtifactSequence{}

	// Detect sequences by document category
	categorySequences := to.detectCategorySequences(artifacts)
	sequences = append(sequences, categorySequences...)

	// Detect sequences by regular intervals
	intervalSequences := to.detectIntervalSequences(artifacts)
	sequences = append(sequences, intervalSequences...)

	return sequences, nil
}

// detectCategorySequences detects sequences of similar document types
func (to *TemporalOrganizer) detectCategorySequences(artifacts []Artifact) []ArtifactSequence {
	sequences := []ArtifactSequence{}

	// Group by category
	categoryGroups := make(map[string][]Artifact)
	for _, artifact := range artifacts {
		if !artifact.ArtifactCategory.Valid {
			continue
		}
		category := artifact.ArtifactCategory.String
		categoryGroups[category] = append(categoryGroups[category], artifact)
	}

	// For each category with 3+ artifacts, create a sequence
	for category, group := range categoryGroups {
		if len(group) < 3 {
			continue
		}

		artifactIDs := make([]uuid.UUID, len(group))
		for i, artifact := range group {
			artifactIDs[i] = artifact.ArtifactID
		}

		sequence := ArtifactSequence{
			SequenceID:       uuid.New(),
			ProgramID:        group[0].ProgramID,
			SequenceName:     fmt.Sprintf("%s Documents", category),
			SequenceType:     "document_type_series",
			ArtifactIDs:      artifactIDs,
			StartDate:        group[0].UploadedAt,
			EndDate:          group[len(group)-1].UploadedAt,
			DetectionMethod:  "auto",
			ConfidenceScore:  0.7, // Medium confidence for category-based detection
		}

		sequences = append(sequences, sequence)
	}

	return sequences
}

// detectIntervalSequences detects sequences with regular time intervals
func (to *TemporalOrganizer) detectIntervalSequences(artifacts []Artifact) []ArtifactSequence {
	sequences := []ArtifactSequence{}

	// This is a simplified heuristic
	// Real implementation would use more sophisticated pattern detection

	// Group by category first
	categoryGroups := make(map[string][]Artifact)
	for _, artifact := range artifacts {
		if !artifact.ArtifactCategory.Valid {
			continue
		}
		category := artifact.ArtifactCategory.String
		categoryGroups[category] = append(categoryGroups[category], artifact)
	}

	// For each category, check for regular intervals
	for category, group := range categoryGroups {
		if len(group) < 3 {
			continue
		}

		// Sort by date
		sort.Slice(group, func(i, j int) bool {
			return group[i].UploadedAt.Before(group[j].UploadedAt)
		})

		// Calculate intervals between consecutive artifacts
		intervals := []float64{}
		for i := 1; i < len(group); i++ {
			daysDiff := group[i].UploadedAt.Sub(group[i-1].UploadedAt).Hours() / 24
			intervals = append(intervals, daysDiff)
		}

		// Check if intervals are relatively consistent
		if len(intervals) > 0 {
			avgInterval := to.average(intervals)
			stdDev := to.stdDev(intervals, avgInterval)

			// If standard deviation is low relative to mean, it's a regular sequence
			if stdDev < avgInterval*0.3 { // Less than 30% variance
				var sequenceType string
				var sequenceName string

				if avgInterval < 8 { // ~7 days
					sequenceType = "weekly_series"
					sequenceName = fmt.Sprintf("Weekly %s Reports", category)
				} else if avgInterval < 35 { // ~30 days
					sequenceType = "monthly_series"
					sequenceName = fmt.Sprintf("Monthly %s Reports", category)
				} else if avgInterval < 100 { // ~quarterly
					sequenceType = "quarterly_series"
					sequenceName = fmt.Sprintf("Quarterly %s Reports", category)
				} else {
					continue // Too irregular
				}

				artifactIDs := make([]uuid.UUID, len(group))
				for i, artifact := range group {
					artifactIDs[i] = artifact.ArtifactID
				}

				sequence := ArtifactSequence{
					SequenceID:       uuid.New(),
					ProgramID:        group[0].ProgramID,
					SequenceName:     sequenceName,
					SequenceType:     sequenceType,
					ArtifactIDs:      artifactIDs,
					StartDate:        group[0].UploadedAt,
					EndDate:          group[len(group)-1].UploadedAt,
					DetectionMethod:  "auto",
					ConfidenceScore:  0.85, // Higher confidence for interval-based detection
				}

				sequences = append(sequences, sequence)
			}
		}
	}

	return sequences
}

// ArtifactSequence represents a detected temporal sequence
type ArtifactSequence struct {
	SequenceID       uuid.UUID
	ProgramID        uuid.UUID
	SequenceName     string
	SequenceType     string
	ArtifactIDs      []uuid.UUID
	StartDate        time.Time
	EndDate          time.Time
	DetectionMethod  string
	ConfidenceScore  float64
}

// Helper functions

func (to *TemporalOrganizer) average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (to *TemporalOrganizer) stdDev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}
	variance /= float64(len(values))
	return math.Sqrt(variance)
}

// SaveSequence persists a detected sequence to the database
func (to *TemporalOrganizer) SaveSequence(
	ctx context.Context,
	sequence *ArtifactSequence,
) error {
	return to.repo.SaveTemporalSequence(ctx, sequence)
}

// GetSequencesForProgram retrieves all saved sequences for a program
func (to *TemporalOrganizer) GetSequencesForProgram(
	ctx context.Context,
	programID uuid.UUID,
) ([]ArtifactSequence, error) {
	return to.repo.GetTemporalSequencesByProgram(ctx, programID)
}

// FormatTimelineForPrompt formats the timeline context for the AI prompt
func (to *TemporalOrganizer) FormatTimelineForPrompt(timeline *TimelineContext) string {
	if timeline == nil || (len(timeline.PrecedingArtifacts) == 0 && len(timeline.FollowingArtifacts) == 0) {
		return "No temporal context available."
	}

	output := "Document Timeline:\n\n"

	if len(timeline.PrecedingArtifacts) > 0 {
		output += "Before Current Artifact:\n"
		for _, entry := range timeline.PrecedingArtifacts {
			output += fmt.Sprintf("  [%s] %s", entry.RelativeTime, entry.Filename)
			if entry.Category != "" {
				output += fmt.Sprintf(" (%s)", entry.Category)
			}
			if entry.Summary != "" {
				output += fmt.Sprintf("\n    %s", entry.Summary)
			}
			output += "\n"
		}
		output += "\n"
	}

	output += "Current Artifact:\n  [Today] [ANALYZING THIS DOCUMENT]\n\n"

	if len(timeline.FollowingArtifacts) > 0 {
		output += "After Current Artifact:\n"
		for _, entry := range timeline.FollowingArtifacts {
			output += fmt.Sprintf("  [%s] %s", entry.RelativeTime, entry.Filename)
			if entry.Category != "" {
				output += fmt.Sprintf(" (%s)", entry.Category)
			}
			if entry.Summary != "" {
				output += fmt.Sprintf("\n    %s", entry.Summary)
			}
			output += "\n"
		}
	}

	return output
}
