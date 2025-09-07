package cache

import (
	"database/sql"
	"fmt"
	"log"
	"order-service/internal/database"
	"sync"
	"time"
)

type CachedOrder struct {
	Order     database.Order
	CreatedAt time.Time
}

// OrderCache реализация интерфейса Cache
type OrderCache struct {
	cache           *sync.Map
	cacheTimestamps map[string]time.Time
	mutex           sync.RWMutex
	maxSize         int
	ttl             time.Duration
	stopChan        chan struct{}
}

// NewOrderCache создает новый кэш
func NewOrderCache(maxSize int, ttl time.Duration) Cache {
	cache := &OrderCache{
		cache:           &sync.Map{},
		cacheTimestamps: make(map[string]time.Time),
		maxSize:         maxSize,
		ttl:             ttl,
		stopChan:        make(chan struct{}),
	}

	go cache.startCleanupWorker()

	return cache
}

// Get возвращает заказ из кэша
func (oc *OrderCache) Get(orderUID string) (database.Order, bool) {
	if cached, ok := oc.cache.Load(orderUID); ok {
		if cachedOrder, ok := cached.(CachedOrder); ok {
			if time.Since(cachedOrder.CreatedAt) > oc.ttl {
				oc.Delete(orderUID)
				return database.Order{}, false
			}
			return cachedOrder.Order, true
		}
	}
	return database.Order{}, false
}

// Set добавляет заказ в кэш
func (oc *OrderCache) Set(order database.Order) {
	if oc.Size() >= oc.maxSize {
		oc.removeOldest()
	}

	cachedOrder := CachedOrder{
		Order:     order,
		CreatedAt: time.Now(),
	}

	oc.cache.Store(order.OrderUID, cachedOrder)

	oc.mutex.Lock()
	oc.cacheTimestamps[order.OrderUID] = time.Now()
	oc.mutex.Unlock()
}

// Delete удаляет заказ из кэша
func (oc *OrderCache) Delete(orderUID string) {
	oc.cache.Delete(orderUID)
	oc.mutex.Lock()
	delete(oc.cacheTimestamps, orderUID)
	oc.mutex.Unlock()
}

// Size возвращает размер кэша
func (oc *OrderCache) Size() int {
	count := 0
	oc.cache.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// Cleanup очищает устаревшие записи
func (oc *OrderCache) Cleanup(ttl time.Duration) {
	oc.mutex.Lock()
	defer oc.mutex.Unlock()

	now := time.Now()
	for orderUID, createdAt := range oc.cacheTimestamps {
		if now.Sub(createdAt) > ttl {
			oc.cache.Delete(orderUID)
			delete(oc.cacheTimestamps, orderUID)
		}
	}
}

// Stop останавливает кэш
func (oc *OrderCache) Stop() {
	close(oc.stopChan)
	log.Println("Кэш остановлен")
}

// Range итерируется по элементам кэша
func (oc *OrderCache) Range(f func(key, value interface{}) bool) {
	oc.cache.Range(f)
}

// Вспомогательные методы
func (oc *OrderCache) startCleanupWorker() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			oc.Cleanup(oc.ttl)
		case <-oc.stopChan:
			return
		}
	}
}

func (oc *OrderCache) removeOldest() {
	oc.mutex.Lock()
	defer oc.mutex.Unlock()

	if len(oc.cacheTimestamps) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, timestamp := range oc.cacheTimestamps {
		if first || timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = timestamp
			first = false
		}
	}

	oc.cache.Delete(oldestKey)
	delete(oc.cacheTimestamps, oldestKey)
}

// RestoreCacheFromDB реализация интерфейса CacheRestorer
func RestoreCacheFromDB(db *sql.DB, cache Cache, limit int) error {
	fmt.Printf("Восстановление кэша, лимит: %d\n", limit)
	query := `
		SELECT o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature, 
			   o.customer_id, o.delivery_service, o.shardkey, o.sm_id, o.date_created, o.oof_shard,
			   d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
			   p.transaction, p.request_id, p.currency, p.provider, p.amount, 
			   p.payment_dt, p.bank, p.delivery_cost, p.goods_total, p.custom_fee
		FROM orders o
		LEFT JOIN delivery d ON o.order_uid = d.order_uid
		LEFT JOIN payment p ON o.order_uid = p.order_uid
		ORDER BY o.date_created DESC
		LIMIT $1
	`

	rows, err := db.Query(query, limit)
	if err != nil {
		return fmt.Errorf("ошибка запроса заказов: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var order database.Order

		err := rows.Scan(
			&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale,
			&order.InternalSignature, &order.CustomerID, &order.DeliveryService,
			&order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard,
			&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip,
			&order.Delivery.City, &order.Delivery.Address, &order.Delivery.Region,
			&order.Delivery.Email,
			&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency,
			&order.Payment.Provider, &order.Payment.Amount, &order.Payment.PaymentDt,
			&order.Payment.Bank, &order.Payment.DeliveryCost, &order.Payment.GoodsTotal,
			&order.Payment.CustomFee,
		)

		if err != nil {
			log.Printf("Ошибка сканирования заказа: %v", err)
			continue
		}

		if err := loadOrderItems(db, &order); err != nil {
			log.Printf("Ошибка загрузки товаров для заказа %s: %v", order.OrderUID, err)
			continue
		}

		cache.Set(order)
		count++
	}

	fmt.Printf("Успешно загружено %d заказов в кэш\n", count)
	return nil
}

func loadOrderItems(db *sql.DB, order *database.Order) error {
	query := `
		SELECT chrt_id, track_number, price, rid, name, 
			   sale, size, total_price, nm_id, brand, status
		FROM items 
		WHERE order_uid = $1
	`

	rows, err := db.Query(query, order.OrderUID)
	if err != nil {
		return fmt.Errorf("ошибка запроса товаров: %v", err)
	}
	defer rows.Close()

	var items []database.Item
	for rows.Next() {
		var item database.Item
		err := rows.Scan(
			&item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid, &item.Name,
			&item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status,
		)
		if err != nil {
			return fmt.Errorf("ошибка сканирования товара: %v", err)
		}
		items = append(items, item)
	}

	order.Items = items
	return nil
}
