BINARY_NAME=api
MAIN_PKG=./cmd/api

.PHONY: build run tidy test

tidy:
	go mod tidy

build: tidy
	go build -o bin/$(BINARY_NAME) $(MAIN_PKG)

run: tidy
	go run $(MAIN_PKG)

test:
	go test ./...
