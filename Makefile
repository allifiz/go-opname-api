DATABASE_URL ?= postgres://opname:opname@localhost:5432/opname?sslmode=disable

.PHONY: dev run db-up db-down migrate-up migrate-down sqlc tidy

dev: db-up migrate-up sqlc
	go run github.com/air-verse/air@latest

run:
	go run ./cmd/api

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

migrate-up:
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations postgres "$(DATABASE_URL)" down

sqlc:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate

tidy:
	go mod tidy
