package cache

import (
	"order-service/internal/database"
	"testing"
	"time"
)

func TestCacheBasicOperations(t *testing.T) {
	// создаем кэш на 5 минут
	cache := NewOrderCache(10, 5*time.Minute)
	defer cache.Stop()

	// создаем тестовый заказ
	testOrder := database.Order{
		OrderUID:    "test123",
		TrackNumber: "TRACK001",
	}

	// Тест 1: Добавляем заказ в кэш
	cache.Set(testOrder)

	// Тест 2: Получаем заказ из кэша
	order, found := cache.Get("test123")
	if !found {
		t.Error("Заказ не найден в кэше")
	}
	if order.OrderUID != "test123" {
		t.Errorf("Ожидался OrderUID 'test123', получен '%s'", order.OrderUID)
	}

	// Тест 3: Удаляем заказ из кэша
	cache.Delete("test123")
	_, found = cache.Get("test123")
	if found {
		t.Error("Заказ все еще в кэше после удаления")
	}
}

func TestCacheSize(t *testing.T) {
	cache := NewOrderCache(3, 10*time.Minute)
	defer cache.Stop()

	// добавляем 3 заказа
	for i := 1; i <= 3; i++ {
		order := database.Order{OrderUID: string(rune('a' + i))}
		cache.Set(order)
	}

	// проверяем размер кэша
	size := cache.Size()
	if size != 3 {
		t.Errorf("Ожидался размер кэша 3, получен %d", size)
	}
}

func TestCacheExpiration(t *testing.T) {
	// кэш с очень коротким временем жизни (100ms)
	cache := NewOrderCache(10, 100*time.Millisecond)
	defer cache.Stop()

	order := database.Order{OrderUID: "test456"}
	cache.Set(order)

	// сразу после добавления должен быть в кэше
	_, found := cache.Get("test456")
	if !found {
		t.Error("Заказ не найден сразу после добавления")
	}

	// ждем пока истечет TTL
	time.Sleep(150 * time.Millisecond)

	// после истечения TTL не должен быть в кэше
	_, found = cache.Get("test456")
	if found {
		t.Error("Заказ все еще в кэше после истечения TTL")
	}
}
