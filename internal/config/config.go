package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DB    DatabaseConfig
	Kafka KafkaConfig
	HTTP  HTTPConfig
	Cache CacheConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type KafkaConfig struct {
	Brokers []string
	Topic   string
	GroupID string
}

type HTTPConfig struct {
	Port string
}
type CacheConfig struct {
	MaxSize      int
	RestoreLimit int
	TTL          time.Duration
}

func LoadConfig() Config {
	return Config{
		DB: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5433"),
			User:     getEnv("DB_USER", ""),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", ""),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			Topic:   getEnv("KAFKA_TOPIC", "orders"),
			GroupID: getEnv("KAFKA_GROUP_ID", "order-service-group"),
		},
		HTTP: HTTPConfig{
			Port: getEnv("HTTP_PORT", ":8080"),
		},
		Cache: CacheConfig{
			MaxSize:      getEnvAsInt("CACHE_MAX_SIZE", 100),
			RestoreLimit: getEnvAsInt("CACHE_RESTORE_LIMIT", 100),
			TTL:          getEnvAsDuration("CACHE_TTL", 60*time.Minute),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		// Парсим из строки (например:"60m")
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		// Пробуем прочитать как число минут (для обратной совместимости)
		if minutes, err := strconv.Atoi(value); err == nil {
			return time.Duration(minutes) * time.Minute
		}
	}
	return defaultValue
}
