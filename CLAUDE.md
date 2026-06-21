# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make generate      # Run oapi-codegen + wire (required before build/run after any spec/DI change)
make oapi-codegen  # Regenerate api.gen.go from api/root.yaml only
make wire          # Regenerate wire_gen.go only
make run           # Generate + start Echo server on :8080 (uses/creates gorm.db SQLite file)
make build         # Generate + compile to bin/server
make test          # Run all tests (go test -v ./...)
make clean         # Remove bin/, gorm.db, and all generated files
```

Run a single test package:
```bash
go test -v ./internal/domain/user/...
go test -v ./internal/usecase/user/...
```

`oapi-codegen` and `wire` binaries must be in `~/go/bin`. Install them once via:
```bash
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
go install github.com/google/wire/cmd/wire@latest
```

## Architecture

Clean Architecture + DDD with strict dependency direction: Delivery → Usecase → Domain ← Infrastructure.

**Domain** (`internal/domain/user/`) — zero external dependencies. `User` is the aggregate root with private fields; all fields are accessed only through getters. Value objects (`Email`, `Name`, `ID`) validate on construction and return typed domain errors (`ErrInvalidEmail`, `ErrEmptyName`, `ErrInvalidID`). The `UserRepository` interface is defined here (the port), not in infrastructure.

**Usecase** (`internal/usecase/user/`) — orchestrates domain objects. Accepts/returns plain DTOs (no domain types leak out). Converts string inputs to domain value objects before calling the repository.

**Delivery** (`internal/delivery/http/`) — Echo v4 HTTP layer, split into two levels:
- `internal/delivery/http/` — two files: `api.gen.go` (generated; never edit manually) defines `ServerInterface`, request parameter/body types, and `RegisterHandlers`; `types.go` (hand-maintained) defines response types (`User`, `ErrorResponse`, `Pagination`, `ListUsersResponse`) that mirror the OpenAPI schemas — update it whenever a response schema changes.
- `internal/delivery/http/handlers/` — concrete handler structs implementing `ServerInterface`. Each resource gets its own file (e.g. `user.go`). Handlers translate HTTP concerns (bind, status codes) to usecase calls and map domain errors to HTTP responses via `errors.Is`.

**Infrastructure** (`internal/infrastructure/db/`) — GORM/SQLite implementation of `UserRepository`. Uses a separate `UserModel` struct (with GORM tags) that is mapped to/from domain `User` to keep database concerns out of the domain.

**DI** (`internal/infrastructure/di/`) — Google Wire. `wire.go` declares providers; `wire_gen.go` is **generated** by `wire ./internal/infrastructure/di`. When adding a new component, add its provider to `wire.go` and re-run `make generate`.

## Coding conventions

| Element | Style | Example |
|---|---|---|
| Types, structs | PascalCase | `type UserModel struct {}` |
| Interfaces | PascalCase prefixed with `I` | `type IUserUseCase interface {}` |
| File names, directory names | kebab-case | `user-repository.go`, `use-case/` |
| Constants | SCREAMING_SNAKE_CASE | `const MAX_RETRY_COUNT = 3` |
| Variables, function parameters | camelCase | `userID`, `createdAt` |

## API spec structure

The OpenAPI spec is split into multiple files under `api/` and assembled via `$ref`:

```
api/
├── root.yaml               # Entry point — oapi-codegen reads this
├── paths/
│   ├── users.yaml          # GET /users, POST /users
│   └── users-by-id.yaml   # GET /users/{id}, DELETE /users/{id}
├── requestBodies/
│   └── create-user.yaml   # Reusable request body definitions
└── schemas/
    ├── user.yaml           # User response object
    ├── list-users-item.yaml
    ├── create-user-request.yaml
    ├── pagination.yaml
    └── error-response.yaml
```

**Rules:**
- `root.yaml` only declares `openapi`, `info`, and `paths` — each path value is a `$ref` to a file in `paths/`.
- `paths/*.yaml` files are Path Item Objects (no wrapping key); they contain the HTTP methods (`get`, `post`, etc.) directly.
- `requestBodies/*.yaml` files are Request Body Objects — wrap a schema `$ref` with `required` and `content`.
- `schemas/*.yaml` files are Schema Objects — no wrapping key, just `type`, `properties`, etc.
- Cross-references use relative paths: `../schemas/foo.yaml`, `../requestBodies/bar.yaml`.
- `oapi-codegen` reads `api/root.yaml` and resolves all `$ref`s automatically; run `make oapi-codegen` after any spec change.

## Key conventions

- Domain fields are unexported; use `NewUser` to construct and `RestoreUser` to reconstruct from storage.
- `NewEmail`/`NewName`/`ParseID` are the only constructors for value objects — always use them, never set `.value` directly.
- Domain errors are sentinel `errors.New` values in the domain package; handlers use `errors.Is` to map them to HTTP codes.
- Generated files (`api.gen.go`, `wire_gen.go`) are committed to the repo and must be regenerated after any change to the API spec or DI providers.
- `types.go` is hand-maintained and must be kept in sync with `api/schemas/*.yaml`; oapi-codegen does not generate response types from external `$ref`s.
- SQLite database file (`gorm.db`) is local only; `make clean` removes it.
