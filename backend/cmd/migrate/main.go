package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cerberus/backend/internal/platform/db"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Parse flags
	command := flag.String("cmd", "up", "Command to run: up, down, status")
	flag.Parse()

	// Get database configuration
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnv("DB_NAME", "cerberus")
	dbUser := getEnv("DB_USER", "cerberus")
	dbPassword := getEnv("DB_PASSWORD", "cerberus_dev")

	// Connect to database
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	database, err := db.Connect(dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	log.Println("Database connection established")

	// Create migrations table if it doesn't exist
	_, err = database.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create migrations table: %v", err)
	}

	// Get migrations directory
	migrationsDir := "migrations"
	if len(os.Args) > 2 {
		migrationsDir = os.Args[2]
	}

	switch *command {
	case "up":
		runMigrationsUp(database, migrationsDir)
	case "down":
		runMigrationsDown(database, migrationsDir)
	case "status":
		showMigrationStatus(database, migrationsDir)
	default:
		log.Fatalf("Unknown command: %s", *command)
	}
}

func runMigrationsUp(db *db.DB, migrationsDir string) {
	// Get all migration files
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		log.Fatalf("Failed to list migrations: %v", err)
	}

	sort.Strings(files)

	for _, file := range files {
		version := filepath.Base(file)
		version = strings.TrimSuffix(version, ".sql")

		// Check if already applied
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists)
		if err != nil {
			log.Fatalf("Failed to check migration status: %v", err)
		}

		if exists {
			log.Printf("Migration %s already applied, skipping", version)
			continue
		}

		// Read migration file
		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("Failed to read migration file %s: %v", file, err)
		}

		// Execute migration
		log.Printf("Applying migration %s...", version)
		_, err = db.Exec(string(content))
		if err != nil {
			log.Fatalf("Failed to apply migration %s: %v", version, err)
		}

		// Record migration
		_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version)
		if err != nil {
			log.Fatalf("Failed to record migration %s: %v", version, err)
		}

		log.Printf("Migration %s applied successfully", version)
	}

	log.Println("All migrations applied successfully")
}

func runMigrationsDown(db *db.DB, migrationsDir string) {
	log.Println("Migration down not implemented yet")
}

func showMigrationStatus(db *db.DB, migrationsDir string) {
	// Get applied migrations
	rows, err := db.Query("SELECT version, applied_at FROM schema_migrations ORDER BY version")
	if err != nil {
		log.Fatalf("Failed to query migrations: %v", err)
	}
	defer rows.Close()

	log.Println("\nApplied migrations:")
	log.Println("==================")

	for rows.Next() {
		var version string
		var appliedAt string
		err := rows.Scan(&version, &appliedAt)
		if err != nil {
			log.Fatalf("Failed to scan migration: %v", err)
		}
		log.Printf("%s (applied at %s)", version, appliedAt)
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
