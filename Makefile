APP_NAME=posts-comments-service

.PHONY: run test fmt tidy docker-build docker-up docker-down

build:
	mkdir -p bin
	go build -o bin/api ./cmd/api

run: build
	./bin/api

test:
	go test ./...

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