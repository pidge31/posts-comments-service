APP_NAME=posts-comments-service

.PHONY: run test test-race fmt tidy generate docker-build docker-up docker-down migrate-up migrate-down

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

migrate-up:
	docker compose exec -T postgres psql -U postgres -d posts_comments < migrations/001_create_posts.up.sql
	docker compose exec -T postgres psql -U postgres -d posts_comments < migrations/002_create_comments.up.sql

migrate-down:
	docker compose exec -T postgres psql -U postgres -d posts_comments < migrations/002_create_comments.down.sql
	docker compose exec -T postgres psql -U postgres -d posts_comments < migrations/001_create_posts.down.sql