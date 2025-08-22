package main

import (
	"log"
	"order-service/internal/cache"
	"order-service/internal/config"
	"order-service/internal/database"
	"order-service/internal/handler"
	"order-service/internal/kafka"
	"order-service/internal/service"

	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load("../../.env"); err != nil {
		log.Println("Не найден .env файл")
	}

	cfg := config.LoadConfig()

	db, err := database.ConnectDB(cfg.DB)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer db.Close()

	orderCache := cache.NewOrderCache()

	if err := cache.RestoreCacheFromDB(db, orderCache, 100); err != nil {
		log.Printf("Ошибка восстановления кэша: %v", err)
	}

	orderService := service.NewOrderService(db, orderCache)

	go handler.StartHTTPServer(orderService, cfg.HTTP.Port)

	kafka.StartKafkaConsumer(cfg.Kafka, orderService)
}
