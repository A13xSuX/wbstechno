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

// создает строку подключения к БД
func buildConnectionString(dbConfig config.DatabaseConfig) string {
	return fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s",
		dbConfig.User, dbConfig.Password, dbConfig.DBName, dbConfig.Host, dbConfig.Port, dbConfig.SSLMode)
}

// создает таблицу для отслеживания миграций
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

// возвращает список примененных миграций
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

// возвращает доступные миграции
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

	result := make([]Migration, 0, len(migrations))
	for _, m := range migrations {
		result = append(result, m)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})

	return result, nil
}

// применяет одну миграцию
func applyMigration(db *sql.DB, migration Migration) error {
	content, err := os.ReadFile(migration.UpFile)
	if err != nil {
		return fmt.Errorf("не удалось прочитать файл миграций: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(string(content)); err != nil {
		return fmt.Errorf("ошибка выполнения миграций: %v", err)
	}

	// запись миграции
	_, err = tx.Exec("INSERT INTO schema_migrations (version, name) VALUES ($1, $2)",
		migration.Version, migration.Name)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// применяет все ожидающие миграции
func applyAllMigrations(db *sql.DB) {
	applied, err := getAppliedMigrations(db)
	if err != nil {
		log.Fatalf("Не удалось получить примененные миграции: %v", err)
	}

	available, err := getAvailableMigrations()
	if err != nil {
		log.Fatalf("Не удалось получить доступные миграции: %v", err)
	}

	appliedCount := 0
	for _, migration := range available {
		if applied[migration.Version] {
			continue
		}

		log.Printf("Применение миграции: %d_%s", migration.Version, migration.Name)
		if err := applyMigration(db, migration); err != nil {
			log.Fatalf("Не удалось применить миграцию %d: %v", migration.Version, err)
		}

		appliedCount++
		log.Printf("Миграция применена: %d_%s", migration.Version, migration.Name)
	}

	if appliedCount == 0 {
		log.Println("Нет миграций для применения")
	} else {
		log.Printf("Применено %d миграция/миграций", appliedCount)
	}
}

// структура миграции
type Migration struct {
	Version  int
	Name     string
	UpFile   string
	DownFile string
}

// применяет все миграции при запуске приложения
func RunMigrations(cfg config.DatabaseConfig) error {
	db, err := sql.Open("postgres", buildConnectionString(cfg))
	if err != nil {
		return fmt.Errorf("Не удалось подключиться к бд: %v", err)
	}
	defer db.Close()

	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("Ошибка создания файла миграций: %v", err)
	}

	log.Println("Применение миграцй")
	applyAllMigrations(db)
	return nil
}
