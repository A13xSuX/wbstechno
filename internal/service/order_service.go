package service

import (
	"encoding/json"
	"fmt"
	"log"
	"order-service/internal/cache"
	"order-service/internal/database"
	"time"
)

// OrderServiceImpl реализация интерфейса OrderService
type OrderServiceImpl struct {
	repo      database.OrderRepository
	cache     cache.Cache
	validator *ValidatorService
}

// NewOrderService создает новый сервис заказов
func NewOrderService(repo database.OrderRepository, cache cache.Cache) *OrderServiceImpl {
	return &OrderServiceImpl{
		repo:      repo,
		cache:     cache,
		validator: NewValidatorService(),
	}
}

// ProcessOrder обрабатывает входящее сообщение с заказом
func (s *OrderServiceImpl) ProcessOrder(message []byte) error {
	var order database.Order

	if len(message) == 0 {
		return fmt.Errorf("пустое сообщение")
	}

	if err := json.Unmarshal(message, &order); err != nil {
		log.Printf("Ошибка парсинга JSON: %v\n", err)
		log.Printf("Содержимое сообщения: %s\n", string(message))
		return fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	if err := s.ValidateOrder(order); err != nil {
		log.Printf("Невалидный заказ: %v\n", err)
		log.Printf("Данные заказа: %+v\n", order)
		return fmt.Errorf("невалидный заказ: %v", err)
	}

	// Получаем соединение из репозитория
	db := s.repo.GetDB()
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			log.Printf("Транзакция откачена: %s\n", order.OrderUID)
		}
	}()

	// Сохраняем заказ через репозиторий
	if err := s.repo.SaveOrder(tx, order); err != nil {
		return fmt.Errorf("ошибка сохранения заказа: %v", err)
	}

	if err := s.repo.SaveDelivery(tx, order); err != nil {
		return fmt.Errorf("ошибка сохранения доставки: %v", err)
	}

	if err := s.repo.SavePayment(tx, order); err != nil {
		return fmt.Errorf("ошибка сохранения платежа: %v", err)
	}

	if err := s.repo.SaveItems(tx, order); err != nil {
		return fmt.Errorf("ошибка сохранения товаров: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %v", err)
	}

	// Сохраняем в кэш
	s.cache.Set(order)

	fmt.Printf("   Обработка заказа: %s\n", order.OrderUID)
	fmt.Printf("   Трек номер: %s\n", order.TrackNumber)
	fmt.Printf("   Клиент: %s (%s)\n", order.Delivery.Name, order.Delivery.Email)
	fmt.Printf("   Сумма заказа: %d %s\n", order.Payment.Amount, order.Payment.Currency)
	fmt.Printf("   Количество товаров: %d\n", len(order.Items))
	fmt.Printf("   Дата создания: %s\n", order.DateCreated.Format(time.RFC3339))
	fmt.Println("   --- Заказ сохранен в БД и кэш ---")

	return nil
}

func (s *OrderServiceImpl) ValidateOrder(order database.Order) error {
	return s.validator.ValidateOrder(order)
}

// GetOrder возвращает заказ по ID
func (s *OrderServiceImpl) GetOrder(orderUID string) (database.Order, error) {
	// Сначала проверяем в кэше
	if order, found := s.cache.Get(orderUID); found {
		return order, nil
	}

	// Если нет в кэше, ищем в БД через репозиторий
	order, err := s.repo.GetOrder(orderUID)
	if err != nil {
		return database.Order{}, err
	}

	// Сохраняем в кэш для будущих запросов
	s.cache.Set(order)
	return order, nil
}

// GetCacheSize возвращает размер кэша
func (s *OrderServiceImpl) GetCacheSize() int {
	return s.cache.Size()
}

// CheckDBConnection проверяет соединение с БД
func (s *OrderServiceImpl) CheckDBConnection() error {
	return s.repo.CheckConnection()
}

// RunBenchmark запускает бенчмарк-тест
func (s *OrderServiceImpl) RunBenchmark(orderUID string) (map[string]time.Duration, error) {
	results := map[string]time.Duration{
		"cache": 0,
		"db":    0,
	}

	// Тест кэша
	start := time.Now()
	for i := 0; i < 1000; i++ {
		s.cache.Get(orderUID)
	}
	results["cache"] = time.Since(start)

	// Тест БД
	start = time.Now()
	for i := 0; i < 1000; i++ {
		_, err := s.repo.GetOrder(orderUID)
		if err != nil {
			return results, fmt.Errorf("заказ не найден в БД: %s", orderUID)
		}
	}
	results["db"] = time.Since(start)

	return results, nil
}

// PrintCacheContents выводит содержимое кэша
func (s *OrderServiceImpl) PrintCacheContents() {
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
