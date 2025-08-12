.PHONY: run build test test-unit test-integration clean

# Имя бинарного файла
BINARY_NAME=gotickets

# Запуск приложения
run:
	go run cmd/gotickets/main.go

# Сборка приложения
build:
	go build -o $(BINARY_NAME) cmd/gotickets/main.go

# Запуск всех тестов
test:
	go test -v ./test/...

# Запуск только unit тестов
test-unit:
	go test -v ./test/unit/...

# Запуск только integration тестов
test-integration:
	go test -v ./test/integration/...

# Очистка собранных файлов
clean:
	rm -f $(BINARY_NAME)
	rm -rf dist/

# Установка зависимостей
deps:
	go mod download
	go mod tidy

# Форматирование кода
fmt:
	go fmt ./...

# Проверка кода
vet:
	go vet ./...

# Полная проверка (форматирование, vet, тесты)
check: fmt vet test