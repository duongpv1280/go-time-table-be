# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make generate      # Run oapi-codegen + wire (required before build/run after any spec/DI change)
make oapi-codegen  # Regenerate internal/delivery/http/openapi/api.gen.go from api/root.yaml only
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
- `internal/delivery/http/openapi/` — **generated** package (`package openapi`). `api.gen.go` is produced by `make oapi-codegen` and defines `ServerInterface`, all request/response types, and `RegisterHandlers`. Never edit manually. `ptr.go` is a small hand-written helper (the `Ptr[T]` generic) that is NOT overwritten by generation.
- `internal/delivery/http/handlers/` — concrete handler structs implementing `ServerInterface`. Each resource gets its own file (`user.go`, `auth.go`, `class.go`). `combined.go` embeds all handlers into `CombinedHandler` which satisfies the full `ServerInterface`. Handlers translate HTTP concerns (bind, status codes) to usecase calls and map domain errors to HTTP responses via `errors.Is`.
- `internal/delivery/http/middleware/` — Echo middleware (JWT auth, permission extraction).

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

The OpenAPI spec lives under `api/`. `root.yaml` is the single source of truth for oapi-codegen — all component schemas are defined **inline** there with `#/components/schemas/X` cross-references so oapi-codegen can emit named Go types.

```
api/
├── root.yaml               # Entry point — inline component schemas + path refs
├── paths/
│   ├── users.yaml          # GET /users, POST /users
│   ├── users-by-id.yaml   # GET /users/{id}, DELETE /users/{id}
│   ├── classes.yaml        # GET /classes (JWT-protected)
│   ├── classes-by-id.yaml # GET /classes/{classId} (JWT-protected)
│   └── auth.yaml           # POST /auth/google
├── requestBodies/
│   └── create-user.yaml   # Reusable request body definitions
└── schemas/
    ├── user.yaml           # Schema docs (not read by oapi-codegen directly)
    ├── class.yaml
    ├── error-response.yaml
    ├── pagination.yaml
    └── ...
```

**Rules:**
- `root.yaml` declares `openapi`, `info`, `paths` (each a `$ref` to `paths/`), and **inline** `components/schemas` — do NOT use `$ref` to external files in the components section.
- `paths/*.yaml` files reference response schemas via `#/components/schemas/X` (root document reference), not via relative file paths. This is what makes oapi-codegen emit named types.
- `requestBodies/*.yaml` files may still use `$ref: '../schemas/X.yaml'` for request body schemas — these generate inline request-body types (e.g., `CreateUserJSONBody`) which is fine.
- `schemas/*.yaml` files are kept for documentation tools (Swagger UI, ReDoc) but are NOT read directly by oapi-codegen.
- Run `make oapi-codegen` after any spec change; run `make wire` after any DI provider change.

## Key conventions

- Domain fields are unexported; use `NewUser` to construct and `RestoreUser` to reconstruct from storage.
- `NewEmail`/`NewName`/`ParseID` are the only constructors for value objects — always use them, never set `.value` directly.
- Domain errors are sentinel `errors.New` values in the domain package; handlers use `errors.Is` to map them to HTTP codes.
- Generated files (`internal/delivery/http/openapi/api.gen.go`, `wire_gen.go`) are committed to the repo and must be regenerated after any change to the API spec or DI providers.
- `ErrorResponse.Error` is a `*string` (optional field). Use `openapi.Ptr("error_code")` from the generated package to set it.
- JWT middleware for class endpoints is attached via `openapi.RegisterHandlersWithOptions` `OperationMiddlewares` in `cmd/server/main.go` — not via manual route registration.
- SQLite database file (`gorm.db`) is local only; `make clean` removes it.
