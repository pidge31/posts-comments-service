APP_NAME=posts-comments-service

.PHONY: run test test-race fmt tidy generate docker-build docker-up docker-down

run:
	go run ./cmd/api

test:
	go test ./...

test-race:
	go test ./... -race

fmt:
	go fmt ./...

tidy:
	go mod tidy

generate:
	go tool gqlgen generate

docker-build:
	docker build -t $(APP_NAME) .

docker-up:
	docker compose up --build

docker-down:
	docker compose down -v