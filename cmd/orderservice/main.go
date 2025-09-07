package main

import (
	"context"
	"log"
	"order-service/internal/cache"
	"order-service/internal/config"
	"order-service/internal/database"
	"order-service/internal/handler"
	"order-service/internal/kafka"
	"order-service/internal/service"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("Не найден .env файл")
	}

	cfg := config.LoadConfig()

	// Подключаемся к БД
	db, err := database.ConnectDB(cfg.DB)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer func() {
		log.Println("Закрываем соединение с БД")
		if err := db.CloseWithTimeout(10 * time.Second); err != nil {
			log.Printf("Ошибка закрытия БД: %v", err)
		} else {
			log.Println("Соединение с БД закрыто")
		}
	}()

	// Создаем репозиторий
	orderRepo := database.NewOrderRepository(db.DB)

	// Создаем кэш
	orderCache := cache.NewOrderCache(cfg.Cache.MaxSize, cfg.Cache.TTL)
	defer orderCache.Stop()

	// Восстанавливаем кэш из БД
	if err := cache.RestoreCacheFromDB(db.DB, orderCache, cfg.Cache.RestoreLimit); err != nil {
		log.Printf("Ошибка восстановления кэша: %v", err)
	}

	// Создаем сервис
	orderService := service.NewOrderService(orderRepo, orderCache)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Запускаем HTTP сервер
	wg.Add(1)
	go func() {
		defer wg.Done()
		handler.StartHTTPServer(ctx, orderService, cfg.HTTP.Port)
	}()

	// Запускаем Kafka consumer
	wg.Add(1)
	go func() {
		defer wg.Done()
		kafka.StartKafkaConsumer(ctx, cfg.Kafka, orderService)
	}()

	log.Println("Для остановки нажмите Ctrl+C")

	// Ожидание сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	sig := <-sigChan
	log.Printf("Получен сигнал: %v", sig)

	cancel()

	wg.Wait()
	log.Println("HTTP сервер и Kafka consumer остановлены")

	log.Println("Все компоненты остановлены")
	time.Sleep(100 * time.Millisecond)
}
