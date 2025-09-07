package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"order-service/internal/service"
	"path/filepath"
	"time"
)

func StartHTTPServer(ctx context.Context, orderService *service.OrderService, port string) {
	server := &http.Server{
		Addr:    port,
		Handler: nil,
	}

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("../../static"))))
	http.HandleFunc("/order/", enableCORS(orderHandler(orderService)))
	http.HandleFunc("/cache", enableCORS(cacheHandler(orderService)))
	http.HandleFunc("/health", enableCORS(healthHandler(orderService)))
	http.HandleFunc("/", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "../../static/index.html")
			return
		}
		http.NotFound(w, r)
	}))

	go func() {
		<-ctx.Done()
		log.Println("Останавливаем HTTP сервер")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Ошибка остановки HTTP сервера: %v", err)
		}
	}()

	log.Printf("   HTTP сервер запущен на %s", port)
	log.Printf("   http://localhost%s/ - веб-интерфейс", port)
	log.Printf("   http://localhost%s/order/{id} - получить заказ", port)
	log.Printf("   http://localhost%s/cache - просмотр кэша", port)
	log.Printf("   http://localhost%s/health - проверка здоровья", port)
	log.Printf("   http://localhost%s/benchmark/{id} - тест производительности", port)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Ошибка HTTP сервера: %v", err)
	}
}

func orderHandler(orderService *service.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		orderUID := r.URL.Path[len("/order/"):]
		if orderUID == "" {
			http.Error(w, "Order ID is required", http.StatusBadRequest)
			return
		}

		fmt.Printf("Поиск заказа: %s\n", orderUID)

		order, err := orderService.GetOrder(orderUID)
		if err != nil {
			fmt.Printf("Заказ не найден: %s\n", orderUID)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":     "Order not found",
				"order_uid": orderUID,
				"message":   "Заказ с указанным ID не существует",
			})
			return
		}

		fmt.Printf("Найден заказ: %s\n", orderUID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(order)
	}
}

func cacheHandler(orderService *service.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		cacheInfo := map[string]interface{}{
			"cache_size":  orderService.GetCacheSize(),
			"server_time": time.Now().Format(time.RFC3339),
			"message":     "Информация о кэше доступна через сервисный слой",
		}

		json.NewEncoder(w).Encode(cacheInfo)
	}
}

func healthHandler(orderService *service.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		dbStatus := "healthy"
		if err := orderService.CheckDBConnection(); err != nil {
			dbStatus = "unhealthy"
			log.Printf("Проверка здоровья БД: %v", err)
		}

		healthStatus := map[string]interface{}{
			"status":      "healthy",
			"timestamp":   time.Now().Format(time.RFC3339),
			"cache_size":  orderService.GetCacheSize(),
			"db_status":   dbStatus,
			"service":     "order-service",
			"version":     "1.0.0",
			"environment": "development",
		}

		json.NewEncoder(w).Encode(healthStatus)
	}
}

func staticHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Отдаем index.html для корневого пути
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "static/index.html")
			return
		}

		// Проверяем, что запрашивается статический файл
		ext := filepath.Ext(r.URL.Path)
		if ext == ".html" || ext == ".css" || ext == ".js" || ext == ".png" || ext == ".jpg" {
			http.ServeFile(w, r, "static"+r.URL.Path)
			return
		}

		// Для всех остальных путей отдаем 404
		http.NotFound(w, r)
	}
}

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}
