package database

import (
	"database/sql"
	"order-service/internal/config"
	"time"
)

// интерфейс для работы с БД
type Database interface {
	ConnectDB(cfg config.DatabaseConfig) (*DB, error)
	CloseWithTimeout(timeout time.Duration) error
	Ping() error
	Begin() (*sql.Tx, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
	SetMaxOpenConns(n int)
	SetMaxIdleConns(n int)
}

// интерфейс для работы с заказами
type OrderRepository interface {
	GetDB() *sql.DB
	SaveOrder(tx *sql.Tx, order Order) error
	SaveDelivery(tx *sql.Tx, order Order) error
	SavePayment(tx *sql.Tx, order Order) error
	SaveItems(tx *sql.Tx, order Order) error
	GetOrder(orderUID string) (Order, error)
	LoadOrderItems(order *Order) error
	CheckConnection() error
}
