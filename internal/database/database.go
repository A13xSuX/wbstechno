package database

import (
	"context"
	"database/sql"
	"fmt"
	"order-service/internal/config"
	"time"

	_ "github.com/lib/pq"
)

// обертка
type DB struct {
	*sql.DB
}

func ConnectDB(cfg config.DatabaseConfig) (*DB, error) {
	connStr := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s",
		cfg.User, cfg.Password, cfg.DBName, cfg.Host, cfg.Port, cfg.SSLMode)

	sqlDB, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ошибка ping: %v", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)

	return &DB{sqlDB}, nil
}

func (db *DB) CloseWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		//Закрываем все idle соединения
		db.SetMaxIdleConns(0)
		db.SetMaxOpenConns(0)

		//Закрываем основное соединение
		done <- db.DB.Close()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("таймаут закрытия БД")
	}
}
