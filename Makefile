.PHONY: test build run migrate seed
test:
	go test ./...
build:
	go build ./cmd/api
run:
	go run ./cmd/api
migrate:
	go run ./cmd/migrate
seed:
	go run ./cmd/seed
