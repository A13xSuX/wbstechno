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

