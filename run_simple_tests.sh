#!/bin/bash

echo "Запуск простых тестов..."

echo "1. Тесты кэша:"
go test ./internal/cache/ -run TestCache -v

echo ""
echo "2. Тесты сервиса:"
go test ./internal/service/ -run TestOrder -v

echo ""
echo "3. Тесты хендлеров:"
go test ./internal/handler/ -run TestOrder -v

echo ""
echo "Все простые тесты завершены!"