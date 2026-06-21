# go-time-table-be — Developer Onboarding Guide

A complete guide for developers joining the project. By the end you will have a running local environment, understand the toolchain, and know how to navigate and extend the codebase.

---

## Table of Contents

1. [What is this project?](#1-what-is-this-project)
2. [Big-Picture Architecture](#2-big-picture-architecture)
3. [Prerequisites](#3-prerequisites)
4. [The Toolchain — What, Why, and How](#4-the-toolchain--what-why-and-how)
5. [Setting Up Your Local Environment](#5-setting-up-your-local-environment)
6. [Understanding the Docker Stack](#6-understanding-the-docker-stack)
7. [Project Structure Deep-Dive](#7-project-structure-deep-dive)
8. [The Four-Layer Architecture](#8-the-four-layer-architecture)
9. [Dependency Injection with Wire](#9-dependency-injection-with-wire)
10. [OpenAPI-First Development](#10-openapi-first-development)
11. [Configuration System](#11-configuration-system)
12. [Hot Reload with Air](#12-hot-reload-with-air)
13. [Database Migrations](#13-database-migrations)
14. [Testing Strategy](#14-testing-strategy)
15. [Daily Development Workflow](#15-daily-development-workflow)
16. [Adding a New Feature End-to-End](#16-adding-a-new-feature-end-to-end)
17. [Deploying](#17-deploying)
18. [Common Pitfalls and Gotchas](#18-common-pitfalls-and-gotchas)

---

## 1. What is this project?

**go-time-table-be** is the backend for a time-table / scheduling platform. It manages users, subjects, and (eventually) schedules, slots, and timetable assignments.

The backend is a **Go REST API** using the Echo framework, persisting to SQLite, and exposing an OpenAPI 3.0 contract.

---

## 2. Big-Picture Architecture

```
┌───────────────────────────────────────────────┐
│                  HTTP Client                  │
└───────────────────┬───────────────────────────┘
                    │
┌───────────────────▼───────────────────────────┐
│          Echo HTTP Server (port 8080)         │
│  Middleware: Logger → Recover                 │
│                                               │
│  Delivery Layer  (internal/delivery/http/)    │
│    ├── api.gen.go   generated server glue     │
│    ├── types.go     hand-maintained responses │
│    └── handlers/    concrete handler structs  │
│           │  calls usecase interface          │
│           ▼                                   │
│  UseCase Layer  (internal/usecase/)           │
│           │  calls domain + repository        │
│           ▼                                   │
│  Domain Layer  (internal/domain/)             │
│    ├── base/        generic repository port   │
│    ├── user/        User aggregate root       │
│    └── subject/     Subject aggregate root    │
│           ▲                                   │
│           │  implements                       │
│  Infrastructure  (internal/infrastructure/)  │
│    ├── db/          GORM/SQLite repositories  │
│    ├── config/      .env loader               │
│    └── di/          Wire DI wiring            │
│           │                                   │
│           ▼                                   │
│         SQLite (gorm.db)                      │
└───────────────────────────────────────────────┘
```

Dependency direction is strict: **Delivery → Usecase → Domain ← Infrastructure**.
The domain layer has zero external dependencies.

---

## 3. Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.24.x | The language |
| Make | System default | Orchestrates all commands |
| Docker Desktop | Latest | Optional — runs the containerised dev stack |

Install Go tools once with:

```bash
make install-tools
```

This installs `oapi-codegen`, `wire`, and `air` binaries to `~/go/bin`.

---

## 4. The Toolchain — What, Why, and How

### 4.1 Google Wire — Dependency Injection

**What:** Wire is a compile-time dependency injection generator for Go.

**Why not just manual wiring in `main.go`?**

For a small service manual wiring is fine. As the codebase grows (more usecases, more repositories), manually threading every constructor argument becomes error-prone and hard to maintain. Wire catches missing or mismatched dependencies at **compile time**.

**How it works:**

1. You declare providers in `internal/infrastructure/di/wire.go`:

```go
//go:build wireinject

func InitializeApp() (*Application, error) {
    wire.Build(
        config.Load,
        provideDBPath,
        db.NewDatabase,
        UserSet,   // wire.NewSet grouping repo + usecase + handler
        NewApplication,
    )
    return nil, nil
}
```

2. Run `make wire` — Wire reads this file and generates `wire_gen.go`, the actual `InitializeApp()` implementation with all constructors called in the correct order.
3. `wire.go` has the `wireinject` build tag — it is **never compiled by `go build`**. Only the Wire tool reads it.

**Rule:** After adding a new provider (repository, usecase, handler), always run `make wire`.

---

### 4.2 oapi-codegen — OpenAPI Code Generation

**What:** Generates Go server interfaces and request/response structs from the OpenAPI 3.0 YAML spec in `api/`.

**Why not hand-write structs?**

The spec and the implementation must stay in sync. With codegen the YAML is the single source of truth — if you define a new endpoint in the spec, the generated `ServerInterface` forces you to implement it or the code will not compile.

**How it works:**

1. Edit YAML files under `api/` (paths, schemas, request bodies).
2. Run `make oapi-codegen` — this reads `api/root.yaml` and resolves all `$ref` entries, writing `internal/delivery/http/api.gen.go`.
3. Implement any new methods on a handler struct in `internal/delivery/http/handlers/`.

**Rule:** Never edit `api.gen.go` manually — it is overwritten on every codegen run.

---

### 4.3 Air — Hot Reload

**What:** Watches `.go` and `.yaml` files, rebuilds, and restarts the server on change.

**Why not rebuild manually?**

Without hot reload every code change requires stopping the server, running `go build`, and restarting. Air automates this so you just save a file and the server reloads within a second or two.

**Config:** `.air.toml` in the project root. Binary is written to `.air/tmp/server`.

```bash
make air   # start the server with hot reload
```

---

### 4.4 golang-migrate — Database Migrations

**What:** Version-controlled SQL migration files, applied in order.

**Why not just GORM AutoMigrate?**

AutoMigrate is convenient for development — it creates and updates tables automatically at startup. But it cannot DROP columns, change column types, or run arbitrary data migrations safely. For production schema changes you need repeatable, reviewable SQL files.

This project uses **both**:
- AutoMigrate at server startup (dev convenience, always in sync with models).
- Migration files in `cmd/migration/sqls/` for explicit, versioned schema management.

**Migration files** are named `{sequence}_{description}.up.sql` / `{sequence}_{description}.down.sql`.

```bash
make migrate-create        # create a new migration pair (prompts for name)
make migrate-up-local      # apply all pending migrations
make migrate-down-local    # roll back all migrations
```

The migration runner is `cmd/migration/main.go` — a small CLI that uses the `golang-migrate/migrate/v4` library with the SQLite3 driver and a file source.

---

### 4.5 testify — Test Assertions

**What:** A testing library providing `assert` and `require` helpers.

**Why not just `t.Errorf`?**

Standard library assertions require verbose manual error formatting. testify provides concise, readable assertions that print clear diffs on failure. `require` stops the test immediately on failure; `assert` continues collecting failures.

```go
require.NoError(t, err)
assert.Equal(t, "john@example.com", dto.Email)
require.ErrorIs(t, err, user.ErrUserNotFound)
```

---

## 5. Setting Up Your Local Environment

### Step 1: Clone and configure environment variables

```bash
cp .env.example .env
```

Edit `.env` if you need non-default values:

```bash
SERVER_PORT=8080
DB_PATH=gorm.db
```

### Step 2: Install Go tools

```bash
make install-tools
```

This installs `oapi-codegen`, `wire`, and `air` to `~/go/bin`. You only need to do this once (or after upgrading Go).

### Step 3: Run the server

```bash
make run
```

This generates code, then starts the server on `:8080`.

Or with hot reload:

```bash
make air
```

### Step 4: Verify the server is running

```bash
curl http://localhost:8080/users
# → {"data":[],"pagination":{"page":0,"pageSize":0,"total":0}}
```

---

## 6. Understanding the Docker Stack

`docker-compose.yml` defines one service for local development:

### `api` — The Go backend with hot reload

```yaml
services:
  api:
    build:
      context: .
      dockerfile: Dockerfile_local
    ports:
      - "8080:8080"
    volumes:
      - .:/go/src/app
    env_file:
      - .env
```

`Dockerfile_local` builds on `golang:1.24.0-alpine`, installs `air`, and runs the server with hot reload. The project directory is volume-mounted so file changes on your host are immediately visible inside the container.

```bash
docker compose up -d          # start in background
docker compose logs -f api    # watch logs
docker compose down           # stop
```

The SQLite database (`gorm.db`) lives on the host volume mount — it persists between container restarts.

---

## 7. Project Structure Deep-Dive

```
go-time-table-be/
├── api/                          OpenAPI spec (source of truth)
│   ├── root.yaml                 Entry point — oapi-codegen reads this
│   ├── paths/
│   │   ├── users.yaml            GET /users, POST /users
│   │   └── users-by-id.yaml     GET /users/{id}, DELETE /users/{id}
│   ├── requestBodies/
│   │   └── create-user.yaml
│   └── schemas/
│       ├── user.yaml
│       ├── list-users-item.yaml
│       ├── list-users-response.yaml
│       ├── create-user-request.yaml
│       ├── pagination.yaml
│       └── error-response.yaml
│
├── cmd/
│   ├── server/
│   │   └── main.go              Entry point — loads config, starts Echo
│   └── migration/
│       ├── main.go              Migration CLI (up / down / version)
│       └── sqls/
│           ├── 000001_create_users_table.up.sql
│           ├── 000001_create_users_table.down.sql
│           ├── 000002_create_subjects_table.up.sql
│           └── 000002_create_subjects_table.down.sql
│
├── internal/
│   ├── delivery/http/            HTTP delivery layer
│   │   ├── api.gen.go            Generated — ServerInterface + request types
│   │   ├── types.go              Hand-maintained response types
│   │   └── handlers/
│   │       └── user.go           UserHandler implementing ServerInterface
│   │
│   ├── domain/                   Domain layer — zero external dependencies
│   │   ├── base/
│   │   │   └── repository.go     Generic IRepository[T, ID] port
│   │   ├── user/
│   │   │   ├── entity.go         User aggregate root (private fields)
│   │   │   ├── value-objects.go  ID, Email, Name value objects + errors
│   │   │   ├── repository.go     IUserRepository interface
│   │   │   └── entity_test.go
│   │   └── subject/
│   │       ├── entity.go         Subject aggregate root
│   │       ├── value-objects.go  ID, Name value objects + errors
│   │       └── repository.go     ISubjectRepository interface
│   │
│   ├── usecase/                  Usecase layer — business logic
│   │   └── user/
│   │       ├── usecase.go        IUserUseCase interface + implementation
│   │       ├── dto.go            CreateUserParams, UserDTO, mapping helpers
│   │       └── usecase_test.go
│   │
│   └── infrastructure/           Infrastructure layer
│       ├── config/
│       │   └── config.go         Loads .env → Config struct
│       ├── db/
│       │   ├── db.go             Opens SQLite connection + AutoMigrate
│       │   ├── models.go         UserModel GORM struct + domain mapping
│       │   ├── subject-model.go  SubjectModel GORM struct + domain mapping
│       │   ├── user-repository.go    GORM IUserRepository implementation
│       │   └── subject-repository.go GORM ISubjectRepository implementation
│       └── di/
│           ├── wire.go           DI provider declarations (hand-written)
│           └── wire_gen.go       DI wiring (auto-generated — do not edit)
│
├── docs/
│   ├── ONBOARDING.md             This file
│   └── tasks/
│       └── TEMPLATE.md           Task spec template
│
├── .air.toml                     Air hot-reload configuration
├── .env                          Local environment variables (git-ignored)
├── .env.example                  Template committed to repo
├── docker-compose.yml            Local Docker stack
├── Dockerfile                    Production multi-stage build
├── Dockerfile_local              Local dev image (uses Air)
├── Makefile                      All developer commands
├── oapi-codegen.yaml             oapi-codegen configuration
├── go.mod
└── go.sum
```

### How the entry-point files work together

```
cmd/server/main.go
    └── config.Load()          reads .env
    └── di.InitializeApp()     Wire-generated: wires all dependencies
    └── echo.New()             Echo server setup
    └── http.RegisterHandlers  registers generated routes
    └── e.Start(":PORT")
```

`main.go` is minimal — it loads config, initialises the DI container, sets up Echo middleware, registers routes, and starts listening. All constructor wiring is in `wire_gen.go`.

---

## 8. The Four-Layer Architecture

**Never skip layers.** The dependency direction is strictly enforced.

```
HTTP Request
     │
     ▼
┌────────────────────────────────────────────┐
│ Delivery   internal/delivery/http/         │
│  Parses HTTP, calls ONE usecase method,    │
│  maps domain errors to HTTP status codes.  │
│  No business logic. No direct DB access.   │
└───────────────────┬────────────────────────┘
                    │ calls interface
                    ▼
┌────────────────────────────────────────────┐
│ UseCase    internal/usecase/               │
│  All business logic. Accepts/returns DTOs. │
│  Constructs domain value objects from raw  │
│  input. Calls repository interfaces.       │
│  No echo.Context. No GORM.                 │
└───────────────────┬────────────────────────┘
                    │ calls interface
                    ▼
┌────────────────────────────────────────────┐
│ Domain     internal/domain/                │
│  Aggregate roots with private fields.      │
│  Value objects validate on construction.   │
│  Sentinel errors (ErrNotFound, etc.).      │
│  Repository interfaces defined here.       │
│  Zero external imports.                    │
└────────────────────────────────────────────┘
                    ▲
                    │ implements
┌────────────────────────────────────────────┐
│ Infrastructure  internal/infrastructure/   │
│  GORM/SQLite implementations of repos.     │
│  Separate model structs with GORM tags.    │
│  Maps between model ↔ domain entity.       │
└────────────────────────────────────────────┘
```

### Domain value objects

Each domain entity uses value objects for validated fields:

```go
// Construction validates — never set .value directly
email, err := user.NewEmail("john@example.com")
name,  err := user.NewName("John")
id,    err := user.ParseID("123e4567-...")

// Aggregate root has only private fields — access via getters
u := user.NewUser(email, name)
u.Email().String()   // "john@example.com"
u.ID().String()      // UUID string
```

### Domain errors

```go
// Defined in domain package as sentinel values
var ErrUserNotFound      = errors.New("user not found")
var ErrUserAlreadyExists = errors.New("user already exists with this email")
var ErrInvalidEmail      = errors.New("invalid email address")
```

Handlers use `errors.Is` to map them to HTTP status codes:

```go
if errors.Is(err, user.ErrUserNotFound) {
    return ctx.JSON(http.StatusNotFound, ...)
}
```

### Infrastructure model separation

The infrastructure layer uses separate GORM model structs. This keeps ORM tags (and the fragile `mattn/go-sqlite3` CGO dependency) out of the domain:

```go
// domain/user/entity.go — pure domain, no gorm tags
type User struct {
    id    ID
    email Email
    name  Name
    ...
}

// infrastructure/db/models.go — ORM struct
type UserModel struct {
    ID        string    `gorm:"primaryKey;type:uuid"`
    Email     string    `gorm:"uniqueIndex;not null"`
    Name      string    `gorm:"not null"`
    CreatedAt time.Time `gorm:"not null"`
    UpdatedAt time.Time `gorm:"not null"`
}

// Explicit mapping — no magic
func (m *UserModel) ToDomain() (*user.User, error) { ... }
func FromDomain(u *user.User) *UserModel            { ... }
```

---

## 9. Dependency Injection with Wire

### Adding a new repository and usecase

1. Define the interface in the domain package:

```go
// internal/domain/subject/repository.go
type ISubjectRepository interface {
    base.IRepository[*Subject, ID]
    // add domain-specific methods here
}
```

2. Implement it in infrastructure:

```go
// internal/infrastructure/db/subject-repository.go
func NewGormSubjectRepository(db *gorm.DB) subject.ISubjectRepository {
    return &gormSubjectRepository{db: db}
}
```

3. Create the usecase:

```go
// internal/usecase/subject/usecase.go
func NewSubjectUseCase(repo subject.ISubjectRepository) ISubjectUseCase {
    return &subjectUseCase{repo: repo}
}
```

4. Register in `wire.go`:

```go
var SubjectSet = wire.NewSet(
    db.NewGormSubjectRepository,
    subjectusecase.NewSubjectUseCase,
    handlers.NewSubjectHandler,
)

func InitializeApp() (*Application, error) {
    wire.Build(
        // ...
        SubjectSet,
        NewApplication,
    )
    return nil, nil
}
```

5. Run `make wire`.

---

## 10. OpenAPI-First Development

### The workflow

```
api/paths/myresource.yaml   ← you write this
         │
         ▼  make oapi-codegen
         │
internal/delivery/http/api.gen.go
    ├── ServerInterface  (all endpoints as methods)
    └── request/param types
         │
         ▼  implement
         │
internal/delivery/http/handlers/myresource.go
```

### What the generated interface looks like

```go
// api.gen.go — generated, do not edit
type ServerInterface interface {
    CreateUser(ctx echo.Context) error
    ListUsers(ctx echo.Context, params ListUsersParams) error
    GetUser(ctx echo.Context, id openapi_types.UUID) error
    DeleteUser(ctx echo.Context, id openapi_types.UUID) error
}
```

Your handler must implement every method in this interface. Adding a YAML endpoint without a handler method causes a compile error.

### Response types

`internal/delivery/http/types.go` is **hand-maintained** and defines response structs that mirror the OpenAPI schemas. oapi-codegen does not generate response types from external `$ref` entries. Keep `types.go` in sync with `api/schemas/*.yaml`.

### API spec structure

```
api/
├── root.yaml               Entry point (openapi, info, paths only)
├── paths/
│   ├── users.yaml          GET /users, POST /users
│   └── users-by-id.yaml   GET /users/{id}, DELETE /users/{id}
├── requestBodies/
│   └── create-user.yaml
└── schemas/
    ├── user.yaml
    ├── list-users-response.yaml
    ├── create-user-request.yaml
    ├── pagination.yaml
    └── error-response.yaml
```

---

## 11. Configuration System

Configuration is loaded from `.env` at startup by `internal/infrastructure/config/config.go` using `godotenv`.

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | HTTP listen port |
| `DB_PATH` | `gorm.db` | SQLite database file path |

The config struct is wired through DI into the database and server:

```
config.Load() → Config.DBPath → db.NewDatabase(dbPath)
              → Config.ServerPort → e.Start(":PORT")
```

For local development, copy `.env.example` to `.env` and edit as needed. The `.env` file is git-ignored.

---

## 12. Hot Reload with Air

Air is configured in `.air.toml`. When running `make air`:

1. Air watches `.go` and `.yaml` files in the project.
2. On change, it runs `go build -o .air/tmp/server ./cmd/server/main.go`.
3. If the build succeeds, Air kills the old process and starts the new binary.
4. If the build fails, the old binary keeps running and the error is printed to stdout.

Generated files (`api.gen.go`, `wire_gen.go`) are excluded from triggering rebuilds to avoid infinite loops.

**Note:** When you change the OpenAPI spec or DI providers, you need to run `make generate` first, then Air will pick up the regenerated files.

---

## 13. Database Migrations

The project uses two complementary mechanisms:

### GORM AutoMigrate (server startup)

`internal/infrastructure/db/db.go` calls `db.AutoMigrate(&UserModel{}, &SubjectModel{})` on every startup. This keeps development fast — models always match the database schema.

### golang-migrate (explicit versioned migrations)

Migration files in `cmd/migration/sqls/` are the authoritative schema definition. They are numbered sequentially (`000001_`, `000002_`, ...) and tracked in a `schema_migrations` table in the database.

```bash
# Create a new migration
make migrate-create
# Prompts: Migration name: add_timetable_slots
# Creates: cmd/migration/sqls/000003_add_timetable_slots.{up,down}.sql

# Apply all pending migrations
make migrate-up-local

# Roll back all migrations
make migrate-down-local
```

The migration CLI accepts a `-steps N` flag:

```bash
go run ./cmd/migration -cmd up -steps 1    # apply one migration
go run ./cmd/migration -cmd down -steps 1  # roll back one migration
go run ./cmd/migration -cmd version        # show current version
```

### When to write a migration

Write a migration file whenever you:
- Add or remove a table
- Add, rename, or remove a column
- Add or change an index
- Need to backfill data

After writing the SQL, run `make migrate-up-local` to apply it locally, then verify the schema matches the GORM models.

---

## 14. Testing Strategy

### Unit tests — no database

Tests in `internal/domain/` and `internal/usecase/` use hand-written in-memory mock repositories. No running server or database is needed.

```go
func TestCreateUser(t *testing.T) {
    repo := newMockUserRepository()
    uc := usecase.NewUserUseCase(repo)

    dto, err := uc.CreateUser(context.Background(), usecase.CreateUserParams{
        Email: "john@example.com",
        Name:  "John",
    })

    require.NoError(t, err)
    assert.Equal(t, "john@example.com", dto.Email)
}
```

```bash
make test                          # all tests
go test -v ./internal/domain/...   # domain only
go test -v ./internal/usecase/...  # usecase only
```

### Test naming convention

Test names must describe expected behaviour:

```go
// Good
func TestCreateUser_InvalidEmail_ReturnsBadRequest(t *testing.T)
func TestGetUser_NotFound_ReturnsError(t *testing.T)

// Bad
func TestCreateUser(t *testing.T)
func TestGet(t *testing.T)
```

### Table-driven tests

Use table-driven tests for value-object validation:

```go
func TestNewEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid email", "test@example.com", false},
        {"invalid format", "not-an-email", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := user.NewEmail(tt.email)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Writing mock repositories

Until mockery is added to the project, mock repositories are hand-written in `_test.go` files in the same package. A mock simply satisfies the domain repository interface using an in-memory map:

```go
type mockUserRepository struct {
    users     map[user.ID]*user.User
    createErr error
}

func (m *mockUserRepository) Create(ctx context.Context, u *user.User) error {
    if m.createErr != nil {
        return m.createErr
    }
    m.users[u.ID()] = u
    return nil
}
// implement FindByID, FindAll, Delete...
```

---

## 15. Daily Development Workflow

### Starting work

```bash
make air                     # hot-reload server
# or: make run              # one-shot start
```

### After editing the OpenAPI spec

```bash
make oapi-codegen            # regenerate api.gen.go
# implement any new methods in handlers/
```

### After adding a new DI provider

```bash
make wire                    # regenerate wire_gen.go
```

### After changing a database schema

```bash
# 1. Update GORM model struct
# 2. Create migration file
make migrate-create
# 3. Write SQL in the generated file
# 4. Apply it
make migrate-up-local
```

### Before committing

```bash
make lint                    # go vet
make test                    # full test suite
make build                   # confirm it compiles
```

---

## 16. Adding a New Feature End-to-End

Let us trace adding `GET /subjects` and `POST /subjects`.

### Step 1: Define the API contract

Add `api/paths/subjects.yaml`:

```yaml
get:
  operationId: listSubjects
  summary: List all subjects
  responses:
    "200":
      description: OK
      content:
        application/json:
          schema:
            $ref: "../schemas/list-subjects-response.yaml"

post:
  operationId: createSubject
  summary: Create a subject
  requestBody:
    $ref: "../requestBodies/create-subject.yaml"
  responses:
    "201":
      description: Created
      content:
        application/json:
          schema:
            $ref: "../schemas/subject.yaml"
    "400":
      $ref: "../schemas/error-response.yaml"
```

Add the `$ref` entry in `api/root.yaml` and create the schema files. Then:

```bash
make oapi-codegen
```

### Step 2: Add the usecase

`internal/usecase/subject/usecase.go`:

```go
type ISubjectUseCase interface {
    CreateSubject(ctx context.Context, params CreateSubjectParams) (SubjectDTO, error)
    ListSubjects(ctx context.Context) ([]SubjectDTO, error)
}

func NewSubjectUseCase(repo subject.ISubjectRepository) ISubjectUseCase {
    return &subjectUseCase{repo: repo}
}
```

### Step 3: Implement the handler

`internal/delivery/http/handlers/subject.go`:

```go
type SubjectHandler struct {
    useCase subjectusecase.ISubjectUseCase
}

func (h *SubjectHandler) ListSubjects(ctx echo.Context) error {
    subjects, err := h.useCase.ListSubjects(ctx.Request().Context())
    if err != nil {
        return ctx.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "internal error"})
    }
    // map and return
    return ctx.JSON(http.StatusOK, ...)
}
```

### Step 4: Register in Wire

```go
// internal/infrastructure/di/wire.go
var SubjectSet = wire.NewSet(
    db.NewGormSubjectRepository,
    subjectusecase.NewSubjectUseCase,
    handlers.NewSubjectHandler,
)
```

Run `make wire`.

### Step 5: Write the migration

```bash
make migrate-create
# Migration name: add_subjects_table
```

Edit `cmd/migration/sqls/000003_add_subjects_table.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS subjects (
    id         TEXT     NOT NULL PRIMARY KEY,
    name       TEXT     NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
```

Apply: `make migrate-up-local`

### Step 6: Write tests

```go
func TestCreateSubject_ReturnsSubjectDTO(t *testing.T) {
    repo := newMockSubjectRepository()
    uc := subjectusecase.NewSubjectUseCase(repo)

    dto, err := uc.CreateSubject(context.Background(), subjectusecase.CreateSubjectParams{
        Name: "Mathematics",
    })

    require.NoError(t, err)
    assert.Equal(t, "Mathematics", dto.Name)
    assert.NotEmpty(t, dto.ID)
}
```

### Step 7: Lint and test

```bash
make lint
make test
```

---

## 17. Deploying

The production Docker image is a two-stage build defined in `Dockerfile`:

```dockerfile
# Stage 1: Build
FROM golang:1.24.0-alpine as build
WORKDIR /go/src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /go/bin/server ./cmd/server/main.go

# Stage 2: Runtime (minimal)
FROM alpine:3.18 as deploy
COPY --from=build /go/bin/server /go/bin/server
CMD ["/go/bin/server"]
```

Environment variables are injected at runtime (via `--env-file` or container environment settings) — not baked into the image.

---

## 18. Common Pitfalls and Gotchas

### "wire: type X is not a provider"

You added a new constructor to `wire.go` but either forgot to import its package, or the return type does not match what Wire expects (it must return the interface, not the concrete struct).

### "api.gen.go has a compile error after editing the spec"

`api.gen.go` is generated from the spec. If you edited the spec incorrectly or a `$ref` is broken, codegen will produce invalid Go. Check the oapi-codegen output for parse errors, fix the YAML, and re-run `make oapi-codegen`.

### "handler does not compile — missing method"

The generated `ServerInterface` requires every endpoint in the spec to be implemented. If you added a YAML endpoint without a handler method, the compiler will report the missing method. Implement it or remove the spec entry.

### "field unexported — cannot set directly"

Domain aggregate fields are private. Always use `NewUser(email, name)` to construct, `RestoreUser(...)` to reconstruct from storage, and value-object constructors (`NewEmail`, `NewName`, `ParseID`) for all fields.

### "test: mock repository missing method"

If you add a method to a domain repository interface, all hand-written mocks in test files must be updated to implement the new method.

### "migration error: dirty database"

If a migration fails partway through, the `schema_migrations` table will have `dirty = 1`. Fix the failing SQL, then manually clear the dirty flag before re-running:

```bash
# Open the SQLite file with any client and run:
UPDATE schema_migrations SET dirty = 0;
```

### "SQL injection — using fmt.Sprintf in a GORM query"

Always use GORM's parameterised queries:

```go
// Dangerous — never do this
db.Raw(fmt.Sprintf("SELECT * FROM users WHERE id = '%s'", userInput))

// Safe — always do this
db.Where("id = ?", userInput).First(&model)
```

---

## Quick Reference Card

```bash
# One-time setup
make install-tools

# Daily
make air                    # start with hot reload
make run                    # start without hot reload

# Code generation
make oapi-codegen           # after editing api/*.yaml
make wire                   # after adding a DI provider
make generate               # both at once

# Database
make migrate-create         # create new migration file pair
make migrate-up-local       # apply pending migrations
make migrate-down-local     # roll back migrations

# Quality
make lint                   # go vet
make test                   # all tests
make build                  # compile check

# Useful URLs
http://localhost:8080/users  # Users API
```
