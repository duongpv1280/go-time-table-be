# Go Backend Project Template (Clean Architecture + DDD)

A Go backend project template utilizing **GORM**, **Echo v4**, **OpenAPI 3.0**, and **Google Wire** following **Clean Architecture** and **Domain-Driven Design (DDD)** principles.

## Architecture

This project is built around the strict separation of concerns, ensuring business logic has no dependencies on external libraries, frameworks, or database choices.

```
┌────────────────────────────────────────────────────────┐
│                   Delivery (Echo v4)                    │
│   HTTP Handlers, Middleware, Req/Res serialization    │
└──────────────────────────┬─────────────────────────────┘
                           │ Uses
                           ▼
┌────────────────────────────────────────────────────────┐
│                   Usecase / Application                │
│   Application Orchestration, DTOs, Repository ports    │
└──────────────────────────┬─────────────────────────────┘
                           │ Uses
                           ▼
┌────────────────────────────────────────────────────────┐
│                   Domain Layer                         │
│   Entities, Value Objects, Domain Errors, Contracts     │
└────────────────────────────────────────────────────────┘
                           ▲
                           │ Implements (Dependency Inversion)
┌──────────────────────────┴─────────────────────────────┐
│                 Infrastructure (GORM SQLite)           │
│   GORM schemas, Database connection, Repository impls   │
└────────────────────────────────────────────────────────┘
```

### Layer Details

- **Domain Layer (`internal/domain/user`)**: Pure Go business rules. It contains `User` aggregate/entity and domain value objects (like `Email` and `Name`) that enforce validation upon construction. There are zero database tags or references here.
- **Usecase Layer (`internal/usecase/user`)**: Coordinates usecase actions (e.g. `CreateUser`, `GetUser`). It uses `UserRepository` interface (port) to persist aggregates and communicates with delivery layer using clean DTOs.
- **Delivery/HTTP Layer (`internal/delivery/http`)**: Receives requests, handles authentication, and outputs responses. The API definitions, model bindings, and route setups are generated directly from the OpenAPI specification using `oapi-codegen`.
- **Infrastructure Layer (`internal/infrastructure/db`)**: Provides implementation of domain repository interfaces using GORM (SQLite for quick local execution/tests). Uses distinct GORM models (`UserModel`) mapped from/to domain entities to avoid database contamination in domain.
- **DI/Wire Container (`internal/infrastructure/di`)**: Configured with Google Wire to wire repositories, usecases, handlers, and databases cleanly.

---

## Folder Structure

```
.
├── Makefile                     # Automation tasks (generate, build, test, run)
├── README.md                    # This instruction file
├── api
│   └── openapi.yaml             # OpenAPI v3 spec
├── cmd
│   └── server
│       └── main.go              # App entry point
├── go.mod                       # Go module spec
├── oapi-codegen.yaml            # Config file for OpenAPI code-generation
└── internal
    ├── delivery
    │   └── http                 # Echo handlers & OpenAPI generated code
    ├── domain
    │   └── user                 # Domain entities, value objects, repo interfaces
    └── infrastructure
        ├── db                   # GORM database connection, GORM models & repos
        └── di                   # Dependency injection setup (Google Wire)
```

---

## Setup & Running

### Prerequisites

- Go (1.21+)
- Make

### Quick Start

1. **Install Dependencies and Generate Code**
   Run the make task to install the generator binaries (`wire`, `oapi-codegen`) and generate the dependency bindings and OpenAPI routes:
   ```bash
   make generate
   ```

2. **Run Development Server**
   To start the Echo HTTP server on port `:8080` (uses/creates local file `gorm.db`):
   ```bash
   make run
   ```

3. **Run Tests**
   ```bash
   make test
   ```

4. **Verify API Endpoints**

   - **Create a User** (Valid):
     ```bash
     curl -i -X POST http://localhost:8080/users \
       -H "Content-Type: application/json" \
       -d '{"email": "john.doe@example.com", "name": "John Doe"}'
     ```

   - **Create a User** (Invalid Email):
     ```bash
     curl -i -X POST http://localhost:8080/users \
       -H "Content-Type: application/json" \
       -d '{"email": "invalid-email", "name": "John"}'
     ```

   - **List all Users**:
     ```bash
     curl -i http://localhost:8080/users
     ```
