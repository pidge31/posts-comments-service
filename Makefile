APP_NAME=posts-comments-service

.PHONY: run test test-race fmt tidy docker-build docker-up docker-down

build:
	mkdir -p bin
	go build -o bin/api ./cmd/api

run: build
	./bin/api

test:
	go test ./...

test-race:
	go test ./... -race

fmt:
	go fmt ./...

tidy:
	go mod tidy

docker-build:
	docker build -t $(APP_NAME) .

docker-up:
	docker compose up --build

docker-down:
	docker compose down -v