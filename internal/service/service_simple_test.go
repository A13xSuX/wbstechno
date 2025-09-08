package service

import (
	"encoding/json"
	"order-service/internal/database"
	"testing"
	"time"
)

// Простой тест валидации заказа
func TestValidation(t *testing.T) {
	service := &OrderServiceImpl{}

	// Тест 1: Валидный заказ
	validOrder := database.Order{
		OrderUID:    "test123",
		TrackNumber: "TRACK001",
		Entry:       "WBIL",
		Delivery: database.Delivery{
			Name:  "Test User",
			Phone: "+1234567890",
		},
		Payment: database.Payment{
			Transaction: "txn123",
			Amount:      100,
			Currency:    "USD",
		},
		Items: []database.Item{
			{
				Name:  "Test Product",
				Price: 100,
			},
		},
	}

	err := service.ValidateOrder(validOrder)
	if err != nil {
		t.Errorf("Валидный заказ не прошел проверку: %v", err)
	}

	// Тест 2: Невалидный заказ (без товаров)
	invalidOrder := validOrder
	invalidOrder.Items = nil

	err = service.ValidateOrder(invalidOrder)
	if err == nil {
		t.Error("Невалидный заказ прошел проверку")
	}
}

// Тест только парсинга JSON без вызова ProcessOrder
func TestJSONParsing(t *testing.T) {
	// Тест валидного JSON
	validJSON := `{
		"order_uid": "test123",
		"track_number": "TRACK001",
		"entry": "WBIL",
		"delivery": {
			"name": "Test User",
			"phone": "+1234567890"
		},
		"payment": {
			"transaction": "txn123",
			"amount": 100,
			"currency": "USD"
		},
		"items": [
			{
				"name": "Test Product",
				"price": 100
			}
		]
	}`

	// Проверяем только парсинг JSON
	var order database.Order
	err := json.Unmarshal([]byte(validJSON), &order)
	if err != nil {
		t.Errorf("Ошибка парсинга валидного JSON: %v", err)
	}

	// Проверяем что данные распарсились правильно
	if order.OrderUID != "test123" {
		t.Errorf("Ожидался OrderUID 'test123', получен '%s'", order.OrderUID)
	}
	if order.TrackNumber != "TRACK001" {
		t.Errorf("Ожидался TrackNumber 'TRACK001', получен '%s'", order.TrackNumber)
	}

	// Тест невалидного JSON
	invalidJSON := `{invalid json`
	err = json.Unmarshal([]byte(invalidJSON), &order)
	if err == nil {
		t.Error("Ожидалась ошибка парсинга невалидного JSON")
	}
}

// Тест пустого сообщения
func TestEmptyMessage(t *testing.T) {
	service := &OrderServiceImpl{}

	err := service.ProcessOrder([]byte{})
	if err == nil {
		t.Error("Ожидалась ошибка для пустого сообщения")
	} else if err.Error() != "пустое сообщение" {
		t.Errorf("Ожидалась ошибка 'пустое сообщение', получено: %v", err)
	}
}

// Тест получения заказа из кэша (упрощенный)
func TestGetOrderFromCache(t *testing.T) {
	// Создаем простой mock кэша
	cache := &SimpleCacheMock{}

	// Создаем сервис только с кэшем (репозиторий nil)
	service := &OrderServiceImpl{
		cache: cache,
	}

	// Добавляем заказ в кэш
	testOrder := database.Order{OrderUID: "cache123", TrackNumber: "FROM_CACHE"}
	cache.Set(testOrder)

	// Должен получить из кэша
	order, err := service.GetOrder("cache123")
	if err != nil {
		t.Errorf("Ошибка при получении заказа: %v", err)
	}
	if order.TrackNumber != "FROM_CACHE" {
		t.Errorf("Ожидался трек-номер 'FROM_CACHE', получен '%s'", order.TrackNumber)
	}
}

// Простой mock для кэша
type SimpleCacheMock struct {
	storage map[string]database.Order
}

func (m *SimpleCacheMock) Get(orderUID string) (database.Order, bool) {
	order, exists := m.storage[orderUID]
	return order, exists
}

func (m *SimpleCacheMock) Set(order database.Order) {
	if m.storage == nil {
		m.storage = make(map[string]database.Order)
	}
	m.storage[order.OrderUID] = order
}

func (m *SimpleCacheMock) Delete(orderUID string) {}

func (m *SimpleCacheMock) Size() int {
	if m.storage == nil {
		return 0
	}
	return len(m.storage)
}

func (m *SimpleCacheMock) Cleanup(ttl time.Duration) {}

func (m *SimpleCacheMock) Stop() {}

// ИСПРАВЛЕННАЯ СИГНАТУРА: правильный возвращаемый тип
func (m *SimpleCacheMock) Range(f func(key, value interface{}) bool) {
	for k, v := range m.storage {
		// Вызываем функцию и проверяем нужно ли продолжать
		if !f(k, v) {
			break
		}
	}
}

// Добавляем тест для validator
func TestValidatorService(t *testing.T) {
	validator := NewValidatorService()

	// Тест валидного заказа
	validOrder := database.Order{
		OrderUID:    "test123",
		TrackNumber: "TRACK001",
		Entry:       "WBIL",
		Delivery: database.Delivery{
			Name:    "Test User",
			Phone:   "+79161234567",
			Zip:     "123456",
			City:    "Moscow",
			Address: "Street 123",
			Region:  "Moscow",
			Email:   "test@example.com",
		},
		Payment: database.Payment{
			Transaction:  "b563feb7-b2b8-4b6a-9f5d-123456789abc", // Правильный UUID формат
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1817,
			PaymentDt:    1637907727,
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317,
			CustomFee:    0,
		},
		Items: []database.Item{
			{
				ChrtID:      9934930,
				TrackNumber: "WBILMTESTTRACK",
				Price:       453,
				Rid:         "ab4219087a764ae0btest123", // alphanum
				Name:        "Mascaras",
				Sale:        30,
				Size:        "0",
				TotalPrice:  317,
				NmID:        2389212,
				Brand:       "Vivienne Sabo",
				Status:      202,
			},
		},
		Locale:            "en",
		CustomerID:        "test-customer123", // alphanum с подчеркиванием
		DeliveryService:   "meest",
		Shardkey:          "9",
		SmID:              99,
		DateCreated:       time.Now(),
		OofShard:          "1",
		InternalSignature: "",
	}

	err := validator.ValidateOrder(validOrder)
	if err != nil {
		t.Errorf("Валидный заказ не прошел проверку: %v", err)
	}

	// Тест невалидного заказа
	invalidOrder := validOrder
	invalidOrder.Delivery.Email = "invalid-email"

	err = validator.ValidateOrder(invalidOrder)
	if err == nil {
		t.Error("Невалидный заказ прошел проверку")
	} else {
		t.Logf("Ожидаемая ошибка валидации: %v", err)
	}
}
