# Order Service

Микросервис для обработки и управления заказами с использованием Go, PostgreSQL, Apache Kafka и in-memory кэша.

## 🚀 Возможности

- ✅ Прием заказов через Apache Kafka
- ✅ In-memory кэширование заказов (sync.Map)
- ✅ RESTful API для поиска заказов
- ✅ Веб-интерфейс для поиска
- ✅ Бенчмаркинг производительности кэша vs БД
- ✅ Поддержка CORS
- ✅ Автоматическое восстановление кэша из БД

## 📋 Предварительные требования

- **Docker** и **Docker Compose** - для запуска Kafka
- **Go 1.21+** - для запуска сервиса
- **PostgreSQL** - база данных

## 🚀 Быстрый старт

### 1. Запуск Kafka инфраструктуры

```bash

# Запустите ZooKeeper, Kafka и Kafka UI
docker-compose up -d

Проверьте что сервисы запущены:

Kafka UI: http://localhost:8082

Kafka Broker: localhost:9092

ZooKeeper: localhost:2181

# Перейдите в папку проекта
cd order-service

# Установите зависимости
go mod download

# Запустите сервис
go run cmd/orderservice/main.go