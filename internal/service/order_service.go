package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"order-service/internal/cache"
	"order-service/internal/database"
	"strings"
	"time"
)

type OrderService struct {
	db    *sql.DB
	cache *cache.OrderCache
}

func NewOrderService(db *sql.DB, cache *cache.OrderCache) *OrderService {
	return &OrderService{db: db, cache: cache}
}

func (s *OrderService) ProcessOrder(message []byte) error {
	var order database.Order

	if len(message) == 0 {
		return fmt.Errorf("пустое сообщение")
	}

	if err := json.Unmarshal(message, &order); err != nil {
		log.Printf("Ошибка парсинга JSON: %v\n", err)
		log.Printf("Содержимое сообщения: %s\n", string(message))
		return fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	if err := s.validateOrder(order); err != nil {
		log.Printf("Невалидный заказ: %v\n", err)
		log.Printf("Данные заказа: %+v\n", order)
		return fmt.Errorf("невалидный заказ: %v", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			log.Printf("Транзакция откачена: %s\n", order.OrderUID)
		}
	}()

	// Сохраняем заказ
	if err := s.saveOrder(tx, order); err != nil {
		return fmt.Errorf("ошибка сохранения заказа: %v", err)
	}

	// Сохраняем доставку
	if err := s.saveDelivery(tx, order); err != nil {
		return fmt.Errorf("ошибка сохранения доставки: %v", err)
	}

	// Сохраняем платеж
	if err := s.savePayment(tx, order); err != nil {
		return fmt.Errorf("ошибка сохранения платежа: %v", err)
	}

	// Сохраняем товары
	if err := s.saveItems(tx, order); err != nil {
		return fmt.Errorf("ошибка сохранения товаров: %v", err)
	}

	// Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %v", err)
	}

	// Сохраняем в кэш
	s.cache.Set(order)

	// Выводим информацию о заказе
	fmt.Printf("   Обработка заказа: %s\n", order.OrderUID)
	fmt.Printf("   Трек номер: %s\n", order.TrackNumber)
	fmt.Printf("   Клиент: %s (%s)\n", order.Delivery.Name, order.Delivery.Email)
	fmt.Printf("   Сумма заказа: %d %s\n", order.Payment.Amount, order.Payment.Currency)
	fmt.Printf("   Количество товаров: %d\n", len(order.Items))
	fmt.Printf("   Дата создания: %s\n", order.DateCreated.Format(time.RFC3339))
	fmt.Println("   --- Заказ сохранен в БД и кэш ---")

	return nil
}

func (s *OrderService) GetOrder(orderUID string) (database.Order, error) {
	// Сначала проверяем в кэше
	if order, found := s.cache.Get(orderUID); found {
		return order, nil
	}

	// Если нет в кэше, ищем в БД
	order, err := s.getOrderFromDB(orderUID)
	if err != nil {
		return database.Order{}, err
	}

	// Сохраняем в кэш для будущих запросов
	s.cache.Set(order)
	return order, nil
}

func (s *OrderService) validateOrder(order database.Order) error {
	if order.OrderUID == "" {
		return fmt.Errorf("пустой OrderUID")
	}
	if order.TrackNumber == "" {
		return fmt.Errorf("пустой TrackNumber")
	}
	if order.Entry == "" {
		return fmt.Errorf("пустой Entry")
	}

	// Валидация доставки
	if order.Delivery.Name == "" {
		return fmt.Errorf("пустое имя получателя")
	}
	if order.Delivery.Phone == "" {
		return fmt.Errorf("пустой телефон")
	}

	// Валидация платежа
	if order.Payment.Transaction == "" {
		return fmt.Errorf("пустая транзакция")
	}
	if order.Payment.Amount <= 0 {
		return fmt.Errorf("невалидная сумма платежа: %d", order.Payment.Amount)
	}
	if order.Payment.Currency == "" {
		return fmt.Errorf("пустая валюта")
	}

	// Валидация товаров
	if len(order.Items) == 0 {
		return fmt.Errorf("нет товаров в заказе")
	}
	for i, item := range order.Items {
		if item.Name == "" {
			return fmt.Errorf("пустое название товара %d", i+1)
		}
		if item.Price <= 0 {
			return fmt.Errorf("невалидная цена товара %d: %d", i+1, item.Price)
		}
	}

	// Валидация email (если указан)
	if order.Delivery.Email != "" {
		if !strings.Contains(order.Delivery.Email, "@") || !strings.Contains(order.Delivery.Email, ".") {
			return fmt.Errorf("невалидный email: %s", order.Delivery.Email)
		}
	}

	return nil
}

func (s *OrderService) saveOrder(tx *sql.Tx, order database.Order) error {
	query := `INSERT INTO orders (
		order_uid, track_number, entry, locale, internal_signature, 
		customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := tx.Exec(query,
		order.OrderUID,
		order.TrackNumber,
		order.Entry,
		order.Locale,
		order.InternalSignature,
		order.CustomerID,
		order.DeliveryService,
		order.Shardkey,
		order.SmID,
		order.DateCreated,
		order.OofShard,
	)

	if err != nil {
		return fmt.Errorf("ошибка сохранения заказа: %v", err)
	}
	return nil
}

func (s *OrderService) saveDelivery(tx *sql.Tx, order database.Order) error {
	query := `INSERT INTO delivery (
		order_uid, name, phone, zip, city, address, region, email
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := tx.Exec(query,
		order.OrderUID,
		order.Delivery.Name,
		order.Delivery.Phone,
		order.Delivery.Zip,
		order.Delivery.City,
		order.Delivery.Address,
		order.Delivery.Region,
		order.Delivery.Email,
	)

	if err != nil {
		return fmt.Errorf("ошибка сохранения доставки: %v", err)
	}
	return nil
}

func (s *OrderService) savePayment(tx *sql.Tx, order database.Order) error {
	query := `INSERT INTO payment (
		order_uid, transaction, request_id, currency, provider, amount, 
		payment_dt, bank, delivery_cost, goods_total, custom_fee
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := tx.Exec(query,
		order.OrderUID,
		order.Payment.Transaction,
		order.Payment.RequestID,
		order.Payment.Currency,
		order.Payment.Provider,
		order.Payment.Amount,
		order.Payment.PaymentDt,
		order.Payment.Bank,
		order.Payment.DeliveryCost,
		order.Payment.GoodsTotal,
		order.Payment.CustomFee,
	)

	if err != nil {
		return fmt.Errorf("ошибка сохранения платежа: %v", err)
	}
	return nil
}

func (s *OrderService) saveItems(tx *sql.Tx, order database.Order) error {
	query := `INSERT INTO items (
		order_uid, chrt_id, track_number, price, rid, name, 
		sale, size, total_price, nm_id, brand, status
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	for _, item := range order.Items {
		_, err := tx.Exec(query,
			order.OrderUID,
			item.ChrtID,
			item.TrackNumber,
			item.Price,
			item.Rid,
			item.Name,
			item.Sale,
			item.Size,
			item.TotalPrice,
			item.NmID,
			item.Brand,
			item.Status,
		)

		if err != nil {
			return fmt.Errorf("ошибка сохранения товара: %v", err)
		}
	}
	return nil
}

func (s *OrderService) getOrderFromDB(orderUID string) (database.Order, error) {
	var order database.Order

	// Основные данные заказа
	query := `
        SELECT o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature, 
               o.customer_id, o.delivery_service, o.shardkey, o.sm_id, o.date_created, o.oof_shard,
               d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
               p.transaction, p.request_id, p.currency, p.provider, p.amount, 
               p.payment_dt, p.bank, p.delivery_cost, p.goods_total, p.custom_fee
        FROM orders o
        LEFT JOIN delivery d ON o.order_uid = d.order_uid
        LEFT JOIN payment p ON o.order_uid = p.order_uid
        WHERE o.order_uid = $1
    `

	err := s.db.QueryRow(query, orderUID).Scan(
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
		return database.Order{}, err
	}

	// Загружаем товары
	if err := s.loadOrderItems(&order); err != nil {
		return database.Order{}, fmt.Errorf("ошибка загрузки товаров: %v", err)
	}

	return order, nil
}

func (s *OrderService) loadOrderItems(order *database.Order) error {
	query := `
		SELECT chrt_id, track_number, price, rid, name, 
			   sale, size, total_price, nm_id, brand, status
		FROM items 
		WHERE order_uid = $1
	`

	rows, err := s.db.Query(query, order.OrderUID)
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

// GetCacheSize возвращает размер кэша
func (s *OrderService) GetCacheSize() int {
	return s.cache.Size()
}

// CheckDBConnection проверяет соединение с БД
func (s *OrderService) CheckDBConnection() error {
	return s.db.Ping()
}

func (s *OrderService) RunBenchmark(orderUID string) (map[string]time.Duration, error) {
	results := map[string]time.Duration{
		"cache": 0,
		"db":    0,
	}

	// Тест кэша
	start := time.Now()
	for i := 0; i < 1000; i++ {
		s.cache.Get(orderUID)
	}
	cacheDuration := time.Since(start)
	results["cache"] = cacheDuration

	// Тест БД
	start = time.Now()
	for i := 0; i < 1000; i++ {
		_, err := s.getOrderFromDB(orderUID)
		if err != nil {
			// Если заказа нет в БД, пропускаем тест
			return results, fmt.Errorf("заказ не найден в БД: %s", orderUID)
		}
	}
	dbDuration := time.Since(start)
	results["db"] = dbDuration

	return results, nil
}

// PrintCacheContents выводит содержимое кэша (для отладки)
func (s *OrderService) PrintCacheContents() {
	fmt.Println("Содержимое кэша:")
	count := 0
	s.cache.Range(func(key, value interface{}) bool {
		if orderUID, ok := key.(string); ok {
			fmt.Printf("  %d. %s\n", count+1, orderUID)
			count++
		}
		return true
	})
	fmt.Printf("Всего заказов в кэше: %d\n", count)
}
