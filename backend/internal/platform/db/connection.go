package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// DB wraps sql.DB with additional functionality
type DB struct {
	*sql.DB
}

// Connect establishes a connection to the database
func Connect(databaseURL string) (*DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)

	// Ping database to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

// EnableExtensions enables required PostgreSQL extensions
func (db *DB) EnableExtensions() error {
	extensions := []string{
		"uuid-ossp",
		"pgcrypto",
		"vector",
		"btree_gin",
		"pg_trgm",
	}

	for _, ext := range extensions {
		_, err := db.Exec(fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS \"%s\"", ext))
		if err != nil {
			return fmt.Errorf("failed to enable extension %s: %w", ext, err)
		}
	}

	return nil
}
