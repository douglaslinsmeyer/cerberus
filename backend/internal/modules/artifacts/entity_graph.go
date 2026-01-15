package artifacts

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// EntityGraphQuery handles entity relationship queries
// It uses the artifact_entity_graph table to find person co-occurrences
// and relationship patterns across artifacts
type EntityGraphQuery struct {
	repo RepositoryInterface
}

// NewEntityGraphQuery creates a new entity graph query service
func NewEntityGraphQuery(repo RepositoryInterface) *EntityGraphQuery {
	return &EntityGraphQuery{
		repo: repo,
	}
}

// PersonRelationship represents a relationship between two people
// based on their co-occurrence across artifacts
type PersonRelationship struct {
	Person1ID        uuid.UUID
	Person1Name      string
	Person2ID        uuid.UUID
	Person2Name      string
	CoOccurrences    int
	SharedArtifacts  []uuid.UUID
	Strength         float64
	LastCoOccurrence *Artifact
}

// ArtifactWithEntityOverlap represents an artifact with entity overlap information
type ArtifactWithEntityOverlap struct {
	Artifact         *Artifact
	SharedPersonIDs  []uuid.UUID
	SharedPersonNames []string
	OverlapRatio     float64
	OverlapCount     int
}

// GetPersonCoOccurrences finds people who frequently appear with the given person
// Returns relationships sorted by co-occurrence count (descending)
func (egq *EntityGraphQuery) GetPersonCoOccurrences(
	ctx context.Context,
	personID uuid.UUID,
	programID uuid.UUID,
) ([]PersonRelationship, error) {
	// Query the artifact_entity_graph table for this person
	// The query needs to check both person_id_1 and person_id_2 since
	// the table stores relationships with person_id_1 < person_id_2

	// This will be implemented in the repository layer
	return egq.repo.GetPersonRelationships(ctx, personID, programID)
}

// FindArtifactsWithSharedPeople finds artifacts that share people with the target
func (egq *EntityGraphQuery) FindArtifactsWithSharedPeople(
	ctx context.Context,
	targetArtifactID uuid.UUID,
	programID uuid.UUID,
	personIDs []uuid.UUID,
	limit int,
) ([]ArtifactWithEntityOverlap, error) {
	if len(personIDs) == 0 {
		return []ArtifactWithEntityOverlap{}, nil
	}

	// Find artifacts that mention any of these people
	// This will be implemented in repository
	return egq.repo.FindArtifactsWithEntityOverlap(ctx, targetArtifactID, programID, personIDs, limit)
}

// GetEntityGraph builds a full entity graph for a program
// This can be used for visualization or advanced relationship analysis
func (egq *EntityGraphQuery) GetEntityGraph(
	ctx context.Context,
	programID uuid.UUID,
	minCoOccurrences int,
) ([]PersonRelationship, error) {
	return egq.repo.GetProgramEntityGraph(ctx, programID, minCoOccurrences)
}

// ComputeEntityOverlapScore calculates how much entity overlap exists
// between two artifacts (Jaccard similarity)
func (egq *EntityGraphQuery) ComputeEntityOverlapScore(
	artifact1PersonIDs []uuid.UUID,
	artifact2PersonIDs []uuid.UUID,
) float64 {
	if len(artifact1PersonIDs) == 0 || len(artifact2PersonIDs) == 0 {
		return 0.0
	}

	// Build set of artifact1 person IDs
	set1 := make(map[uuid.UUID]bool)
	for _, id := range artifact1PersonIDs {
		set1[id] = true
	}

	// Count intersection and union
	intersection := 0
	union := len(artifact1PersonIDs)

	for _, id := range artifact2PersonIDs {
		if set1[id] {
			intersection++
		} else {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	// Jaccard similarity = |intersection| / |union|
	return float64(intersection) / float64(union)
}

// BuildEntityGraphForArtifact updates the entity graph after an artifact is processed
// This is typically called by a background worker or trigger
func (egq *EntityGraphQuery) BuildEntityGraphForArtifact(
	ctx context.Context,
	artifactID uuid.UUID,
) error {
	// Get all persons in this artifact
	persons, err := egq.repo.GetPersonsByArtifact(ctx, artifactID)
	if err != nil {
		return fmt.Errorf("failed to get persons for artifact: %w", err)
	}

	if len(persons) < 2 {
		// Need at least 2 people for relationships
		return nil
	}

	// Get program ID
	artifact, err := egq.repo.GetArtifactByID(ctx, artifactID)
	if err != nil {
		return fmt.Errorf("failed to get artifact: %w", err)
	}

	// For each pair of people, create or update edge
	for i := 0; i < len(persons); i++ {
		for j := i + 1; j < len(persons); j++ {
			person1 := persons[i]
			person2 := persons[j]

			// Ensure person1 < person2 (consistent ordering)
			if person1.PersonID.String() > person2.PersonID.String() {
				person1, person2 = person2, person1
			}

			// Upsert edge in entity graph
			err := egq.repo.UpsertEntityGraphEdge(ctx, artifact.ProgramID, person1.PersonID, person2.PersonID, artifactID)
			if err != nil {
				return fmt.Errorf("failed to upsert entity graph edge: %w", err)
			}
		}
	}

	return nil
}

// GetKeyPeopleForProgram returns the most mentioned people in a program
func (egq *EntityGraphQuery) GetKeyPeopleForProgram(
	ctx context.Context,
	programID uuid.UUID,
	limit int,
) ([]PersonContext, error) {
	return egq.repo.GetKeyPeopleByProgram(ctx, programID, limit)
}

// FindPeopleByName finds people matching a name pattern (for entity resolution)
func (egq *EntityGraphQuery) FindPeopleByName(
	ctx context.Context,
	programID uuid.UUID,
	namePattern string,
) ([]Person, error) {
	return egq.repo.FindPeopleByNamePattern(ctx, programID, namePattern)
}

// GetEntityStats returns statistics about the entity graph
func (egq *EntityGraphQuery) GetEntityStats(
	ctx context.Context,
	programID uuid.UUID,
) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count total unique people
	totalPeople, err := egq.repo.CountUniquePeopleInProgram(ctx, programID)
	if err != nil {
		return nil, err
	}
	stats["total_unique_people"] = totalPeople

	// Count total relationships
	totalRelationships, err := egq.repo.CountEntityGraphEdges(ctx, programID)
	if err != nil {
		return nil, err
	}
	stats["total_relationships"] = totalRelationships

	// Average co-occurrences
	avgCoOccurrences, err := egq.repo.GetAverageCoOccurrences(ctx, programID)
	if err != nil {
		return nil, err
	}
	stats["avg_co_occurrences"] = avgCoOccurrences

	// Get most connected person
	mostConnected, err := egq.repo.GetMostConnectedPerson(ctx, programID)
	if err == nil && mostConnected != nil {
		stats["most_connected_person"] = mostConnected.PersonName
		stats["most_connected_person_relationships"] = mostConnected.MentionCount // reuse this field for relationship count
	}

	return stats, nil
}
