package cache

import (
	"database/sql"
	"order-service/internal/database"
	"time"
)

// интерфейс для кэша
type Cache interface {
	Get(orderUID string) (database.Order, bool)
	Set(order database.Order)
	Delete(orderUID string)
	Size() int
	Cleanup(ttl time.Duration)
	Stop()
	Range(f func(key, value interface{}) bool)
}

// интерфейс для восстановления кэша из БД
type CacheRestorer interface {
	RestoreCacheFromDB(db *sql.DB, cache Cache, limit int) error
}
