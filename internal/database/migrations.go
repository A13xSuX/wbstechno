package database

import (
	"database/sql"
	"fmt"
	"log"
	"order-service/internal/config"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// buildConnectionString создает строку подключения к БД
func buildConnectionString(dbConfig config.DatabaseConfig) string {
	return fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s",
		dbConfig.User, dbConfig.Password, dbConfig.DBName, dbConfig.Host, dbConfig.Port, dbConfig.SSLMode)
}

// createMigrationsTable создает таблицу для отслеживания миграций
func createMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := db.Exec(query)
	return err
}

// getAppliedMigrations возвращает список примененных миграций
func getAppliedMigrations(db *sql.DB) (map[int]bool, error) {
	rows, err := db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}
	return applied, nil
}

// getAvailableMigrations возвращает доступные миграции
func getAvailableMigrations() ([]Migration, error) {
	files, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		return nil, err
	}

	migrations := make(map[int]Migration)
	for _, file := range files {
		filename := filepath.Base(file)
		parts := strings.Split(filename, "_")
		if len(parts) < 2 {
			continue
		}

		var version int
		fmt.Sscanf(parts[0], "%d", &version)

		migration := migrations[version]
		migration.Version = version

		if strings.Contains(filename, ".up.sql") {
			migration.UpFile = file
			migration.Name = strings.TrimSuffix(strings.Join(parts[1:], "_"), ".up.sql")
		} else if strings.Contains(filename, ".down.sql") {
			migration.DownFile = file
		}

		migrations[version] = migration
	}

	// Convert to slice and sort
	result := make([]Migration, 0, len(migrations))
	for _, m := range migrations {
		result = append(result, m)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})

	return result, nil
}

// applyMigration применяет одну миграцию
func applyMigration(db *sql.DB, migration Migration) error {
	content, err := os.ReadFile(migration.UpFile)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute migration
	if _, err := tx.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute migration: %v", err)
	}

	// Record migration
	_, err = tx.Exec("INSERT INTO schema_migrations (version, name) VALUES ($1, $2)",
		migration.Version, migration.Name)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// applyAllMigrations применяет все pending миграции
func applyAllMigrations(db *sql.DB) {
	applied, err := getAppliedMigrations(db)
	if err != nil {
		log.Fatalf("Failed to get applied migrations: %v", err)
	}

	available, err := getAvailableMigrations()
	if err != nil {
		log.Fatalf("Failed to get available migrations: %v", err)
	}

	appliedCount := 0
	for _, migration := range available {
		if applied[migration.Version] {
			continue
		}

		log.Printf("Applying migration: %d_%s", migration.Version, migration.Name)
		if err := applyMigration(db, migration); err != nil {
			log.Fatalf("Failed to apply migration %d: %v", migration.Version, err)
		}

		appliedCount++
		log.Printf("Successfully applied migration: %d_%s", migration.Version, migration.Name)
	}

	if appliedCount == 0 {
		log.Println("No migrations to apply")
	} else {
		log.Printf("Applied %d migration(s)", appliedCount)
	}
}

// Migration представляет структуру миграции
type Migration struct {
	Version  int
	Name     string
	UpFile   string
	DownFile string
}

// RunMigrations применяет все миграции при запуске приложения
func RunMigrations(cfg config.DatabaseConfig) error {
	db, err := sql.Open("postgres", buildConnectionString(cfg))
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Ensure migrations table exists
	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}

	log.Println("Applying database migrations...")
	applyAllMigrations(db)
	return nil
}
