package programs

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/cerberus/backend/internal/platform/db"
	"github.com/google/uuid"
)

// StakeholderRepository handles database operations for stakeholders
type StakeholderRepository struct {
	db *db.DB
}

// NewStakeholderRepository creates a new stakeholder repository
func NewStakeholderRepository(database *db.DB) *StakeholderRepository {
	return &StakeholderRepository{db: database}
}

// Create inserts a new stakeholder
func (r *StakeholderRepository) Create(ctx context.Context, stakeholder *Stakeholder) error {
	query := `
		INSERT INTO program_stakeholders (
			stakeholder_id, program_id, person_name, email, role, organization,
			stakeholder_type, is_internal, engagement_level, department, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		stakeholder.StakeholderID,
		stakeholder.ProgramID,
		stakeholder.PersonName,
		stakeholder.Email,
		stakeholder.Role,
		stakeholder.Organization,
		stakeholder.StakeholderType,
		stakeholder.IsInternal,
		stakeholder.EngagementLevel,
		stakeholder.Department,
		stakeholder.Notes,
	)

	if err != nil {
		return fmt.Errorf("failed to create stakeholder: %w", err)
	}

	return nil
}

// GetByID retrieves a stakeholder by ID
func (r *StakeholderRepository) GetByID(ctx context.Context, stakeholderID uuid.UUID) (*Stakeholder, error) {
	query := `
		SELECT stakeholder_id, program_id, person_name, email, role, organization,
		       stakeholder_type, is_internal, engagement_level, department, notes,
		       created_at, updated_at, deleted_at
		FROM program_stakeholders
		WHERE stakeholder_id = $1 AND deleted_at IS NULL
	`

	var s Stakeholder
	err := r.db.QueryRowContext(ctx, query, stakeholderID).Scan(
		&s.StakeholderID,
		&s.ProgramID,
		&s.PersonName,
		&s.Email,
		&s.Role,
		&s.Organization,
		&s.StakeholderType,
		&s.IsInternal,
		&s.EngagementLevel,
		&s.Department,
		&s.Notes,
		&s.CreatedAt,
		&s.UpdatedAt,
		&s.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("stakeholder not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get stakeholder: %w", err)
	}

	return &s, nil
}

// ListByProgram retrieves all stakeholders for a program with optional filtering
func (r *StakeholderRepository) ListByProgram(ctx context.Context, filter StakeholderFilter) ([]Stakeholder, error) {
	// Build query with filters
	query := `
		SELECT stakeholder_id, program_id, person_name, email, role, organization,
		       stakeholder_type, is_internal, engagement_level, department, notes,
		       created_at, updated_at, deleted_at
		FROM program_stakeholders
		WHERE program_id = $1 AND deleted_at IS NULL
	`
	args := []interface{}{filter.ProgramID}
	argPos := 2

	// Add optional filters
	if filter.StakeholderType != "" {
		query += fmt.Sprintf(" AND stakeholder_type = $%d", argPos)
		args = append(args, filter.StakeholderType)
		argPos++
	}

	if filter.IsInternal != nil {
		query += fmt.Sprintf(" AND is_internal = $%d", argPos)
		args = append(args, *filter.IsInternal)
		argPos++
	}

	if filter.EngagementLevel != "" {
		query += fmt.Sprintf(" AND engagement_level = $%d", argPos)
		args = append(args, filter.EngagementLevel)
		argPos++
	}

	// Add ordering and pagination
	query += " ORDER BY person_name ASC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filter.Limit)
		argPos++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list stakeholders: %w", err)
	}
	defer rows.Close()

	stakeholders := make([]Stakeholder, 0)
	for rows.Next() {
		var s Stakeholder
		err := rows.Scan(
			&s.StakeholderID,
			&s.ProgramID,
			&s.PersonName,
			&s.Email,
			&s.Role,
			&s.Organization,
			&s.StakeholderType,
			&s.IsInternal,
			&s.EngagementLevel,
			&s.Department,
			&s.Notes,
			&s.CreatedAt,
			&s.UpdatedAt,
			&s.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stakeholder: %w", err)
		}
		stakeholders = append(stakeholders, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating stakeholders: %w", err)
	}

	return stakeholders, nil
}

// ListInternal retrieves internal stakeholders for a program
func (r *StakeholderRepository) ListInternal(ctx context.Context, programID uuid.UUID) ([]Stakeholder, error) {
	filter := StakeholderFilter{
		ProgramID:  programID,
		IsInternal: boolPtr(true),
		Limit:      100, // Reasonable limit for internal stakeholders
	}
	return r.ListByProgram(ctx, filter)
}

// ListExternal retrieves external stakeholders for a program
func (r *StakeholderRepository) ListExternal(ctx context.Context, programID uuid.UUID) ([]Stakeholder, error) {
	filter := StakeholderFilter{
		ProgramID:  programID,
		IsInternal: boolPtr(false),
		Limit:      100, // Reasonable limit for external stakeholders
	}
	return r.ListByProgram(ctx, filter)
}

// Update modifies an existing stakeholder
func (r *StakeholderRepository) Update(ctx context.Context, stakeholderID uuid.UUID, req UpdateStakeholderRequest) error {
	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.PersonName != nil {
		updates = append(updates, fmt.Sprintf("person_name = $%d", argPos))
		args = append(args, *req.PersonName)
		argPos++
	}
	if req.Email != nil {
		updates = append(updates, fmt.Sprintf("email = $%d", argPos))
		args = append(args, sqlNullString(*req.Email))
		argPos++
	}
	if req.Role != nil {
		updates = append(updates, fmt.Sprintf("role = $%d", argPos))
		args = append(args, sqlNullString(*req.Role))
		argPos++
	}
	if req.Organization != nil {
		updates = append(updates, fmt.Sprintf("organization = $%d", argPos))
		args = append(args, sqlNullString(*req.Organization))
		argPos++
	}
	if req.StakeholderType != nil {
		updates = append(updates, fmt.Sprintf("stakeholder_type = $%d", argPos))
		args = append(args, *req.StakeholderType)
		argPos++
	}
	if req.IsInternal != nil {
		updates = append(updates, fmt.Sprintf("is_internal = $%d", argPos))
		args = append(args, *req.IsInternal)
		argPos++
	}
	if req.EngagementLevel != nil {
		updates = append(updates, fmt.Sprintf("engagement_level = $%d", argPos))
		args = append(args, sqlNullString(*req.EngagementLevel))
		argPos++
	}
	if req.Department != nil {
		updates = append(updates, fmt.Sprintf("department = $%d", argPos))
		args = append(args, sqlNullString(*req.Department))
		argPos++
	}
	if req.Notes != nil {
		updates = append(updates, fmt.Sprintf("notes = $%d", argPos))
		args = append(args, sqlNullString(*req.Notes))
		argPos++
	}

	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	// Add updated_at
	updates = append(updates, fmt.Sprintf("updated_at = $%d", argPos))
	args = append(args, "NOW()")
	argPos++

	// Add stakeholder ID
	args = append(args, stakeholderID)

	query := fmt.Sprintf(`
		UPDATE program_stakeholders
		SET %s
		WHERE stakeholder_id = $%d AND deleted_at IS NULL
	`, strings.Join(updates, ", "), argPos)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update stakeholder: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("stakeholder not found or already deleted")
	}

	return nil
}

// Delete soft-deletes a stakeholder
func (r *StakeholderRepository) Delete(ctx context.Context, stakeholderID uuid.UUID) error {
	query := `
		UPDATE program_stakeholders
		SET deleted_at = NOW()
		WHERE stakeholder_id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, stakeholderID)
	if err != nil {
		return fmt.Errorf("failed to delete stakeholder: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("stakeholder not found or already deleted")
	}

	return nil
}

// AutoLinkByName attempts to find a stakeholder by fuzzy name matching
// This is used to automatically link extracted person names to known stakeholders
func (r *StakeholderRepository) AutoLinkByName(ctx context.Context, programID uuid.UUID, personName string) (uuid.UUID, error) {
	// Try exact match first
	query := `
		SELECT stakeholder_id
		FROM program_stakeholders
		WHERE program_id = $1
		  AND LOWER(person_name) = LOWER($2)
		  AND deleted_at IS NULL
		LIMIT 1
	`

	var stakeholderID uuid.UUID
	err := r.db.QueryRowContext(ctx, query, programID, personName).Scan(&stakeholderID)
	if err == nil {
		return stakeholderID, nil
	}

	if err != sql.ErrNoRows {
		return uuid.Nil, fmt.Errorf("failed to query stakeholder: %w", err)
	}

	// Try fuzzy match using pg_trgm similarity
	query = `
		SELECT stakeholder_id
		FROM program_stakeholders
		WHERE program_id = $1
		  AND deleted_at IS NULL
		  AND similarity(person_name, $2) > 0.6
		ORDER BY similarity(person_name, $2) DESC
		LIMIT 1
	`

	err = r.db.QueryRowContext(ctx, query, programID, personName).Scan(&stakeholderID)
	if err == sql.ErrNoRows {
		return uuid.Nil, fmt.Errorf("no matching stakeholder found for: %s", personName)
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to fuzzy match stakeholder: %w", err)
	}

	return stakeholderID, nil
}

// GetSuggestions retrieves person mentions that haven't been linked to stakeholders yet
func (r *StakeholderRepository) GetSuggestions(ctx context.Context, programID uuid.UUID) ([]PersonSuggestion, error) {
	query := `
		WITH program_data AS (
			SELECT
				p.program_id,
				p.internal_organization
			FROM programs p
			WHERE p.program_id = $1
		),
		program_aliases AS (
			SELECT
				unnest(string_to_array(pd.internal_organization, ',')) as alias
			FROM program_data pd
		)
		SELECT
			ap.person_id,
			ap.person_name,
			ap.person_role,
			ap.person_organization,
			ap.confidence_score,
			COUNT(DISTINCT ap.artifact_id) as artifact_count,
			SUM(ap.mention_count) as total_mentions,
			MAX(ap.extracted_at) as last_mentioned,
			COALESCE(
				(
					SELECT stakeholder_id
					FROM program_stakeholders ps
					WHERE ps.program_id = (SELECT program_id FROM program_data)
					  AND ps.deleted_at IS NULL
					  AND similarity(ps.person_name, ap.person_name) > 0.6
					ORDER BY similarity(ps.person_name, ap.person_name) DESC
					LIMIT 1
				),
				NULL
			) as suggested_stakeholder_id,
			COALESCE(
				(
					SELECT similarity(ps.person_name, ap.person_name)
					FROM program_stakeholders ps
					WHERE ps.program_id = (SELECT program_id FROM program_data)
					  AND ps.deleted_at IS NULL
					  AND similarity(ps.person_name, ap.person_name) > 0.6
					ORDER BY similarity(ps.person_name, ap.person_name) DESC
					LIMIT 1
				),
				0
			) as similarity_score,
			CASE
				WHEN ap.person_organization IS NULL THEN 'internal'
				WHEN EXISTS (
					SELECT 1 FROM program_aliases pa
					WHERE LOWER(TRIM(pa.alias)) = LOWER(TRIM(ap.person_organization))
					   OR LOWER(TRIM(ap.person_organization)) LIKE '%' || LOWER(TRIM(pa.alias)) || '%'
					   OR LOWER(TRIM(pa.alias)) LIKE '%' || LOWER(TRIM(ap.person_organization)) || '%'
				) THEN 'internal'
				ELSE 'external'
			END as suggested_stakeholder_type,
			CASE
				WHEN ap.person_organization IS NULL THEN TRUE
				WHEN EXISTS (
					SELECT 1 FROM program_aliases pa
					WHERE LOWER(TRIM(pa.alias)) = LOWER(TRIM(ap.person_organization))
					   OR LOWER(TRIM(ap.person_organization)) LIKE '%' || LOWER(TRIM(pa.alias)) || '%'
					   OR LOWER(TRIM(pa.alias)) LIKE '%' || LOWER(TRIM(ap.person_organization)) || '%'
				) THEN TRUE
				ELSE FALSE
			END as suggested_is_internal
		FROM artifact_persons ap
		JOIN artifacts a ON ap.artifact_id = a.artifact_id
		CROSS JOIN program_data pd
		WHERE a.program_id = pd.program_id
		  AND ap.stakeholder_id IS NULL
		GROUP BY ap.person_id, ap.person_name, ap.person_role, ap.person_organization, ap.confidence_score
		ORDER BY total_mentions DESC, last_mentioned DESC
	`

	rows, err := r.db.QueryContext(ctx, query, programID)
	if err != nil {
		return nil, fmt.Errorf("failed to get suggestions: %w", err)
	}
	defer rows.Close()

	suggestions := make([]PersonSuggestion, 0)
	for rows.Next() {
		var s PersonSuggestion
		err := rows.Scan(
			&s.PersonID,
			&s.PersonName,
			&s.PersonRole,
			&s.PersonOrganization,
			&s.ConfidenceScore,
			&s.ArtifactCount,
			&s.TotalMentions,
			&s.LastMentioned,
			&s.SuggestedStakeholderID,
			&s.SimilarityScore,
			&s.SuggestedStakeholderType,
			&s.SuggestedIsInternal,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan suggestion: %w", err)
		}

		// Get artifacts for this person
		artifactsQuery := `
			SELECT a.artifact_id, a.filename, ap.mention_count
			FROM artifact_persons ap
			JOIN artifacts a ON ap.artifact_id = a.artifact_id
			WHERE ap.person_id = $1
			ORDER BY ap.mention_count DESC, a.uploaded_at DESC
			LIMIT 5
		`

		artifactRows, err := r.db.QueryContext(ctx, artifactsQuery, s.PersonID)
		if err != nil {
			return nil, fmt.Errorf("failed to get artifacts for person: %w", err)
		}

		artifacts := make([]SuggestionArtifact, 0)
		for artifactRows.Next() {
			var a SuggestionArtifact
			err := artifactRows.Scan(&a.ArtifactID, &a.Filename, &a.MentionCount)
			if err != nil {
				artifactRows.Close()
				return nil, fmt.Errorf("failed to scan artifact: %w", err)
			}
			artifacts = append(artifacts, a)
		}
		artifactRows.Close()

		s.Artifacts = artifacts
		suggestions = append(suggestions, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating suggestions: %w", err)
	}

	return suggestions, nil
}

// LinkPersonToStakeholder links an artifact person to a stakeholder
func (r *StakeholderRepository) LinkPersonToStakeholder(ctx context.Context, personID uuid.UUID, stakeholderID uuid.UUID) error {
	query := `
		UPDATE artifact_persons
		SET stakeholder_id = $1
		WHERE person_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, stakeholderID, personID)
	if err != nil {
		return fmt.Errorf("failed to link person to stakeholder: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("person not found")
	}

	return nil
}

// GroupPersonSuggestions creates merge groups for similar person mentions
func (r *StakeholderRepository) GroupPersonSuggestions(ctx context.Context, programID uuid.UUID) error {
	// Build similarity matrix and cluster persons
	query := `
		WITH unlinked AS (
			SELECT DISTINCT
				ap.person_id,
				ap.person_name,
				ap.person_role,
				ap.person_organization,
				ap.confidence_score
			FROM artifact_persons ap
			JOIN artifacts a ON ap.artifact_id = a.artifact_id
			WHERE a.program_id = $1
			  AND ap.stakeholder_id IS NULL
		),
		similarity_pairs AS (
			SELECT
				p1.person_id as person_id_1,
				p2.person_id as person_id_2,
				p1.person_name as name_1,
				p2.person_name as name_2,
				p1.person_role as role_1,
				p2.person_role as role_2,
				p1.person_organization as org_1,
				p2.person_organization as org_2,
				similarity(p1.person_name, p2.person_name) as name_similarity,
				CASE
					WHEN p1.person_organization IS NOT NULL
					 AND p2.person_organization IS NOT NULL
					 AND p1.person_organization = p2.person_organization THEN 1.0
					ELSE 0.0
				END as org_match
			FROM unlinked p1
			CROSS JOIN unlinked p2
			WHERE p1.person_id < p2.person_id
			  AND (
				  similarity(p1.person_name, p2.person_name) > 0.7
				  OR (
					  similarity(p1.person_name, p2.person_name) > 0.5
					  AND p1.person_organization IS NOT NULL
					  AND p2.person_organization IS NOT NULL
					  AND p1.person_organization = p2.person_organization
				  )
			  )
		)
		SELECT * FROM similarity_pairs
		ORDER BY name_similarity DESC
	`

	rows, err := r.db.QueryContext(ctx, query, programID)
	if err != nil {
		return fmt.Errorf("failed to build similarity matrix: %w", err)
	}
	defer rows.Close()

	// Build adjacency map for clustering
	type edge struct {
		p1    uuid.UUID
		p2    uuid.UUID
		score float64
	}

	var edges []edge
	for rows.Next() {
		var e edge
		var name1, name2, role1, role2, org1, org2 sql.NullString
		var orgMatch float64

		if err := rows.Scan(
			&e.p1, &e.p2, &name1, &name2, &role1, &role2, &org1, &org2,
			&e.score, &orgMatch,
		); err != nil {
			return fmt.Errorf("failed to scan similarity pair: %w", err)
		}

		edges = append(edges, e)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating similarity pairs: %w", err)
	}

	// Use Union-Find to cluster connected persons
	parent := make(map[uuid.UUID]uuid.UUID)
	rank := make(map[uuid.UUID]int)

	var find func(uuid.UUID) uuid.UUID
	find = func(x uuid.UUID) uuid.UUID {
		if _, exists := parent[x]; !exists {
			parent[x] = x
			rank[x] = 0
		}
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}

	union := func(x, y uuid.UUID) {
		rootX := find(x)
		rootY := find(y)
		if rootX == rootY {
			return
		}
		if rank[rootX] < rank[rootY] {
			parent[rootX] = rootY
		} else if rank[rootX] > rank[rootY] {
			parent[rootY] = rootX
		} else {
			parent[rootY] = rootX
			rank[rootX]++
		}
	}

	// Cluster persons using edges
	for _, e := range edges {
		union(e.p1, e.p2)
	}

	// Group persons by cluster root
	clusters := make(map[uuid.UUID][]uuid.UUID)
	for personID := range parent {
		root := find(personID)
		clusters[root] = append(clusters[root], personID)
	}

	// Create merge groups for clusters with 2+ persons
	for root, members := range clusters {
		if len(members) < 2 {
			continue
		}

		// Get the most common/confident name
		suggestedName, hasRoleConflict, hasOrgConflict, err := r.analyzeMemberConflicts(ctx, members)
		if err != nil {
			return fmt.Errorf("failed to analyze member conflicts: %w", err)
		}

		// Create merge group
		groupID := uuid.New()
		insertGroup := `
			INSERT INTO person_merge_groups (
				group_id, program_id, suggested_name, status,
				has_role_conflicts, has_org_conflicts
			) VALUES ($1, $2, $3, 'pending', $4, $5)
		`

		_, err = r.db.ExecContext(ctx, insertGroup,
			groupID, programID, suggestedName, hasRoleConflict, hasOrgConflict,
		)
		if err != nil {
			return fmt.Errorf("failed to create merge group: %w", err)
		}

		// Insert members
		insertMember := `
			INSERT INTO person_merge_group_members (
				group_id, person_id, similarity_score, matching_method
			) VALUES ($1, $2, $3, 'fuzzy_name')
		`

		for _, personID := range members {
			// Calculate similarity to root/anchor person
			similarity := 1.0
			if personID != root {
				// For simplicity, use 0.8 for non-root members
				// In production, you'd calculate actual similarity
				similarity = 0.8
			}

			_, err := r.db.ExecContext(ctx, insertMember, groupID, personID, similarity)
			if err != nil {
				return fmt.Errorf("failed to insert group member: %w", err)
			}
		}
	}

	return nil
}

// analyzeMemberConflicts checks for role/org conflicts and returns suggested name
func (r *StakeholderRepository) analyzeMemberConflicts(ctx context.Context, personIDs []uuid.UUID) (string, bool, bool, error) {
	if len(personIDs) == 0 {
		return "", false, false, fmt.Errorf("no persons provided")
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(personIDs))
	args := make([]interface{}, len(personIDs))
	for i, id := range personIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT
			person_name,
			person_role,
			person_organization,
			confidence_score,
			COUNT(*) as name_count
		FROM artifact_persons
		WHERE person_id IN (%s)
		GROUP BY person_name, person_role, person_organization, confidence_score
		ORDER BY name_count DESC, confidence_score DESC NULLS LAST
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return "", false, false, fmt.Errorf("failed to query person details: %w", err)
	}
	defer rows.Close()

	var suggestedName string
	distinctRoles := make(map[string]bool)
	distinctOrgs := make(map[string]bool)
	first := true

	for rows.Next() {
		var name string
		var role, org sql.NullString
		var confidence sql.NullFloat64
		var count int

		if err := rows.Scan(&name, &role, &org, &confidence, &count); err != nil {
			return "", false, false, fmt.Errorf("failed to scan person details: %w", err)
		}

		if first {
			suggestedName = name
			first = false
		}

		if role.Valid && role.String != "" {
			distinctRoles[role.String] = true
		}
		if org.Valid && org.String != "" {
			distinctOrgs[org.String] = true
		}
	}

	if err := rows.Err(); err != nil {
		return "", false, false, fmt.Errorf("error iterating person details: %w", err)
	}

	hasRoleConflict := len(distinctRoles) > 1
	hasOrgConflict := len(distinctOrgs) > 1

	return suggestedName, hasRoleConflict, hasOrgConflict, nil
}

// GetGroupedSuggestions retrieves all merge groups with their members and statistics
func (r *StakeholderRepository) GetGroupedSuggestions(ctx context.Context, programID uuid.UUID) ([]GroupedSuggestion, error) {
	// Get all pending groups for this program
	groupsQuery := `
		SELECT
			pmg.group_id,
			pmg.suggested_name,
			pmg.status,
			pmg.has_role_conflicts,
			pmg.has_org_conflicts,
			COUNT(DISTINCT pgm.person_id) as total_persons,
			COUNT(DISTINCT ap.artifact_id) as total_artifacts,
			SUM(ap.mention_count) as total_mentions,
			AVG(COALESCE(ap.confidence_score, 0)) as average_confidence,
			MAX(ap.extracted_at) as last_mentioned
		FROM person_merge_groups pmg
		JOIN person_merge_group_members pgm ON pmg.group_id = pgm.group_id
		JOIN artifact_persons ap ON pgm.person_id = ap.person_id
		WHERE pmg.program_id = $1
		  AND pmg.status = 'pending'
		GROUP BY pmg.group_id
		ORDER BY total_mentions DESC, last_mentioned DESC
	`

	rows, err := r.db.QueryContext(ctx, groupsQuery, programID)
	if err != nil {
		return nil, fmt.Errorf("failed to query grouped suggestions: %w", err)
	}
	defer rows.Close()

	var groups []GroupedSuggestion
	for rows.Next() {
		var g GroupedSuggestion
		var avgConfidence sql.NullFloat64

		if err := rows.Scan(
			&g.GroupID, &g.SuggestedName, &g.Status,
			&g.HasRoleConflicts, &g.HasOrgConflicts,
			&g.TotalPersons, &g.TotalArtifacts, &g.TotalMentions,
			&avgConfidence, &g.LastMentioned,
		); err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}

		if avgConfidence.Valid {
			g.AverageConfidence = avgConfidence.Float64
		}

		// Get members
		members, err := r.getGroupMembers(ctx, g.GroupID)
		if err != nil {
			return nil, fmt.Errorf("failed to get group members: %w", err)
		}
		g.Members = members

		// Get conflict options if conflicts exist
		if g.HasRoleConflicts || g.HasOrgConflicts {
			roleOpts, orgOpts, err := r.getConflictOptions(ctx, g.GroupID)
			if err != nil {
				return nil, fmt.Errorf("failed to get conflict options: %w", err)
			}
			g.RoleOptions = roleOpts
			g.OrgOptions = orgOpts
		}

		// Get all context snippets
		contexts, err := r.getAllContexts(ctx, g.GroupID)
		if err != nil {
			return nil, fmt.Errorf("failed to get contexts: %w", err)
		}
		g.AllContexts = contexts

		groups = append(groups, g)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating groups: %w", err)
	}

	return groups, nil
}

// getGroupMembers retrieves all members of a merge group
func (r *StakeholderRepository) getGroupMembers(ctx context.Context, groupID uuid.UUID) ([]GroupMember, error) {
	query := `
		SELECT
			ap.person_id,
			ap.person_name,
			ap.person_role,
			ap.person_organization,
			COALESCE(ap.confidence_score, 0) as confidence_score,
			pgm.similarity_score,
			COUNT(DISTINCT ap.artifact_id) as artifact_count,
			SUM(ap.mention_count) as mention_count
		FROM person_merge_group_members pgm
		JOIN artifact_persons ap ON pgm.person_id = ap.person_id
		WHERE pgm.group_id = $1
		GROUP BY ap.person_id, ap.person_name, ap.person_role, ap.person_organization,
		         ap.confidence_score, pgm.similarity_score
		ORDER BY pgm.similarity_score DESC
	`

	rows, err := r.db.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query group members: %w", err)
	}
	defer rows.Close()

	var members []GroupMember
	for rows.Next() {
		var m GroupMember
		var confScore sql.NullFloat64

		if err := rows.Scan(
			&m.PersonID, &m.PersonName, &m.PersonRole, &m.PersonOrganization,
			&confScore, &m.SimilarityScore, &m.ArtifactCount, &m.MentionCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}

		if confScore.Valid {
			m.ConfidenceScore = confScore.Float64
		}

		members = append(members, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating members: %w", err)
	}

	return members, nil
}

// getConflictOptions retrieves distinct values for role and organization
func (r *StakeholderRepository) getConflictOptions(ctx context.Context, groupID uuid.UUID) ([]ConflictOption, []ConflictOption, error) {
	query := `
		SELECT
			ap.person_role,
			ap.person_organization,
			COUNT(*) as occurrence_count,
			AVG(COALESCE(ap.confidence_score, 0)) as avg_confidence
		FROM person_merge_group_members pgm
		JOIN artifact_persons ap ON pgm.person_id = ap.person_id
		WHERE pgm.group_id = $1
		GROUP BY ap.person_role, ap.person_organization
	`

	rows, err := r.db.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query conflict options: %w", err)
	}
	defer rows.Close()

	roleMap := make(map[string]*ConflictOption)
	orgMap := make(map[string]*ConflictOption)

	for rows.Next() {
		var role, org sql.NullString
		var count int
		var avgConf float64

		if err := rows.Scan(&role, &org, &count, &avgConf); err != nil {
			return nil, nil, fmt.Errorf("failed to scan conflict options: %w", err)
		}

		if role.Valid && role.String != "" {
			if opt, exists := roleMap[role.String]; exists {
				opt.Count += count
			} else {
				roleMap[role.String] = &ConflictOption{
					Value:      role.String,
					Count:      count,
					Confidence: avgConf,
				}
			}
		}

		if org.Valid && org.String != "" {
			if opt, exists := orgMap[org.String]; exists {
				opt.Count += count
			} else {
				orgMap[org.String] = &ConflictOption{
					Value:      org.String,
					Count:      count,
					Confidence: avgConf,
				}
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating conflict options: %w", err)
	}

	// Convert maps to slices
	var roleOptions []ConflictOption
	for _, opt := range roleMap {
		roleOptions = append(roleOptions, *opt)
	}

	var orgOptions []ConflictOption
	for _, opt := range orgMap {
		orgOptions = append(orgOptions, *opt)
	}

	return roleOptions, orgOptions, nil
}

// getAllContexts retrieves all context snippets for a merge group
func (r *StakeholderRepository) getAllContexts(ctx context.Context, groupID uuid.UUID) ([]ContextMention, error) {
	query := `
		SELECT
			a.artifact_id,
			a.filename,
			a.uploaded_at,
			ap.person_name,
			ap.context_snippets
		FROM person_merge_group_members pgm
		JOIN artifact_persons ap ON pgm.person_id = ap.person_id
		JOIN artifacts a ON ap.artifact_id = a.artifact_id
		WHERE pgm.group_id = $1
		ORDER BY a.uploaded_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query contexts: %w", err)
	}
	defer rows.Close()

	var contexts []ContextMention
	for rows.Next() {
		var artifactID uuid.UUID
		var filename, personName string
		var uploadedAt sql.NullTime
		var snippetsJSON sql.NullString

		if err := rows.Scan(&artifactID, &filename, &uploadedAt, &personName, &snippetsJSON); err != nil {
			return nil, fmt.Errorf("failed to scan context: %w", err)
		}

		// If snippets exist, parse them (they're stored as JSONB array)
		// For simplicity, we'll use the first snippet or a placeholder
		snippet := ""
		if snippetsJSON.Valid && snippetsJSON.String != "" {
			// In production, you'd parse the JSONB array properly
			// For now, use placeholder
			snippet = strings.TrimPrefix(snippetsJSON.String, "[\"")
			snippet = strings.TrimSuffix(snippet, "\"]")
			if len(snippet) > 200 {
				snippet = snippet[:200] + "..."
			}
		}

		contexts = append(contexts, ContextMention{
			ArtifactID:   artifactID,
			ArtifactName: filename,
			UploadedAt:   uploadedAt.Time,
			Snippet:      snippet,
			PersonName:   personName,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating contexts: %w", err)
	}

	return contexts, nil
}

// ConfirmMergeGroup confirms a merge group and optionally creates a stakeholder
func (r *StakeholderRepository) ConfirmMergeGroup(ctx context.Context, groupID uuid.UUID, req ConfirmMergeGroupRequest) (*Stakeholder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update group with resolved values
	updateGroup := `
		UPDATE person_merge_groups
		SET resolved_name = $1,
		    resolved_role = $2,
		    resolved_organization = $3,
		    status = CASE
		        WHEN $4 THEN 'merged'
		        ELSE 'confirmed'
		    END,
		    updated_at = NOW()
		WHERE group_id = $5
	`

	_, err = tx.ExecContext(ctx, updateGroup,
		req.SelectedName,
		req.SelectedRole,
		req.SelectedOrganization,
		req.CreateStakeholder,
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update merge group: %w", err)
	}

	var stakeholder *Stakeholder

	if req.CreateStakeholder {
		// Get program_id and internal_organization from group
		var programID uuid.UUID
		var internalOrg string
		err = tx.QueryRowContext(ctx, `
			SELECT pmg.program_id, p.internal_organization
			FROM person_merge_groups pmg
			JOIN programs p ON pmg.program_id = p.program_id
			WHERE pmg.group_id = $1
		`, groupID).Scan(&programID, &internalOrg)
		if err != nil {
			return nil, fmt.Errorf("failed to get program ID: %w", err)
		}

		// Determine stakeholder type based on intelligent org matching
		stakeholderType := "internal"
		isInternal := true
		if req.SelectedOrganization != nil && *req.SelectedOrganization != "" {
			// Check if organization matches internal org aliases
			isInternal = matchesInternalOrg(*req.SelectedOrganization, internalOrg)
			if isInternal {
				stakeholderType = "internal"
			} else {
				stakeholderType = "external"
			}
		}

		// Create stakeholder
		stakeholderID := uuid.New()
		insertStakeholder := `
			INSERT INTO program_stakeholders (
				stakeholder_id, program_id, person_name, role, organization,
				stakeholder_type, is_internal, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
			RETURNING stakeholder_id, program_id, person_name, role, organization,
			          stakeholder_type, is_internal, created_at, updated_at
		`

		stakeholder = &Stakeholder{}
		err = tx.QueryRowContext(ctx, insertStakeholder,
			stakeholderID,
			programID,
			req.SelectedName,
			req.SelectedRole,
			req.SelectedOrganization,
			stakeholderType,
			isInternal,
		).Scan(
			&stakeholder.StakeholderID,
			&stakeholder.ProgramID,
			&stakeholder.PersonName,
			&stakeholder.Role,
			&stakeholder.Organization,
			&stakeholder.StakeholderType,
			&stakeholder.IsInternal,
			&stakeholder.CreatedAt,
			&stakeholder.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create stakeholder: %w", err)
		}

		// Link all persons in group to new stakeholder
		linkPersons := `
			UPDATE artifact_persons
			SET stakeholder_id = $1
			WHERE person_id IN (
				SELECT person_id
				FROM person_merge_group_members
				WHERE group_id = $2
			)
		`

		_, err = tx.ExecContext(ctx, linkPersons, stakeholderID, groupID)
		if err != nil {
			return nil, fmt.Errorf("failed to link persons to stakeholder: %w", err)
		}

		// Update group with stakeholder reference
		updateGroupStakeholder := `
			UPDATE person_merge_groups
			SET merged_stakeholder_id = $1,
			    merged_at = NOW()
			WHERE group_id = $2
		`

		_, err = tx.ExecContext(ctx, updateGroupStakeholder, stakeholderID, groupID)
		if err != nil {
			return nil, fmt.Errorf("failed to update group with stakeholder: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return stakeholder, nil
}

// RejectMergeGroup marks a merge group as rejected
func (r *StakeholderRepository) RejectMergeGroup(ctx context.Context, groupID uuid.UUID) error {
	query := `
		UPDATE person_merge_groups
		SET status = 'rejected',
		    updated_at = NOW()
		WHERE group_id = $1
	`

	result, err := r.db.ExecContext(ctx, query, groupID)
	if err != nil {
		return fmt.Errorf("failed to reject merge group: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("merge group not found")
	}

	return nil
}

// ModifyGroupMembers adds or removes persons from a merge group
func (r *StakeholderRepository) ModifyGroupMembers(ctx context.Context, groupID uuid.UUID, req ModifyGroupMembersRequest) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Remove persons
	if len(req.RemovePersonIDs) > 0 {
		placeholders := make([]string, len(req.RemovePersonIDs))
		args := []interface{}{groupID}
		for i, id := range req.RemovePersonIDs {
			placeholders[i] = fmt.Sprintf("$%d", i+2)
			args = append(args, id)
		}

		deleteQuery := fmt.Sprintf(`
			DELETE FROM person_merge_group_members
			WHERE group_id = $1 AND person_id IN (%s)
		`, strings.Join(placeholders, ","))

		_, err := tx.ExecContext(ctx, deleteQuery, args...)
		if err != nil {
			return fmt.Errorf("failed to remove members: %w", err)
		}
	}

	// Add persons
	if len(req.AddPersonIDs) > 0 {
		insertQuery := `
			INSERT INTO person_merge_group_members (group_id, person_id, similarity_score, matching_method)
			VALUES ($1, $2, $3, 'manual')
		`

		for _, personID := range req.AddPersonIDs {
			_, err := tx.ExecContext(ctx, insertQuery, groupID, personID, 0.5)
			if err != nil {
				return fmt.Errorf("failed to add member: %w", err)
			}
		}
	}

	// Update group timestamp
	_, err = tx.ExecContext(ctx, `
		UPDATE person_merge_groups SET updated_at = NOW() WHERE group_id = $1
	`, groupID)
	if err != nil {
		return fmt.Errorf("failed to update group timestamp: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetLinkedArtifacts retrieves all artifacts where a stakeholder is mentioned
func (r *StakeholderRepository) GetLinkedArtifacts(ctx context.Context, stakeholderID uuid.UUID) ([]LinkedArtifact, error) {
	query := `
		SELECT DISTINCT
			a.artifact_id,
			a.filename,
			a.uploaded_at,
			SUM(ap.mention_count) as total_mentions
		FROM artifact_persons ap
		JOIN artifacts a ON ap.artifact_id = a.artifact_id
		WHERE ap.stakeholder_id = $1
		GROUP BY a.artifact_id, a.filename, a.uploaded_at
		ORDER BY a.uploaded_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, stakeholderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get linked artifacts: %w", err)
	}
	defer rows.Close()

	var artifacts []LinkedArtifact
	for rows.Next() {
		var a LinkedArtifact
		if err := rows.Scan(&a.ArtifactID, &a.Filename, &a.UploadedAt, &a.MentionCount); err != nil {
			return nil, fmt.Errorf("failed to scan artifact: %w", err)
		}
		artifacts = append(artifacts, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating artifacts: %w", err)
	}

	return artifacts, nil
}

// Helper functions

// matchesInternalOrg checks if an organization name matches any internal org alias
func matchesInternalOrg(orgName string, internalOrgAliases string) bool {
	aliases := strings.Split(internalOrgAliases, ",")
	orgLower := strings.ToLower(strings.TrimSpace(orgName))

	for _, alias := range aliases {
		aliasLower := strings.ToLower(strings.TrimSpace(alias))
		if orgLower == aliasLower ||
			strings.Contains(orgLower, aliasLower) ||
			strings.Contains(aliasLower, orgLower) {
			return true
		}
	}
	return false
}

func boolPtr(b bool) *bool {
	return &b
}

func sqlNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
