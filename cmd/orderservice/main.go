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

	// запуск миграций БД
	log.Println("Запуск миграций базы данных...")
	if err := database.RunMigrations(cfg.DB); err != nil {
		log.Fatalf("Ошибка применения миграций: %v", err)
	}
	log.Println("Миграции успешно применены")

	// cоздаем репозиторий
	orderRepo := database.NewOrderRepository(db.DB)

	// cоздаем кэш
	orderCache := cache.NewOrderCache(cfg.Cache.MaxSize, cfg.Cache.TTL)
	defer orderCache.Stop()

	// восстанавливаем кэш из БД
	if err := cache.RestoreCacheFromDB(db.DB, orderCache, cfg.Cache.RestoreLimit); err != nil {
		log.Printf("Ошибка восстановления кэша: %v", err)
	}

	// cоздаем сервис
	orderService := service.NewOrderService(orderRepo, orderCache)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// запускаем HTTP сервер
	wg.Add(1)
	go func() {
		defer wg.Done()
		handler.StartHTTPServer(ctx, orderService, cfg.HTTP.Port)
	}()

	// запускаем Kafka
	wg.Add(1)
	go func() {
		defer wg.Done()
		kafka.StartKafkaConsumer(ctx, cfg.Kafka, orderService)
	}()

	log.Println("Для остановки нажмите Ctrl+C")

	// ожидание сигналов завершения
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
