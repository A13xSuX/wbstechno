package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"order-service/internal/database"
	"testing"
	"time"
)

// простой mock сервиса, реализующий интерфейс service.OrderService
type MockOrderService struct {
	orders map[string]database.Order
}

func NewMockOrderService() *MockOrderService {
	return &MockOrderService{
		orders: map[string]database.Order{
			"found123": {
				OrderUID:    "found123",
				TrackNumber: "FOUND_TRACK",
				Delivery: database.Delivery{
					Name:  "Test User",
					Phone: "+1234567890",
				},
				Payment: database.Payment{
					Transaction: "test_txn",
					Amount:      100,
					Currency:    "USD",
				},
				Items: []database.Item{
					{
						Name:  "Test Item",
						Price: 100,
					},
				},
			},
		},
	}
}

func (m *MockOrderService) GetOrder(orderUID string) (database.Order, error) {
	order, exists := m.orders[orderUID]
	if !exists {
		return database.Order{}, fmt.Errorf("Заказ не найден!")
	}
	return order, nil
}

func (m *MockOrderService) ProcessOrder(message []byte) error {
	return nil
}

func (m *MockOrderService) ValidateOrder(order database.Order) error {
	return nil
}

func (m *MockOrderService) GetCacheSize() int {
	return len(m.orders)
}

func (m *MockOrderService) CheckDBConnection() error {
	return nil
}

func (m *MockOrderService) RunBenchmark(orderUID string) (map[string]time.Duration, error) {
	return map[string]time.Duration{"cache": time.Millisecond, "db": time.Second}, nil
}

func (m *MockOrderService) PrintCacheContents() {
	// пустая реализация для тестов
}

func TestOrderHandlerFound(t *testing.T) {
	service := NewMockOrderService()
	handler := orderHandler(service)

	// создаем тестовый запрос
	req := httptest.NewRequest("GET", "/order/found123", nil)
	w := httptest.NewRecorder()

	// вызываем хендлер
	handler(w, req)

	// проверяем ответ
	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус 200, получен %d", w.Code)
	}

	// проверяем содержимое ответа
	var response database.Order
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Ошибка декодирования JSON: %v", err)
	}

	if response.OrderUID != "found123" {
		t.Errorf("Ожидался OrderUID 'found123', получен '%s'", response.OrderUID)
	}
	if response.TrackNumber != "FOUND_TRACK" {
		t.Errorf("Ожидался TrackNumber 'FOUND_TRACK', получен '%s'", response.TrackNumber)
	}
}

func TestOrderHandlerNotFound(t *testing.T) {
	service := NewMockOrderService()
	handler := orderHandler(service)

	// Запрос несуществующего заказа
	req := httptest.NewRequest("GET", "/order/notfound999", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	// Должен вернуть 404
	if w.Code != http.StatusNotFound {
		t.Errorf("Ожидался статус 404, получен %d", w.Code)
	}

	// Проверяем сообщение об ошибке
	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Ошибка декодирования JSON: %v", err)
	}

	if response["error"] != "Order not found" {
		t.Errorf("Ожидалась ошибка 'Order not found', получено '%v'", response["error"])
	}
}

func TestCacheHandler(t *testing.T) {
	service := NewMockOrderService()
	handler := cacheHandler(service)

	req := httptest.NewRequest("GET", "/cache", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус 200, получен %d", w.Code)
	}

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Ошибка декодирования JSON: %v", err)
	}

	if response["cache_size"] != float64(1) {
		t.Errorf("Ожидался размер кэша 1, получен %v", response["cache_size"])
	}
}

func TestHealthHandler(t *testing.T) {
	service := NewMockOrderService()
	handler := healthHandler(service)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ожидался статус 200, получен %d", w.Code)
	}

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Ошибка декодирования JSON: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Ожидался статус 'healthy', получен '%v'", response["status"])
	}
	if response["cache_size"] != float64(1) {
		t.Errorf("Ожидался размер кэша 1, получен %v", response["cache_size"])
	}
}
