.PHONY: tidy generate oapi-codegen wire build run test clean

# Help command to list targets
help:
	@echo "Available targets:"
	@echo "  tidy         - Run go mod tidy"
	@echo "  generate     - Generate code using oapi-codegen and wire"
	@echo "  oapi-codegen - Generate Echo server and models from OpenAPI spec"
	@echo "  wire         - Generate dependency injection bindings"
	@echo "  build        - Compile the project"
	@echo "  run          - Run the development server"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean generated files and databases"

install-tools:
	docker buildx build --progress=plain --no-cache -t $(BUSYBOX) -f $(MAKEFILE_DIR)docker/busybox/Dockerfile .

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

test:
	go test -v ./...

clean:
	rm -rf bin/
	rm -f gorm.db
	rm -f internal/delivery/http/api.gen.go
	rm -f internal/infrastructure/di/wire_gen.go
