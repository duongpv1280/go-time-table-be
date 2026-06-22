# Test Plan: GS-003 — Validators and Custom Validators for Class POST/PUT Endpoints

## Scope

This plan covers all tests needed to verify the `IValidator`/`Validator` infrastructure, the `unique_in` custom validation rule, the new `POST /api/v1/classes` and `PUT /api/v1/classes/:classId` HTTP endpoints (ADMIN-only), the `CreateClass`/`UpdateClass` usecase methods, and the `UpdateName`/`UpdateGrade` mutation methods on the `class.Class` entity.

Out of scope: existing GET endpoints (covered by GS-002 test plan), other domains (users, subjects, auth), frontend code, infrastructure-level DB migration scripts, and OpenAPI spec codegen correctness.

---

## Test Environment

- **Go version**: as declared in `go.mod` (1.21+)
- **Database**: SQLite in-memory (`gorm.io/driver/sqlite` with `file::memory:?cache=shared`) for integration tests; pure mock structs for unit tests
- **HTTP framework**: Echo v4 via `net/http/httptest` — no real server process started
- **Assertion library**: `github.com/testify/assert` + `github.com/testify/require`
- **Required env vars**: none (in-memory DB, mocked JWT)
- **Seed data keys** (used across test cases — see Test Data Catalogue):
  - `CLASS_10A_ID` — UUID of the seeded "10A" class
  - `CLASS_11B_ID` — UUID of a second seeded class used as a collision target
  - `ADMIN_TOKEN` — JWT that Verify() resolves to `{UserID:"admin-1", Role:"ADMIN"}`
  - `TEACHER_TOKEN` — JWT that Verify() resolves to `{UserID:"teacher-1", Role:"TEACHER"}`
  - `STUDENT_TOKEN` — JWT that Verify() resolves to `{UserID:"student-1", Role:"STUDENT"}`

---

## Acceptance Criteria Coverage Matrix

| AC | Description | Test Cases |
|----|-------------|------------|
| AC-1 | `IValidator` and `Validator` can be reused for many other rules | TC-001, TC-002, TC-003, TC-004 |
| AC-2 | Test coverage for each rule must be 100% | TC-005, TC-006, TC-007, TC-008, TC-009, TC-010, TC-011, TC-012, TC-013, TC-014, TC-015, TC-016 |
| AC-3 | Must have integration test | TC-060, TC-061, TC-062, TC-063, TC-064, TC-065, TC-066, TC-067 |
| AC-4 | Pass all tests | All TCs must pass; no build errors |

---

## Test Cases

---

### Group A — Unit: Validator Infrastructure (`internal/delivery/http/validator/rules/validator_test.go`)

---

### TC-001: NewValidator returns a non-nil IValidator
- **Type**: Unit
- **Covers**: AC-1 — IValidator and Validator can be reused for many other rules
- **Pre-conditions**: A real `*gorm.DB` instance (in-memory SQLite) is available
- **Test data**:
  - Input: `rules.NewValidator(db)` where `db` is a valid `*gorm.DB`
  - Expected output: returned value is non-nil and satisfies the `validator.IValidator` interface
  - Supporting data: SQLite in-memory DB opened with `gorm.Open`
- **Steps**:
  1. Open an in-memory SQLite DB.
  2. Call `rules.NewValidator(db)`.
  3. Assert: returned value is not nil.
  4. Assert: returned value implements `validator.IValidator` (compile-time assignability check via `var _ validator.IValidator = rules.NewValidator(db)`).
- **Pass criteria**: `NewValidator` returns a non-nil value; the assignment to `validator.IValidator` compiles without error.
- **Fail indicators**: nil pointer returned; compiler error on interface assignment; panic on construction.

---

### TC-002: ValidateCtx passes for a structurally valid struct
- **Type**: Unit
- **Covers**: AC-1 — IValidator and Validator can be reused for many other rules
- **Pre-conditions**: `Validator` constructed with an in-memory SQLite DB that has the `classes` table created.
- **Test data**:
  - Input: `ValidateCtx(context.Background(), struct{ Name string \`validate:"required"\` }{Name: "10A"})`
  - Expected output: `nil` error
  - Supporting data: none beyond DB fixture
- **Steps**:
  1. Construct `Validator` with DB.
  2. Call `ValidateCtx` with a struct carrying a `required` tag and a non-empty `Name` value `"10A"`.
  3. Assert: returned error is nil.
- **Pass criteria**: `ValidateCtx` returns nil.
- **Fail indicators**: non-nil error returned; panic.

---

### TC-003: ValidateCtx fails for a struct with a missing required field
- **Type**: Unit
- **Covers**: AC-1 — IValidator and Validator can be reused for many other rules
- **Pre-conditions**: `Validator` constructed.
- **Test data**:
  - Input: `ValidateCtx(context.Background(), struct{ Name string \`validate:"required"\` }{Name: ""})`
  - Expected output: non-nil error; error message or type indicates `required` tag failure
  - Supporting data: none
- **Steps**:
  1. Construct `Validator`.
  2. Call `ValidateCtx` with a struct where `Name` is an empty string and the tag is `required`.
  3. Assert: returned error is non-nil.
- **Pass criteria**: `ValidateCtx` returns a non-nil error.
- **Fail indicators**: nil error returned (validation silently passed).

---

### TC-004: unique_in tag is registered and invokable by the Validator
- **Type**: Unit
- **Covers**: AC-1 — IValidator and Validator can be reused for many other rules
- **Pre-conditions**: `Validator` constructed with an in-memory SQLite DB that has the `classes` table. The table is empty.
- **Test data**:
  - Input: struct with `Name string \`validate:"unique_in:classes:name"\`` and `Name = "NewClass"`
  - Expected output: nil error (name does not exist in the empty DB)
  - Supporting data: DB with `classes` table auto-migrated via GORM
- **Steps**:
  1. Construct in-memory SQLite DB; auto-migrate `classes` table.
  2. Construct `Validator`.
  3. Define anonymous struct `{ Name string \`validate:"unique_in:classes:name"\` }` with `Name = "NewClass"`.
  4. Call `ValidateCtx(ctx, struct)`.
  5. Assert: nil error (proving the tag was found and the handler executed without panicking).
- **Pass criteria**: `ValidateCtx` returns nil; no "undefined tag" error or panic.
- **Fail indicators**: error saying the `unique_in` tag is unknown; panic; non-nil error when table is empty.

---

### Group B — Unit: UniqueInValidator Rule (`internal/delivery/http/validator/rules/unique_in_test.go`)

All tests in this group call `UniqueInValidator` directly or exercise it through a struct tag and `ValidateCtx`.

---

### TC-005: Value that does NOT exist in DB returns true (valid)
- **Type**: Unit
- **Covers**: AC-2 — 100% rule coverage; happy path of uniqueness check
- **Pre-conditions**: `classes` table exists in the test DB and is empty.
- **Test data**:
  - Input: validate struct `{ Name string \`validate:"unique_in:classes:name"\` }` with `Name = "NonExistent"`
  - Expected output: `ValidateCtx` returns nil (true = valid)
  - Supporting data: empty `classes` table
- **Steps**:
  1. Create in-memory DB; migrate `classes` table.
  2. Construct `Validator`.
  3. Call `ValidateCtx(context.Background(), input)`.
  4. Assert: nil error.
- **Pass criteria**: nil error returned.
- **Fail indicators**: non-nil error (false positive — name flagged as duplicate when table is empty).

---

### TC-006: Value that EXISTS in DB returns false (invalid)
- **Type**: Unit
- **Covers**: AC-2 — 100% rule coverage; duplicate name detection
- **Pre-conditions**: `classes` table contains a row with `name = "10A"`.
- **Test data**:
  - Input: struct `{ Name string \`validate:"unique_in:classes:name"\` }` with `Name = "10A"`
  - Expected output: `ValidateCtx` returns non-nil validation error for the `Name` field
  - Supporting data: seed one ClassModel row `{ID: uuid, Name: "10A", Grade: 10}`
- **Steps**:
  1. Create in-memory DB; migrate; seed one row with `name = "10A"`.
  2. Construct `Validator`.
  3. Call `ValidateCtx(context.Background(), input)`.
  4. Assert: error is non-nil.
- **Pass criteria**: non-nil error is returned.
- **Fail indicators**: nil error (false negative — duplicate name accepted).

---

### TC-007: Value that EXISTS in DB but is excluded by ExcludeIDKey returns true (valid — self-reference)
- **Type**: Unit
- **Covers**: AC-2 — 100% rule coverage; self-exclusion branch for PUT
- **Pre-conditions**: `classes` table contains a row with `id = CLASS_10A_ID`, `name = "10A"`.
- **Test data**:
  - Input: struct with `Name = "10A"`; context enriched with `context.WithValue(ctx, rules.ExcludeIDKey, CLASS_10A_ID)`
  - Expected output: `ValidateCtx` returns nil (the row that matched is excluded by ID)
  - Supporting data: seed one ClassModel row `{ID: CLASS_10A_ID, Name: "10A", Grade: 10}`
- **Steps**:
  1. Create in-memory DB; migrate; seed row `{ID: CLASS_10A_ID, Name: "10A"}`.
  2. Construct `Validator`.
  3. Build context: `ctx = context.WithValue(context.Background(), rules.ExcludeIDKey, CLASS_10A_ID)`.
  4. Call `ValidateCtx(ctx, input)`.
  5. Assert: nil error.
- **Pass criteria**: nil error (own name allowed when ID is excluded).
- **Fail indicators**: non-nil error (own name incorrectly blocked on PUT).

---

### TC-008: Invalid param format (tag with no colon separator) returns false (invalid config)
- **Type**: Unit
- **Covers**: AC-2 — 100% rule coverage; malformed tag branch
- **Pre-conditions**: `Validator` constructed. DB can be in any state.
- **Test data**:
  - Input: struct `{ Name string \`validate:"unique_in:classesnocolon"\` }` with `Name = "Whatever"` (param string has only one segment — no second colon)
  - Expected output: `ValidateCtx` returns non-nil error (rule returns false on bad param)
  - Supporting data: none
- **Steps**:
  1. Construct `Validator`.
  2. Call `ValidateCtx(context.Background(), input)`.
  3. Assert: error is non-nil.
- **Pass criteria**: non-nil error returned.
- **Fail indicators**: nil error (malformed tag silently ignored); panic.

---

### TC-009: Empty field value queried against a non-existent table returns false (DB error path)
- **Type**: Unit
- **Covers**: AC-2 — 100% rule coverage; DB error / non-existent table branch
- **Pre-conditions**: DB has NOT been migrated — the `nonexistent_table` table does not exist.
- **Test data**:
  - Input: struct `{ Name string \`validate:"unique_in:nonexistent_table:name"\` }` with `Name = ""`
  - Expected output: `ValidateCtx` returns non-nil error (DB error causes rule to return false)
  - Supporting data: in-memory DB with no tables
- **Steps**:
  1. Create in-memory DB without running migrations.
  2. Construct `Validator`.
  3. Call `ValidateCtx(context.Background(), input)`.
  4. Assert: error is non-nil.
- **Pass criteria**: non-nil error returned (rule correctly returns false on DB error).
- **Fail indicators**: nil error; panic propagating from DB layer instead of returning false.

---

### TC-010: UniqueInValidator — case-sensitive match (different case not treated as duplicate)
- **Type**: Unit
- **Covers**: AC-2 — 100% rule coverage; case sensitivity boundary
- **Pre-conditions**: `classes` table contains `name = "10a"` (lower-case).
- **Test data**:
  - Input: struct with `Name = "10A"` (upper-case A)
  - Expected output: depends on DB collation; document the expected behaviour. For SQLite default (case-insensitive LIKE but case-sensitive `=`): `"10A" != "10a"` → nil error (valid).
  - Supporting data: seed row `{Name: "10a"}`
- **Steps**:
  1. Seed row with `name = "10a"`.
  2. Validate struct with `Name = "10A"`.
  3. Assert: nil error (SQLite `=` is case-sensitive for ASCII).
- **Pass criteria**: nil error; pass criterion changes if the query uses `LOWER()` or `ILIKE` — document if so.
- **Fail indicators**: non-nil error (case-insensitive match producing false positive).

---

### Group C — Unit: Class Entity Mutation Methods (`internal/domain/class/entity_test.go`)

---

### TC-011: UpdateName sets the new name and bumps updatedAt
- **Type**: Unit
- **Covers**: AC-2 — 100% coverage of new entity mutation methods
- **Pre-conditions**: A `Class` entity is constructed with `name = "10A"`, `grade = 10`.
- **Test data**:
  - Input: `c.UpdateName(newName)` where `newName` is the result of `class.NewName("11B")`
  - Expected output: `c.Name().String() == "11B"`; `c.UpdatedAt()` is strictly after `c.CreatedAt()`
  - Supporting data: none (pure domain unit test)
- **Steps**:
  1. Record `before := time.Now()`.
  2. Construct class `c` with name `"10A"`.
  3. Call `c.UpdateName(newName)` where `newName` was constructed from `"11B"`.
  4. Assert: `c.Name().String() == "11B"`.
  5. Assert: `c.UpdatedAt().After(before)` is true.
- **Pass criteria**: name field changed to `"11B"` AND `updatedAt` is after the pre-call timestamp.
- **Fail indicators**: name unchanged; `updatedAt` equal to `createdAt` (not bumped).

---

### TC-012: UpdateGrade sets the new grade and bumps updatedAt
- **Type**: Unit
- **Covers**: AC-2 — 100% coverage of new entity mutation methods
- **Pre-conditions**: A `Class` entity is constructed with `grade = 10`.
- **Test data**:
  - Input: `c.UpdateGrade(newGrade)` where `newGrade` is the result of `class.NewGrade(12)`
  - Expected output: `c.Grade().Value() == 12`; `c.UpdatedAt()` is strictly after `c.CreatedAt()`
  - Supporting data: none
- **Steps**:
  1. Record `before := time.Now()`.
  2. Construct class `c` with grade `10`.
  3. Call `c.UpdateGrade(newGrade)`.
  4. Assert: `c.Grade().Value() == 12`.
  5. Assert: `c.UpdatedAt().After(before)` is true.
- **Pass criteria**: grade changed to `12` AND `updatedAt` is after the pre-call timestamp.
- **Fail indicators**: grade unchanged; `updatedAt` not bumped.

---

### TC-013: UpdateName does not change grade
- **Type**: Unit
- **Covers**: AC-2 — mutation isolation
- **Pre-conditions**: Class constructed with `name = "10A"`, `grade = 10`.
- **Test data**:
  - Input: `c.UpdateName(newName)` where `newName = "11B"`
  - Expected output: `c.Grade().Value() == 10` (unchanged)
  - Supporting data: none
- **Steps**:
  1. Construct class with grade `10`.
  2. Call `UpdateName("11B")`.
  3. Assert: `c.Grade().Value() == 10`.
- **Pass criteria**: grade is exactly `10` after `UpdateName`.
- **Fail indicators**: grade changed.

---

### TC-014: UpdateGrade does not change name
- **Type**: Unit
- **Covers**: AC-2 — mutation isolation
- **Pre-conditions**: Class constructed with `name = "10A"`, `grade = 10`.
- **Test data**:
  - Input: `c.UpdateGrade(newGrade)` where `newGrade.Value() == 12`
  - Expected output: `c.Name().String() == "10A"` (unchanged)
  - Supporting data: none
- **Steps**:
  1. Construct class with name `"10A"`.
  2. Call `UpdateGrade(12)`.
  3. Assert: `c.Name().String() == "10A"`.
- **Pass criteria**: name is exactly `"10A"` after `UpdateGrade`.
- **Fail indicators**: name changed.

---

### Group D — Unit: Class Usecase — CreateClass and UpdateClass (`internal/usecase/class/usecase_test.go`)

The mock repository must be extended with `Create(ctx, *Class) error` and `Update(ctx, *Class) error` stubs.

---

### TC-015: CreateClass — ADMIN role, valid params — returns ClassDTO, no error
- **Type**: Unit
- **Covers**: AC-4 — pass all tests; happy path
- **Pre-conditions**: Mock repo `Create` returns nil. Permission `{Role: "ADMIN"}`.
- **Test data**:
  - Input: `CreateClass(ctx, CreateClassParams{Name: "10A", Grade: 10}, perm{Role:"ADMIN"})`
  - Expected output: `ClassDTO{Name: "10A", Grade: 10}`, nil error
  - Supporting data: mock repo; no DB
- **Steps**:
  1. Construct mock repo with `Create` returning nil.
  2. Call `uc.CreateClass(ctx, params, adminPerm)`.
  3. Assert: nil error.
  4. Assert: returned `ClassDTO.Name == "10A"` and `ClassDTO.Grade == 10`.
  5. Assert: `ClassDTO.ID` is a valid non-empty UUID string.
- **Pass criteria**: nil error; DTO fields match input; ID is non-empty.
- **Fail indicators**: non-nil error; zero-value DTO; empty ID.

---

### TC-016: CreateClass — TEACHER role — returns ErrUnauthorized
- **Type**: Unit
- **Covers**: AC-2 — auth enforcement in usecase
- **Pre-conditions**: Mock repo. Permission `{Role: "TEACHER"}`.
- **Test data**:
  - Input: `CreateClass(ctx, CreateClassParams{Name: "10A", Grade: 10}, perm{Role:"TEACHER"})`
  - Expected output: error `== domainAuth.ErrUnauthorized`
  - Supporting data: mock repo (Create should NOT be called)
- **Steps**:
  1. Call `uc.CreateClass` with TEACHER permission.
  2. Assert: error is `domainAuth.ErrUnauthorized`.
  3. Assert: mock repo `Create` was NOT called (create should be blocked before repo).
- **Pass criteria**: `errors.Is(err, domainAuth.ErrUnauthorized)` is true.
- **Fail indicators**: nil error; wrong error type; repo Create called.

---

### TC-017: CreateClass — STUDENT role — returns ErrUnauthorized
- **Type**: Unit
- **Covers**: AC-2 — auth enforcement in usecase
- **Pre-conditions**: Mock repo. Permission `{Role: "STUDENT"}`.
- **Test data**:
  - Input: `CreateClass(ctx, CreateClassParams{Name: "10A", Grade: 10}, perm{Role:"STUDENT"})`
  - Expected output: `domainAuth.ErrUnauthorized`
  - Supporting data: none
- **Steps**:
  1. Call `uc.CreateClass` with STUDENT permission.
  2. Assert: `errors.Is(err, domainAuth.ErrUnauthorized)`.
- **Pass criteria**: ErrUnauthorized returned.
- **Fail indicators**: nil error; class created.

---

### TC-018: CreateClass — empty name — returns domain error
- **Type**: Unit
- **Covers**: AC-2 — domain validation in usecase
- **Pre-conditions**: Mock repo. Permission `{Role: "ADMIN"}`.
- **Test data**:
  - Input: `CreateClassParams{Name: "", Grade: 10}`
  - Expected output: error wrapping or equal to `class.ErrEmptyClassName`
  - Supporting data: none
- **Steps**:
  1. Call `uc.CreateClass` with empty name.
  2. Assert: error is non-nil.
  3. Assert: `errors.Is(err, class.ErrEmptyClassName)`.
- **Pass criteria**: `ErrEmptyClassName` propagated.
- **Fail indicators**: nil error; wrong error type.

---

### TC-019: CreateClass — grade <= 0 — returns domain error
- **Type**: Unit
- **Covers**: AC-2 — domain validation in usecase; boundary grade=0
- **Pre-conditions**: Mock repo. Permission `{Role: "ADMIN"}`.
- **Test data**:
  - Input: `CreateClassParams{Name: "10A", Grade: 0}`
  - Expected output: error wrapping `class.ErrInvalidGrade`
  - Supporting data: none
- **Steps**:
  1. Call `uc.CreateClass` with `Grade = 0`.
  2. Assert: `errors.Is(err, class.ErrInvalidGrade)`.
- **Pass criteria**: `ErrInvalidGrade` returned.
- **Fail indicators**: nil error; class created with grade 0.

---

### TC-020: CreateClass — grade = 1 — succeeds (boundary minimum valid grade)
- **Type**: Unit
- **Covers**: AC-2 — boundary value analysis: grade boundary min=1
- **Pre-conditions**: Mock repo. Permission `{Role: "ADMIN"}`.
- **Test data**:
  - Input: `CreateClassParams{Name: "1A", Grade: 1}`
  - Expected output: nil error; `ClassDTO.Grade == 1`
  - Supporting data: mock repo Create returns nil
- **Steps**:
  1. Call `uc.CreateClass` with `Grade = 1`.
  2. Assert: nil error.
  3. Assert: `result.Grade == 1`.
- **Pass criteria**: nil error; grade `1` accepted.
- **Fail indicators**: non-nil error; grade treated as invalid.

---

### TC-021: UpdateClass — ADMIN role, valid params — returns updated ClassDTO
- **Type**: Unit
- **Covers**: AC-4 — happy path
- **Pre-conditions**: Mock repo `FindByID` returns a seeded class `{ID: CLASS_10A_ID, Name: "10A", Grade: 10}`; `Update` returns nil. Permission `{Role: "ADMIN"}`.
- **Test data**:
  - Input: `UpdateClass(ctx, CLASS_10A_ID, UpdateClassParams{Name: "10A-Renamed", Grade: 10}, adminPerm)`
  - Expected output: `ClassDTO{Name: "10A-Renamed", Grade: 10}`, nil error
  - Supporting data: mock repo
- **Steps**:
  1. Seed mock repo with class `{ID: CLASS_10A_ID, Name: "10A", Grade: 10}`.
  2. Call `uc.UpdateClass` with `Name = "10A-Renamed"`.
  3. Assert: nil error.
  4. Assert: `result.Name == "10A-Renamed"`.
  5. Assert: `result.Grade == 10` (unchanged).
- **Pass criteria**: nil error; name updated; grade unchanged; `UpdatedAt` > `CreatedAt`.
- **Fail indicators**: non-nil error; old name retained; grade changed.

---

### TC-022: UpdateClass — ADMIN, update only grade — name unchanged
- **Type**: Unit
- **Covers**: AC-4 — partial update (grade only)
- **Pre-conditions**: Mock repo has class `{Name: "10A", Grade: 10}`. Permission ADMIN.
- **Test data**:
  - Input: `UpdateClassParams{Name: "10A", Grade: 11}` (name same, grade changed)
  - Expected output: `ClassDTO{Name: "10A", Grade: 11}`
  - Supporting data: mock repo
- **Steps**:
  1. Call `uc.UpdateClass` with same name `"10A"` but grade `11`.
  2. Assert: nil error.
  3. Assert: `result.Name == "10A"`.
  4. Assert: `result.Grade == 11`.
- **Pass criteria**: nil error; grade updated to `11`; name unchanged.
- **Fail indicators**: name changed; grade not updated.

---

### TC-023: UpdateClass — TEACHER role — returns ErrUnauthorized
- **Type**: Unit
- **Covers**: AC-2 — auth enforcement
- **Pre-conditions**: Mock repo. Permission `{Role: "TEACHER"}`.
- **Test data**:
  - Input: `UpdateClass(ctx, CLASS_10A_ID, UpdateClassParams{Name: "10A", Grade: 10}, teacherPerm)`
  - Expected output: `domainAuth.ErrUnauthorized`
  - Supporting data: none
- **Steps**:
  1. Call `uc.UpdateClass` with TEACHER permission.
  2. Assert: `errors.Is(err, domainAuth.ErrUnauthorized)`.
- **Pass criteria**: ErrUnauthorized returned; repo not called.
- **Fail indicators**: nil error; class updated.

---

### TC-024: UpdateClass — STUDENT role — returns ErrUnauthorized
- **Type**: Unit
- **Covers**: AC-2 — auth enforcement
- **Pre-conditions**: Mock repo. Permission `{Role: "STUDENT"}`.
- **Test data**:
  - Input: `UpdateClass(ctx, CLASS_10A_ID, UpdateClassParams{Name: "10A", Grade: 10}, studentPerm)`
  - Expected output: `domainAuth.ErrUnauthorized`
  - Supporting data: none
- **Steps**:
  1. Call `uc.UpdateClass` with STUDENT permission.
  2. Assert: `errors.Is(err, domainAuth.ErrUnauthorized)`.
- **Pass criteria**: ErrUnauthorized returned.
- **Fail indicators**: nil error.

---

### TC-025: UpdateClass — class ID not found — returns ErrUnauthorized (anti-enumeration)
- **Type**: Unit
- **Covers**: AC-2 — not-found mapped to ErrUnauthorized (anti-enumeration)
- **Pre-conditions**: Mock repo `FindByID` returns `class.ErrClassNotFound`. Permission ADMIN.
- **Test data**:
  - Input: `UpdateClass(ctx, "non-existent-uuid", UpdateClassParams{Name: "10A", Grade: 10}, adminPerm)`
  - Expected output: `domainAuth.ErrUnauthorized` (NOT 404)
  - Supporting data: mock repo returning ErrClassNotFound
- **Steps**:
  1. Configure mock repo `FindByID` to return `classDomain.ErrClassNotFound`.
  2. Call `uc.UpdateClass`.
  3. Assert: `errors.Is(err, domainAuth.ErrUnauthorized)`.
  4. Assert: `errors.Is(err, classDomain.ErrClassNotFound)` is false (original error not leaked).
- **Pass criteria**: ErrUnauthorized is the returned error, not ErrClassNotFound.
- **Fail indicators**: ErrClassNotFound leaked; nil error; 404-equivalent response.

---

### TC-026: UpdateClass — invalid UUID format for classId — returns ErrUnauthorized
- **Type**: Unit
- **Covers**: AC-2 — malformed ID handling
- **Pre-conditions**: Mock repo. Permission ADMIN.
- **Test data**:
  - Input: classId `"not-a-uuid"`, valid params
  - Expected output: `domainAuth.ErrUnauthorized`
  - Supporting data: none
- **Steps**:
  1. Call `uc.UpdateClass(ctx, "not-a-uuid", params, adminPerm)`.
  2. Assert: `errors.Is(err, domainAuth.ErrUnauthorized)`.
- **Pass criteria**: ErrUnauthorized returned; no panic.
- **Fail indicators**: ErrInvalidClassID leaked; panic.

---

### Group E — Happy Path HTTP Tests (Integration)

File: `internal/delivery/http/handlers/integration_test.go`

Router setup for POST/PUT tests:
- Echo instance with `JWTAuth` middleware on `/api/v1`
- `rules.NewValidator(db)` wired into Echo as the validator (`e.Validator = validator`)
- Real SQLite in-memory DB with `classes` table migrated
- Mock usecase for usecase-layer results (usecase is mocked; validator uses real DB for `unique_in`)

---

### TC-060: POST /api/v1/classes — ADMIN, valid body — returns 201 with class object
- **Type**: Integration
- **Covers**: AC-3 — integration test; AC-4 — happy path
- **Pre-conditions**: DB empty. ADMIN JWT. Mock usecase `CreateClass` returns a `ClassDTO{ID: CLASS_10A_ID, Name: "10A", Grade: 10}`.
- **Test data**:
  - Input: `POST /api/v1/classes`, `Authorization: Bearer ADMIN_TOKEN`, body `{"name":"10A","grade":10}`
  - Expected output: HTTP 201; body `{"id":"<uuid>","name":"10A","grade":10,"createdAt":"...","updatedAt":"..."}`
  - Supporting data: empty classes table (unique_in passes); mock usecase returning ClassDTO
- **Steps**:
  1. Set up router with real validator (empty DB) and mock usecase.
  2. Send `POST /api/v1/classes` with valid JSON body and ADMIN bearer token.
  3. Assert: status code is `201`.
  4. Assert: response body parses as `Class`; `name == "10A"`, `grade == 10`, `id` is non-empty.
- **Pass criteria**: HTTP 201; body contains correct name, grade, and non-empty UUID id.
- **Fail indicators**: status 422 (validation spuriously failed); status 403/401; 500; missing `id` field.

---

### TC-061: PUT /api/v1/classes/:classId — ADMIN, rename to new unique name — returns 200
- **Type**: Integration
- **Covers**: AC-3 — integration test; AC-4 — happy path
- **Pre-conditions**: DB contains class `{ID: CLASS_10A_ID, Name: "10A", Grade: 10}`. ADMIN JWT. Mock usecase `UpdateClass` returns updated DTO `{Name: "10A-Renamed", Grade: 10}`.
- **Test data**:
  - Input: `PUT /api/v1/classes/CLASS_10A_ID`, body `{"name":"10A-Renamed","grade":10}`, ADMIN token
  - Expected output: HTTP 200; body `{"name":"10A-Renamed","grade":10}`
  - Supporting data: seed classes table with `{ID: CLASS_10A_ID, Name: "10A"}`; ExcludeIDKey set to CLASS_10A_ID before validation
- **Steps**:
  1. Seed DB with one class.
  2. Send PUT with new name `"10A-Renamed"`.
  3. Assert: status `200`.
  4. Assert: response body `name == "10A-Renamed"`.
- **Pass criteria**: HTTP 200; name in response matches input.
- **Fail indicators**: 422 (rename blocked as duplicate of non-existent row); 401; 500.

---

### TC-062: PUT /api/v1/classes/:classId — ADMIN, update with same (own) name — returns 200 (self-exclusion)
- **Type**: Integration
- **Covers**: AC-3 — integration test; self-exclusion (ExcludeIDKey) correctness
- **Pre-conditions**: DB contains class `{ID: CLASS_10A_ID, Name: "10A", Grade: 10}`. ADMIN JWT.
- **Test data**:
  - Input: `PUT /api/v1/classes/CLASS_10A_ID`, body `{"name":"10A","grade":11}` (same name, new grade)
  - Expected output: HTTP 200
  - Supporting data: seed class with `name = "10A"` and `id = CLASS_10A_ID`; handler passes `ExcludeIDKey = CLASS_10A_ID` into context before calling `ValidateCtx`
- **Steps**:
  1. Seed DB with `{ID: CLASS_10A_ID, Name: "10A"}`.
  2. Send PUT to `CLASS_10A_ID` with same name `"10A"` and new grade `11`.
  3. Assert: status `200`.
- **Pass criteria**: HTTP 200 — own name is not treated as a duplicate.
- **Fail indicators**: 422 with `fields.name` set (own name incorrectly rejected).

---

### TC-063: POST /api/v1/classes — no Authorization header — returns 401
- **Type**: Integration
- **Covers**: AC-3 — integration test; auth failure
- **Pre-conditions**: Any DB state.
- **Test data**:
  - Input: `POST /api/v1/classes`, no `Authorization` header, body `{"name":"10A","grade":10}`
  - Expected output: HTTP 401; body `{"error":"unauthorized","message":"Missing or invalid authorization header"}`
  - Supporting data: none
- **Steps**:
  1. Send POST without Authorization header.
  2. Assert: status `401`.
  3. Assert: response body `error == "unauthorized"`.
- **Pass criteria**: HTTP 401; correct error body.
- **Fail indicators**: 400; 403; 500; request reaches handler.

---

### TC-064: POST /api/v1/classes — TEACHER role JWT — returns 403
- **Type**: Integration
- **Covers**: AC-3 — integration test; role-based rejection at usecase
- **Pre-conditions**: Mock JWT Verify returns `{Role: "TEACHER"}`. Mock usecase `CreateClass` returns `domainAuth.ErrUnauthorized`.
- **Test data**:
  - Input: `POST /api/v1/classes`, `Authorization: Bearer TEACHER_TOKEN`, body `{"name":"10A","grade":10}`
  - Expected output: HTTP 403; body `{"error":"forbidden","message":"..."}`
  - Supporting data: mock usecase returning ErrUnauthorized
- **Steps**:
  1. Set up router; mock usecase returns ErrUnauthorized for CreateClass.
  2. Send POST with TEACHER token.
  3. Assert: status `403`.
- **Pass criteria**: HTTP 403.
- **Fail indicators**: 201 (class created); 401 (middleware rejecting valid TEACHER token); 500.

---

### TC-065: POST /api/v1/classes — STUDENT role JWT — returns 403
- **Type**: Integration
- **Covers**: AC-3 — integration test; student cannot create
- **Pre-conditions**: Mock JWT Verify returns `{Role: "STUDENT"}`. Mock usecase returns `domainAuth.ErrUnauthorized`.
- **Test data**:
  - Input: `POST /api/v1/classes`, `Authorization: Bearer STUDENT_TOKEN`, body `{"name":"10A","grade":10}`
  - Expected output: HTTP 403
  - Supporting data: none
- **Steps**:
  1. Send POST with STUDENT token.
  2. Assert: status `403`.
- **Pass criteria**: HTTP 403.
- **Fail indicators**: 201; 401; 500.

---

### TC-066: POST /api/v1/classes — duplicate name — returns 422 with fields map
- **Type**: Integration
- **Covers**: AC-3 — integration test; duplicate name on create
- **Pre-conditions**: DB contains a class with `name = "10A"`. ADMIN JWT. Validator wired to the real DB.
- **Test data**:
  - Input: `POST /api/v1/classes`, ADMIN token, body `{"name":"10A","grade":10}`
  - Expected output: HTTP 422; body `{"error":"validation_error","message":"...","fields":{"name":"..."}}`
  - Supporting data: seed one class row with `name = "10A"` in the real in-memory DB
- **Steps**:
  1. Seed DB with `{Name: "10A"}`.
  2. Send POST with `name = "10A"`.
  3. Assert: status `422`.
  4. Assert: response body parses as `ValidationErrorResponse`; `fields["name"]` is non-empty.
- **Pass criteria**: HTTP 422; `fields` map present with `name` key populated.
- **Fail indicators**: 201 (duplicate accepted); 500; 422 with no `fields` map; `fields` missing the `name` key.

---

### TC-067: PUT /api/v1/classes/:classId — duplicate name of a DIFFERENT class — returns 422 with fields map
- **Type**: Integration
- **Covers**: AC-3 — integration test; duplicate name on update targeting another class
- **Pre-conditions**: DB contains `{ID: CLASS_10A_ID, Name: "10A"}` AND `{ID: CLASS_11B_ID, Name: "11B"}`. ADMIN JWT.
- **Test data**:
  - Input: `PUT /api/v1/classes/CLASS_10A_ID`, body `{"name":"11B","grade":10}` (trying to rename 10A to 11B which already exists)
  - Expected output: HTTP 422; body contains `fields.name`
  - Supporting data: seed both classes; ExcludeIDKey = CLASS_10A_ID in the validation context (so 10A itself is excluded, but 11B is NOT)
- **Steps**:
  1. Seed both classes.
  2. Send PUT to CLASS_10A_ID with `name = "11B"`.
  3. Assert: status `422`.
  4. Assert: `fields["name"]` is non-empty.
- **Pass criteria**: HTTP 422; `fields.name` populated.
- **Fail indicators**: 200 (rename accepted despite collision); 500; missing `fields` key.

---

### Group F — Error & Edge Case HTTP Tests

---

### TC-068: POST /api/v1/classes — missing `name` field — returns 422 (required validation)
- **Type**: Integration
- **Covers**: AC-2 — missing required field
- **Pre-conditions**: ADMIN JWT. Empty DB.
- **Test data**:
  - Input: `POST /api/v1/classes`, ADMIN token, body `{"grade":10}` (name absent)
  - Expected output: HTTP 422; `fields["name"]` present
  - Supporting data: none
- **Steps**:
  1. Send POST with body missing `name`.
  2. Assert: status `422`.
  3. Assert: `fields["name"]` is non-empty.
- **Pass criteria**: HTTP 422 with `fields.name`.
- **Fail indicators**: 400 (wrong status); 500; 201; fields map missing.

---

### TC-069: POST /api/v1/classes — missing `grade` field — returns 422 (required validation)
- **Type**: Integration
- **Covers**: AC-2 — missing required field
- **Pre-conditions**: ADMIN JWT. Empty DB.
- **Test data**:
  - Input: `POST /api/v1/classes`, body `{"name":"10A"}` (grade absent / zero)
  - Expected output: HTTP 422; `fields["grade"]` present
  - Supporting data: none
- **Steps**:
  1. Send POST with body missing `grade`.
  2. Assert: status `422`.
  3. Assert: `fields["grade"]` is non-empty.
- **Pass criteria**: HTTP 422 with `fields.grade`.
- **Fail indicators**: 201; 400; `fields` map missing `grade` key.

---

### TC-070: POST /api/v1/classes — grade = 0 — returns 422 (min=1 validation)
- **Type**: Integration
- **Covers**: AC-2 — boundary value grade=0
- **Pre-conditions**: ADMIN JWT. Empty DB.
- **Test data**:
  - Input: `POST /api/v1/classes`, body `{"name":"10A","grade":0}`
  - Expected output: HTTP 422; `fields["grade"]` present
  - Supporting data: none
- **Steps**:
  1. Send POST with `grade = 0`.
  2. Assert: status `422`.
  3. Assert: `fields["grade"]` is non-empty.
- **Pass criteria**: HTTP 422 with grade validation error.
- **Fail indicators**: 201 (grade 0 accepted); missing fields key.

---

### TC-071: POST /api/v1/classes — grade = -1 — returns 422 (min=1 validation)
- **Type**: Integration
- **Covers**: AC-2 — boundary value grade < 0
- **Pre-conditions**: ADMIN JWT. Empty DB.
- **Test data**:
  - Input: `POST /api/v1/classes`, body `{"name":"10A","grade":-1}`
  - Expected output: HTTP 422; `fields["grade"]` present
  - Supporting data: none
- **Steps**:
  1. Send POST with `grade = -1`.
  2. Assert: status `422`.
  3. Assert: `fields["grade"]` is non-empty.
- **Pass criteria**: HTTP 422 with grade validation error.
- **Fail indicators**: 201; missing fields.

---

### TC-072: POST /api/v1/classes — grade = 1 — returns 201 (boundary minimum valid)
- **Type**: Integration
- **Covers**: AC-2 — boundary value grade=1 is accepted
- **Pre-conditions**: ADMIN JWT. Empty DB. Mock usecase returns a DTO with grade=1.
- **Test data**:
  - Input: `POST /api/v1/classes`, body `{"name":"1A","grade":1}`
  - Expected output: HTTP 201
  - Supporting data: mock usecase `CreateClass` returns `ClassDTO{Name:"1A", Grade:1}`
- **Steps**:
  1. Send POST with `grade = 1`.
  2. Assert: status `201`.
- **Pass criteria**: HTTP 201.
- **Fail indicators**: 422 (grade 1 incorrectly rejected).

---

### TC-073: POST /api/v1/classes — whitespace-only name — returns 422 (required after trim)
- **Type**: Integration
- **Covers**: AC-2 — whitespace-only name; NewName trims whitespace
- **Pre-conditions**: ADMIN JWT. Empty DB.
- **Test data**:
  - Input: `POST /api/v1/classes`, body `{"name":"   ","grade":10}`
  - Expected output: HTTP 422; `fields["name"]` present
  - Supporting data: none
- **Steps**:
  1. Send POST with `name = "   "` (spaces only).
  2. Assert: status `422`.
  3. Assert: `fields["name"]` is non-empty.
- **Pass criteria**: HTTP 422 with name validation error.
- **Fail indicators**: 201 (whitespace name accepted); `fields` missing `name`.

---

### TC-074: PUT /api/v1/classes/:classId — nonexistent classId — returns 401 (anti-enumeration)
- **Type**: Integration
- **Covers**: AC-2 — nonexistent class returns 401 not 404; anti-enumeration
- **Pre-conditions**: DB is empty (or does not contain the target ID). ADMIN JWT. Mock usecase returns `domainAuth.ErrUnauthorized` for this ID.
- **Test data**:
  - Input: `PUT /api/v1/classes/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb`, ADMIN token, body `{"name":"10A","grade":10}`
  - Expected output: HTTP 401; body `{"error":"unauthorized","message":"..."}`
  - Supporting data: mock usecase UpdateClass returns ErrUnauthorized
- **Steps**:
  1. Send PUT to a UUID that does not exist.
  2. Assert: status `401`.
  3. Assert: status is NOT `404`.
- **Pass criteria**: HTTP 401.
- **Fail indicators**: 404 (class not found exposed to caller); 200; 500.

---

### TC-075: PUT /api/v1/classes/:classId — malformed UUID in path — returns 401 or 400
- **Type**: Integration
- **Covers**: AC-2 — malformed ID in URL path
- **Pre-conditions**: ADMIN JWT.
- **Test data**:
  - Input: `PUT /api/v1/classes/not-a-uuid`, ADMIN token, body `{"name":"10A","grade":10}`
  - Expected output: HTTP 401 (usecase maps `ErrInvalidClassID` to `ErrUnauthorized`) OR HTTP 400 (router-level binding rejects)
  - Supporting data: none
- **Steps**:
  1. Send PUT to `not-a-uuid` path.
  2. Assert: status is either `400` or `401` (both acceptable; document which is chosen by the implementation).
  3. Assert: status is NOT `200`, `201`, or `500`.
- **Pass criteria**: status is `400` or `401`.
- **Fail indicators**: 200; 500; panic.

---

### TC-076: POST /api/v1/classes — malformed JSON body — returns 400
- **Type**: Integration
- **Covers**: AC-2 — invalid body format
- **Pre-conditions**: ADMIN JWT.
- **Test data**:
  - Input: `POST /api/v1/classes`, ADMIN token, body `{"name":}` (invalid JSON)
  - Expected output: HTTP 400
  - Supporting data: none
- **Steps**:
  1. Send POST with invalid JSON.
  2. Assert: status `400`.
- **Pass criteria**: HTTP 400.
- **Fail indicators**: 500; 201; panic.

---

### TC-077: Idempotency — POST /api/v1/classes called twice with same name — second call returns 422
- **Type**: Integration
- **Covers**: AC-2 — idempotency / duplicate prevention
- **Pre-conditions**: DB starts empty. ADMIN JWT. First call succeeds and seeds the DB row directly (or mock usecase creates it).
- **Test data**:
  - First call: `POST /api/v1/classes`, body `{"name":"10A","grade":10}` → expect 201; DB now has the row.
  - Second call: same body → expect 422 (unique_in fires)
  - Supporting data: after first call, seed the real DB with `{Name: "10A"}` to simulate the created row
- **Steps**:
  1. Send first POST — assert status `201`.
  2. (DB now has `name = "10A"`.)
  3. Send second POST with identical body.
  4. Assert: status `422`.
  5. Assert: exactly one row with `name = "10A"` exists in DB.
- **Pass criteria**: second call returns `422`; only one row in DB.
- **Fail indicators**: second call returns `201` (duplicate inserted); two rows in DB.

---

## Test Data Catalogue

| Name | Value | Used By |
|------|-------|---------|
| `CLASS_10A_ID` | `"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"` | TC-007, TC-061, TC-062, TC-067, TC-077 |
| `CLASS_11B_ID` | `"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"` | TC-067 |
| `NONEXISTENT_ID` | `"cccccccc-cccc-cccc-cccc-cccccccccccc"` | TC-074 |
| `ADMIN_TOKEN` | any string; mock JWT Verify returns `{UserID:"admin-1", Role:"ADMIN"}` | TC-060–TC-067, TC-068–TC-077 |
| `TEACHER_TOKEN` | any string; mock JWT Verify returns `{UserID:"teacher-1", Role:"TEACHER"}` | TC-064 |
| `STUDENT_TOKEN` | any string; mock JWT Verify returns `{UserID:"student-1", Role:"STUDENT"}` | TC-065 |
| Valid class body | `{"name":"10A","grade":10}` | TC-060, TC-066, TC-073, TC-077 |
| Duplicate name body | `{"name":"10A","grade":10}` after seeding `name="10A"` | TC-066, TC-077 |
| Whitespace name body | `{"name":"   ","grade":10}` | TC-073 |
| Zero grade body | `{"name":"10A","grade":0}` | TC-070 |
| Negative grade body | `{"name":"10A","grade":-1}` | TC-071 |
| Min valid grade body | `{"name":"1A","grade":1}` | TC-072 |
| Missing name body | `{"grade":10}` | TC-068 |
| Missing grade body | `{"name":"10A"}` | TC-069 |
| Malformed JSON | `{"name":}` | TC-076 |

---

## Out of Scope

- **GET /api/v1/classes and GET /api/v1/classes/:classId** — covered by GS-002 test plan.
- **Other domains** (users, subjects, auth) — no mutations in scope.
- **Database migration scripts** — tested implicitly via GORM AutoMigrate in integration setup.
- **OpenAPI spec codegen output** (`api.gen.go`) — generated artefact; not tested directly.
- **Load / performance testing** — not required by any AC.
- **TLS / HTTPS** — infrastructure concern, out of scope for this task.
- **Concurrent duplicate-create race condition** — noted as a risk (see below); out of scope for unit/integration coverage, but noted for future DB-constraint-level test.

---

## Risks Flagged

1. **`ValidationErrorResponse` shape not yet committed to spec.** The context document specifies `{Error, Message, Fields map[string]string}` but `internal/delivery/http/types.go` only contains `ErrorResponse{Error, Message}`. The tester must confirm the exact struct name and JSON field names (`"fields"` vs `"validation_errors"`) once the developer lands the type. TC-066, TC-067, TC-068, TC-069, TC-070, TC-071, TC-073 all depend on the `fields` key.

2. **`ExcludeIDKey` type for context.WithValue.** The context document states the key is `rules.ExcludeIDKey`. If this is an unexported or typed key, test code in `_test` packages must import the `rules` package to reference it. Confirm the key is exported.

3. **`IClassUseCase` interface extension.** The existing `IClassUseCase` interface (as read from source) only has `GetClasses` and `GetClassByID`. The developer must add `CreateClass` and `UpdateClass` before the integration test mock can implement the full interface. The test plan assumes those methods will be added; the tester should verify the updated interface signature before implementing test doubles.

4. **PUT endpoint path parameter name.** The current `classes-by-id.yaml` path item uses `classId` as the path parameter name. The handler is registered as `/classes/:classId` in the integration tests. The new PUT endpoint must use the same parameter name or TC-062, TC-067, TC-074 will target the wrong path.

5. **Concurrent duplicate-create.** If two POST requests for the same name arrive simultaneously, the `unique_in` validator passes for both (neither row exists yet), but the DB unique constraint on the `name` column catches the second insert. The test plan does not include a concurrency test; the tester should verify the DB constraint exists and add a note about the 500 vs 422 discrepancy if the DB error is not mapped to a 422.
