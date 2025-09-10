# Order Service

Микросервис для обработки и управления заказами с использованием **Go**, **PostgreSQL**, **Apache Kafka** и **in-memory кэша**.

---

## 🚀 Возможности

- 📥 Прием заказов через **Apache Kafka**
- ⚡ In-memory кэширование заказов (`sync.Map`)
- 🌐 **RESTful API** для поиска заказов
- 🖥️ Веб-интерфейс для поиска заказов
- 📊 Бенчмаркинг производительности кэша vs БД
- 🔄 Автоматическое восстановление кэша из БД
- 🌍 Поддержка **CORS**

---

## 🛠 Предварительные требования

- **Docker** и **Docker Compose** — для запуска Kafka
- **Go 1.21+** — для запуска сервиса
- **PostgreSQL** — база данных

---

## ⚡ Быстрый старт

### 1. Запуск Kafka инфраструктуры

```bash
# Запустите Zookeeper, Kafka и Kafka UI
docker-compose up -d
```
Проверьте, что сервисы запущены:

Kafka UI: http://localhost:8082

Kafka Broker: localhost:9092

Zookeeper: localhost:2181

### 2. Перейдите в папку проекта
```cd order-service```

### 3. Установите зависимости
```go mod download```

### 4. Запустите сервис
```go run cmd/orderservice/main.go```

# Руководство по тестированию Order Service

## 🎯 Обзор тестирования

Проект включает unit-тесты для всех основных компонентов:
- Кэш заказов (`internal/cache/`)
- Сервисный слой (`internal/service/`) 
- HTTP хендлеры (`internal/handler/`)

## 🧪 Типы тестов

### 1. Тесты кэша (`cache_simple_test.go`)
- **TestCacheBasicOperations**: базовые операции (добавление, получение, удаление)
- **TestCacheSize**: проверка ограничения размера кэша
- **TestCacheExpiration**: проверка TTL (времени жизни записей)

### 2. Тесты хендлеров (`handler_simple_test.go`)
- **TestOrderHandlerFound**: успешный поиск существующего заказа
- **TestOrderHandlerNotFound**: обработка несуществующего заказа (404)
- **TestCacheHandler**: получение информации о размере кэша
- **TestHealthHandler**: проверка health check endpoint

### 3. Тесты сервиса (`service_simple_test.go`)
- **TestValidation**: валидация заказов
- **TestJSONParsing**: парсинг JSON сообщений
- **TestEmptyMessage**: обработка пустых сообщений
- **TestGetOrderFromCache**: получение заказов из кэша
- **TestValidatorService**: тестирование валидатора заказов

## 🚀 Запуск тестов

### Вариант 1: Все тесты
```bash
go test ./internal/... -v

### Вариант 2: По компонентам
```bash
# Только кэш
go test ./internal/cache/ -v

# Только хендлеры  
go test ./internal/handler/ -v

# Только сервис
go test ./internal/service/ -v```

Вариант 3: Конкретные тесты
```bash
# Только тесты валидации
go test ./internal/service/ -run TestValidation -v

# Только тесты кэша
go test ./internal/cache/ -run TestCache -v```

Вариант 4: Через скрипт
```bash
chmod +x run_simple_tests.sh
./run_simple_tests.sh```
