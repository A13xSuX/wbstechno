package database

import (
	"context"
	"database/sql"
	"fmt"
	"order-service/internal/config"
	"time"

	_ "github.com/lib/pq"
)

// DB обертка с реализацией интерфейса
type DB struct {
	*sql.DB
}

// реализация интерфейса Database
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

// реализация закрытия Database
func (db *DB) CloseWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		db.SetMaxIdleConns(0)
		db.SetMaxOpenConns(0)
		done <- db.DB.Close()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("таймаут закрытия БД")
	}
}

type OrderRepositoryImpl struct {
	db *sql.DB
}

// создает новую реализацию репозитория
func NewOrderRepository(db *sql.DB) OrderRepository {
	return &OrderRepositoryImpl{db: db}
}

// возвращает соединение с БД
func (r *OrderRepositoryImpl) GetDB() *sql.DB {
	return r.db
}

// сохраняет заказ в БД
func (r *OrderRepositoryImpl) SaveOrder(tx *sql.Tx, order Order) error {
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

// сохраняет данные доставки
func (r *OrderRepositoryImpl) SaveDelivery(tx *sql.Tx, order Order) error {
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

// сохраняет данные платежа
func (r *OrderRepositoryImpl) SavePayment(tx *sql.Tx, order Order) error {
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

// сохраняет товары заказа
func (r *OrderRepositoryImpl) SaveItems(tx *sql.Tx, order Order) error {
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

// получает заказ из БД
func (r *OrderRepositoryImpl) GetOrder(orderUID string) (Order, error) {
	var order Order

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

	err := r.db.QueryRow(query, orderUID).Scan(
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
		return Order{}, err
	}

	if err := r.LoadOrderItems(&order); err != nil {
		return Order{}, fmt.Errorf("ошибка загрузки товаров: %v", err)
	}

	return order, nil
}

// загружает товары для заказа
func (r *OrderRepositoryImpl) LoadOrderItems(order *Order) error {
	query := `
		SELECT chrt_id, track_number, price, rid, name, 
			   sale, size, total_price, nm_id, brand, status
		FROM items 
		WHERE order_uid = $1
	`

	rows, err := r.db.Query(query, order.OrderUID)
	if err != nil {
		return fmt.Errorf("ошибка запроса товаров: %v", err)
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
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

func (r *OrderRepositoryImpl) CheckConnection() error {
	return r.db.Ping()
}
