package kafka

import (
	"context"
	"log"
	"order-service/internal/config"
	"order-service/internal/service"

	"github.com/segmentio/kafka-go"
)

func StartKafkaConsumer(cfg config.KafkaConfig, orderService *service.OrderService) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	log.Printf("Подписались на топик: %s", cfg.Topic)

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Ошибка чтения сообщения: %v", err)
			continue
		}

		if err := orderService.ProcessOrder(msg.Value); err != nil {
			log.Printf("Ошибка обработки сообщения: %v", err)
		}
	}
}
