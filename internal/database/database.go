package database

import (
	"database/sql"
	"fmt"
	"order-service/internal/config"
	"time"

	_ "github.com/lib/pq"
)

func ConnectDB(cfg config.DatabaseConfig) (*sql.DB, error) {
	connStr := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s",
		cfg.User, cfg.Password, cfg.DBName, cfg.Host, cfg.Port, cfg.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка ping: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}
