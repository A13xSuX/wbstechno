package service

import (
	"order-service/internal/database"
	"time"
)

// интерфейс для обработки заказов
type OrderProcessor interface {
	ProcessOrder(message []byte) error
	GetOrder(orderUID string) (database.Order, error)
	ValidateOrder(order database.Order) error
}

// интерфейс сервиса заказов
type OrderService interface {
	OrderProcessor
	GetCacheSize() int
	CheckDBConnection() error
	RunBenchmark(orderUID string) (map[string]time.Duration, error)
	PrintCacheContents()
}
