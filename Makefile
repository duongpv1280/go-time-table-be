.PHONY: tidy generate oapi-codegen wire build run air test lint \
        migrate-up-local migrate-down-local migrate-create \
        install-tools clean help

DB_PATH ?= gorm.db

help:
	@echo "Available targets:"
	@echo "  install-tools      - Install required Go tools (oapi-codegen, wire, air)"
	@echo "  tidy               - Run go mod tidy"
	@echo "  generate           - Run oapi-codegen + wire"
	@echo "  oapi-codegen       - Regenerate api.gen.go from api/root.yaml"
	@echo "  wire               - Regenerate wire_gen.go"
	@echo "  build              - Compile to bin/server"
	@echo "  run                - Generate + start server on :8080"
	@echo "  air                - Start server with hot reload (requires air)"
	@echo "  test               - Run all tests"
	@echo "  lint               - Run go vet"
	@echo "  migrate-up-local   - Apply all pending migrations"
	@echo "  migrate-down-local - Roll back all migrations"
	@echo "  migrate-create     - Create a new migration file pair"
	@echo "  clean              - Remove bin/, gorm.db, and generated files"

install-tools:
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/air-verse/air@latest

tidy:
	go mod tidy

oapi-codegen:
	PATH=$(PATH):$(HOME)/go/bin oapi-codegen -config oapi-codegen.yaml api/root.yaml

wire:
	PATH=$(PATH):$(HOME)/go/bin wire ./internal/infrastructure/di

generate: oapi-codegen wire

build: generate
	go build -o bin/server cmd/server/main.go

run: generate
	go run cmd/server/main.go

air:
	PATH=$(PATH):$(HOME)/go/bin air -c .air.toml

test:
	go test -v ./...

lint:
	go vet ./...

migrate-up-local:
	go run ./cmd/migration -cmd up

migrate-down-local:
	go run ./cmd/migration -cmd down

migrate-create:
	@read -p "Migration name: " name; \
	count=$$(ls cmd/migration/sqls/*.sql 2>/dev/null | wc -l | tr -d ' '); \
	num=$$(printf "%06d" $$(( count / 2 + 1 ))); \
	touch "cmd/migration/sqls/$${num}_$${name}.up.sql"; \
	touch "cmd/migration/sqls/$${num}_$${name}.down.sql"; \
	echo "Created cmd/migration/sqls/$${num}_$${name}.{up,down}.sql"

clean:
	rm -rf bin/
	rm -f gorm.db
	rm -f internal/delivery/http/openapi/api.gen.go
	rm -f internal/infrastructure/di/wire_gen.go
