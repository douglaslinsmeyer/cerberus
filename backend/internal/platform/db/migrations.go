package db

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations executes all pending migrations
func (db *DB) RunMigrations(migrationsDir string) error {
	// Create migrations table if it doesn't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	log.Println("Checking for pending migrations...")

	// Get all migration files
	var files []string
	err = filepath.WalkDir(migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".sql") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to list migrations: %w", err)
	}

	sort.Strings(files)

	appliedCount := 0
	for _, file := range files {
		version := filepath.Base(file)
		version = strings.TrimSuffix(version, ".sql")

		// Check if already applied
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration status: %w", err)
		}

		if exists {
			log.Printf("Migration %s already applied, skipping", version)
			continue
		}

		// Read migration file
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		// Execute migration
		log.Printf("Applying migration %s...", version)
		_, err = db.Exec(string(content))
		if err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", version, err)
		}

		// Record migration
		_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version)
		if err != nil {
			return fmt.Errorf("failed to record migration %s: %w", version, err)
		}

		log.Printf("Migration %s applied successfully", version)
		appliedCount++
	}

	if appliedCount == 0 {
		log.Println("No pending migrations found")
	} else {
		log.Printf("Applied %d migration(s) successfully", appliedCount)
	}

	return nil
}
