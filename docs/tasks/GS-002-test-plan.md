# Test Plan: GS-002 — RBAC Authorization Middleware and Class Endpoints

## Scope

This plan covers all new and modified behaviour introduced by GS-002:

1. **JWT middleware** (`internal/delivery/http/middleware/auth.go`) — token extraction, validation, and `ContextPermission` injection.
2. **Class domain** (`internal/domain/class/`) — `Class` entity, value objects (`ID`, `Name`, `Grade`), `IClassRepository` interface, `ErrClassNotFound` and `ErrUnauthorized` sentinels.
3. **Class usecase** (`internal/usecase/class/`) — role-scoped `GetClasses` and `GetClassByID`.
4. **Class handler** (`internal/delivery/http/handlers/class.go`) — HTTP binding, error-to-status mapping.
5. **JWT service** (`internal/infrastructure/auth/jwt-service.go`) — token signing and verification.
6. **`POST /auth/google` response update** — `AuthResponseDTO` and `api.AuthResponse` now include a `token` field.
7. **Casbin policy additions** — new `p` rows for TEACHER and STUDENT on `/api/v1/classes` and `/api/v1/classes/:class_id`.

**Out of scope:** changes to user management endpoints, Google OAuth flow itself (covered in GS-001), slot editing, subject endpoints, FE code, and any role-promotion flows. Infrastructure wiring (`wire.go` / `wire_gen.go`) is explicitly excluded from test cases — correctness is verified transitively through integration tests.

---

## Test Environment

- **Go version**: per `go.mod` (1.22+)
- **Database**: in-process SQLite (`:memory:` or temp file) via GORM
- **JWT signing key**: HS256 or RS256 as chosen by the implementer; unit tests use a known test secret/key pair
- **External services**: none called by class or middleware code; Google tokeninfo is not invoked in this task
- **Required env vars**: `JWT_SECRET` (or equivalent key env var as implemented), `ADMIN_EMAIL`, `ADMIN_PASSWORD`
- **Test packages**:
  - `internal/domain/class/...` (unit)
  - `internal/usecase/class/...` (unit — mocked repository)
  - `internal/delivery/http/middleware/...` (unit — mocked JWT verifier)
  - `internal/infrastructure/auth/...` (unit — JWT service sign/verify)
  - `internal/delivery/http/handlers/...` (integration — in-process Echo, mocked usecase)
  - E2E tests run the full Echo server with an in-memory SQLite DB

---

## Acceptance Criteria Coverage Matrix

| AC | Description | Test Cases |
|----|-------------|------------|
| AC-1 | Follow SOLID principles | TC-001–TC-004 (interfaces are injected as dependencies; each component is independently testable) |
| AC-2 | Reusable middleware (can be applied to any Echo route group) | TC-005–TC-011, TC-046–TC-049 |
| AC-3 | Pass all unit and integration tests for the changes | All TC-001–TC-058 |
| AC-4 | Test coverage 100% | Coverage noted per section; every branch in middleware, usecase, and handler is exercised |

---

## Test Cases

---

### Happy Paths

---

#### Group A — JWT Service (sign and verify)

---

### TC-001: Sign a token and verify it round-trips successfully
- **Type**: Unit
- **Covers**: AC-1 — JWT service produces a verifiable token; AC-4 — happy path of `Sign` + `Verify`
- **Pre-conditions**: JWT service is initialised with a known test secret
- **Test data**:
  - Input to Sign: `userID = "550e8400-e29b-41d4-a716-446655440001"`, `role = "TEACHER"`
  - Expected output of Verify: `ContextPermission{UserID: "550e8400-e29b-41d4-a716-446655440001", Role: "TEACHER"}`, no error
  - Supporting data: `JWT_SECRET = "test-secret-key-32-bytes-long!!"` (or RSA key pair stored in test fixtures)
- **Steps**:
  1. Call `jwtService.Sign(userID, role)` → capture `tokenString`
  2. Call `jwtService.Verify(tokenString)` → capture `permission, err`
  3. Assert: `err == nil`
  4. Assert: `permission.UserID == "550e8400-e29b-41d4-a716-446655440001"`
  5. Assert: `permission.Role == "TEACHER"`
- **Pass criteria**: `err` is nil AND both fields match exactly
- **Fail indicators**: error returned, wrong userID, wrong role, zero-value struct

---

### TC-002: Token has 24-hour expiry
- **Type**: Unit
- **Covers**: AC-4 — expiry claim is set
- **Pre-conditions**: JWT service initialised with test secret; system clock is real
- **Test data**:
  - Input: Sign with any valid `userID` and `role`
  - Expected: JWT `exp` claim is within `[now+23h59m, now+24h1m]`
  - Supporting data: parse raw claims after signing
- **Steps**:
  1. Record `beforeSign = time.Now()`
  2. Call `jwtService.Sign(userID, role)` → `tokenString`
  3. Parse `tokenString` without verification to read raw claims
  4. Record `afterSign = time.Now()`
  5. Assert: `claims.ExpiresAt` is between `beforeSign + 23h59m` and `afterSign + 24h1m`
- **Pass criteria**: `exp` claim falls within the 2-minute tolerance window around 24 hours from sign time
- **Fail indicators**: `exp` missing, `exp` is more than 1 hour away from expected, token never expires

---

#### Group B — JWT Middleware Unit Tests

---

### TC-003: Valid Bearer token injects ContextPermission and calls next handler
- **Type**: Unit
- **Covers**: AC-2 — reusable middleware normal path; AC-4 — happy path branch
- **Pre-conditions**: Middleware initialised with a mock JWT verifier that returns `ContextPermission{UserID: "u-001", Role: "STUDENT"}` for token `"valid.jwt.token"`
- **Test data**:
  - Input: HTTP request with header `Authorization: Bearer valid.jwt.token`
  - Expected output: next handler called; `ctx.Get("permission")` returns `ContextPermission{UserID: "u-001", Role: "STUDENT"}`; HTTP status not set by middleware
  - Supporting data: minimal Echo context; next handler captures the permission value and returns 200
- **Steps**:
  1. Construct Echo context with header `Authorization: Bearer valid.jwt.token`
  2. Register a next handler that reads `ctx.Get("permission")` and stores it
  3. Call the middleware with the Echo context
  4. Assert: next handler was invoked (called flag is true)
  5. Assert: `ctx.Get("permission")` equals `ContextPermission{UserID: "u-001", Role: "STUDENT"}`
  6. Assert: middleware did not write an error response
- **Pass criteria**: next was called AND injected permission matches exactly
- **Fail indicators**: middleware returned 401, next was not called, permission is nil or has wrong fields

---

### TC-004: Missing Authorization header returns 401
- **Type**: Unit
- **Covers**: AC-2, AC-4 — missing token error branch
- **Pre-conditions**: Middleware initialised with any JWT verifier
- **Test data**:
  - Input: HTTP request with no `Authorization` header
  - Expected output: HTTP 401, body `{"message":"missing or malformed authorization header"}` (or equivalent message per implementation), next handler NOT called
- **Steps**:
  1. Construct Echo context with no Authorization header
  2. Call the middleware
  3. Assert: HTTP status code is 401
  4. Assert: response body contains an error field indicating missing/malformed token
  5. Assert: next handler was not invoked
- **Pass criteria**: status 401 AND next not called
- **Fail indicators**: next was called, status is not 401, body is empty

---

### TC-005: Authorization header present but no "Bearer" prefix returns 401
- **Type**: Unit
- **Covers**: AC-2, AC-4 — malformed header format
- **Pre-conditions**: Middleware initialised with any JWT verifier
- **Test data**:
  - Input: `Authorization: sometoken.without.prefix` (no "Bearer " prefix)
  - Expected output: HTTP 401, next not called
- **Steps**:
  1. Set `Authorization` header to `"sometoken.without.prefix"`
  2. Call middleware
  3. Assert: status 401
  4. Assert: next not called
- **Pass criteria**: status 401 AND next not called
- **Fail indicators**: status not 401, next called

---

### TC-006: Bearer token with invalid signature returns 401
- **Type**: Unit
- **Covers**: AC-2, AC-4 — invalid JWT error branch
- **Pre-conditions**: Real JWT verifier with test secret; token signed with a DIFFERENT secret
- **Test data**:
  - Input: `Authorization: Bearer <token signed with wrong-secret>`
  - Expected output: HTTP 401, next not called
- **Steps**:
  1. Sign a token with a different secret than the one the middleware uses
  2. Set `Authorization: Bearer <that token>`
  3. Call middleware
  4. Assert: status 401
  5. Assert: next not called
- **Pass criteria**: status 401 AND next not called
- **Fail indicators**: middleware passes the bad token through, status not 401

---

### TC-007: Expired JWT returns 401
- **Type**: Unit
- **Covers**: AC-2, AC-4 — expired token error branch
- **Pre-conditions**: Real JWT service; token signed with `exp = time.Now() - 1 second`
- **Test data**:
  - Input: `Authorization: Bearer <already-expired-token>`
  - Expected output: HTTP 401, next not called
- **Steps**:
  1. Construct a JWT with `exp` set to 1 second in the past using the correct secret
  2. Set `Authorization: Bearer <that token>`
  3. Call middleware
  4. Assert: status 401
  5. Assert: next not called
- **Pass criteria**: status 401 AND next not called
- **Fail indicators**: expired token accepted, next called

---

### TC-008: Token with unknown role still passes middleware (role enforcement is usecase concern)
- **Type**: Unit
- **Covers**: AC-2, AC-4 — unknown role is NOT rejected at middleware level; rejection happens at usecase
- **Pre-conditions**: Real JWT verifier with test secret; token contains `role = "UNKNOWN"`
- **Test data**:
  - Input: `Authorization: Bearer <token with role=UNKNOWN, signed correctly>`
  - Expected output: next handler IS called; `ctx.Get("permission").Role == "UNKNOWN"`
- **Steps**:
  1. Sign a valid JWT containing `role = "UNKNOWN"` using the correct secret
  2. Set `Authorization: Bearer <that token>`
  3. Call middleware
  4. Assert: next handler IS called
  5. Assert: `ctx.Get("permission")` is `ContextPermission{UserID: <id>, Role: "UNKNOWN"}`
- **Pass criteria**: next was called AND Role field is "UNKNOWN"
- **Fail indicators**: middleware rejects the token with 401 before the usecase sees it

---

### TC-009: Token with empty userID claim returns 401
- **Type**: Unit
- **Covers**: AC-4 — boundary: missing required claim
- **Pre-conditions**: JWT signed correctly but `sub`/userID claim is empty string
- **Test data**:
  - Input: `Authorization: Bearer <token with empty userID>`
  - Expected output: HTTP 401, next not called
- **Steps**:
  1. Construct JWT with `userID = ""` signed correctly
  2. Call middleware
  3. Assert: status 401
  4. Assert: next not called
- **Pass criteria**: status 401 AND next not called
- **Fail indicators**: empty userID injected into context, next called

---

### TC-010: Token with empty role claim returns 401
- **Type**: Unit
- **Covers**: AC-4 — boundary: missing required claim
- **Pre-conditions**: JWT signed correctly but `role` claim is empty string
- **Test data**:
  - Input: `Authorization: Bearer <token with empty role claim>`
  - Expected output: HTTP 401, next not called
- **Steps**:
  1. Construct JWT with `role = ""` signed correctly
  2. Call middleware
  3. Assert: status 401
  4. Assert: next not called
- **Pass criteria**: status 401 AND next not called
- **Fail indicators**: empty role injected into context, next called

---

### TC-011: Authorization header with "Bearer " prefix but empty token string returns 401
- **Type**: Unit
- **Covers**: AC-2, AC-4 — boundary: "Bearer " followed by nothing
- **Pre-conditions**: Middleware initialised with any JWT verifier
- **Test data**:
  - Input: `Authorization: Bearer ` (trailing space, no token)
  - Expected output: HTTP 401, next not called
- **Steps**:
  1. Set `Authorization` to `"Bearer "` (seven chars, nothing after)
  2. Call middleware
  3. Assert: status 401
  4. Assert: next not called
- **Pass criteria**: status 401 AND next not called
- **Fail indicators**: panic, next called, status other than 401

---

#### Group C — Class Domain Unit Tests

---

### TC-012: Class entity is constructed with valid Name and Grade
- **Type**: Unit
- **Covers**: AC-1, AC-4 — domain value object happy path
- **Pre-conditions**: None
- **Test data**:
  - Input: `name = "1A1"`, `grade = 1`
  - Expected: `class.Name().String() == "1A1"`, `class.Grade() == 1`, `class.ID()` is a valid non-zero UUID
- **Steps**:
  1. Call `class.NewClass(name, grade)` (or equivalent constructor)
  2. Assert: no error
  3. Assert: `Name()`, `Grade()`, and `ID()` return expected values
- **Pass criteria**: entity constructed without error, all getters return correct values
- **Fail indicators**: panic, error returned, zero-value ID

---

### TC-013: Class Name value object rejects empty string
- **Type**: Unit
- **Covers**: AC-4 — boundary: empty name
- **Pre-conditions**: None
- **Test data**:
  - Input: `name = ""`
  - Expected: `ErrEmptyClassName` (or equivalent) returned, no entity created
- **Steps**:
  1. Call `class.NewName("")`
  2. Assert: error returned
  3. Assert: error wraps or equals the expected sentinel
- **Pass criteria**: non-nil error whose message or type identifies an invalid/empty name
- **Fail indicators**: nil error, entity created with empty name

---

### TC-014: Class ID ParseID rejects non-UUID string
- **Type**: Unit
- **Covers**: AC-4 — boundary: invalid ID format
- **Pre-conditions**: None
- **Test data**:
  - Input: `"not-a-uuid"`
  - Expected: domain error returned (same pattern as `user.ErrInvalidID`)
- **Steps**:
  1. Call `classID.ParseID("not-a-uuid")`
  2. Assert: error returned
- **Pass criteria**: non-nil error
- **Fail indicators**: nil error, panic

---

### TC-015: Class Grade rejects zero value
- **Type**: Unit
- **Covers**: AC-4 — boundary: grade = 0
- **Pre-conditions**: None
- **Test data**:
  - Input: `grade = 0`
  - Expected: error returned (grade must be a positive integer)
- **Steps**:
  1. Call `class.NewGrade(0)` (or equivalent)
  2. Assert: error returned
- **Pass criteria**: non-nil error
- **Fail indicators**: nil error, grade 0 accepted

---

#### Group D — Class Usecase Unit Tests: GetClasses

---

### TC-016: ADMIN gets all classes
- **Type**: Unit
- **Covers**: AC-3, AC-4 — ADMIN GetClasses happy path
- **Pre-conditions**: Mock `IClassRepository.FindAll` returns `[classA, classB, classC]`
- **Test data**:
  - Input: `permission = ContextPermission{UserID: "admin-u-001", Role: "ADMIN"}`
  - Expected output: `[]ClassDTO` with all three classes, `err == nil`
  - Supporting data: mock repository seeded with 3 class records
- **Steps**:
  1. Construct `classUseCase` with mock repository
  2. Call `GetClasses(ctx, ContextPermission{UserID: "admin-u-001", Role: "ADMIN"})`
  3. Assert: no error
  4. Assert: returned slice length is 3
  5. Assert: each DTO contains the expected `ID`, `Name`, `Grade` values
- **Pass criteria**: `err == nil` AND `len(result) == 3` AND all IDs present
- **Fail indicators**: error returned, fewer or more than 3 classes, wrong class data

---

### TC-017: ADMIN gets all classes when repository is empty
- **Type**: Unit
- **Covers**: AC-4 — ADMIN empty result boundary
- **Pre-conditions**: Mock `FindAll` returns empty slice, nil error
- **Test data**:
  - Input: `permission = ContextPermission{UserID: "admin-u-001", Role: "ADMIN"}`
  - Expected output: empty `[]ClassDTO{}`, `err == nil`
- **Steps**:
  1. Call `GetClasses` with ADMIN permission and empty repository
  2. Assert: no error
  3. Assert: result is non-nil empty slice (not nil)
- **Pass criteria**: `err == nil` AND `len(result) == 0` AND result is not nil
- **Fail indicators**: nil returned instead of empty slice, error returned

---

### TC-018: TEACHER gets only their own classes
- **Type**: Unit
- **Covers**: AC-3, AC-4 — TEACHER GetClasses happy path
- **Pre-conditions**: Mock `IClassRepository.FindByTeacherUserID("teacher-u-002")` returns `[classA, classC]`; `FindAll` would return `[classA, classB, classC]` but must not be called
- **Test data**:
  - Input: `permission = ContextPermission{UserID: "teacher-u-002", Role: "TEACHER"}`
  - Expected output: `[]ClassDTO` with classA and classC only, `err == nil`
- **Steps**:
  1. Call `GetClasses(ctx, ContextPermission{UserID: "teacher-u-002", Role: "TEACHER"})`
  2. Assert: `FindByTeacherUserID` was called with `"teacher-u-002"`
  3. Assert: `FindAll` was NOT called
  4. Assert: no error
  5. Assert: result contains exactly classA and classC
- **Pass criteria**: `err == nil` AND result matches TEACHER's classes AND wrong repo method not called
- **Fail indicators**: `FindAll` called, wrong classes returned, error returned

---

### TC-019: TEACHER with no assigned classes returns empty slice
- **Type**: Unit
- **Covers**: AC-4 — TEACHER empty result boundary
- **Pre-conditions**: Mock `FindByTeacherUserID` returns empty slice, nil error
- **Test data**:
  - Input: `permission = ContextPermission{UserID: "teacher-u-999", Role: "TEACHER"}`
  - Expected output: empty `[]ClassDTO{}`, `err == nil`
- **Steps**:
  1. Call `GetClasses` with TEACHER permission and no classes assigned
  2. Assert: no error
  3. Assert: empty non-nil slice
- **Pass criteria**: `err == nil` AND `len(result) == 0`
- **Fail indicators**: error returned, nil result

---

### TC-020: STUDENT gets only their single homeroom class
- **Type**: Unit
- **Covers**: AC-3, AC-4 — STUDENT GetClasses happy path
- **Pre-conditions**: Mock `IClassRepository.FindByStudentUserID("student-u-003")` returns `classB`
- **Test data**:
  - Input: `permission = ContextPermission{UserID: "student-u-003", Role: "STUDENT"}`
  - Expected output: `[]ClassDTO` with exactly one entry matching classB, `err == nil`
- **Steps**:
  1. Call `GetClasses(ctx, ContextPermission{UserID: "student-u-003", Role: "STUDENT"})`
  2. Assert: `FindByStudentUserID` was called with `"student-u-003"`
  3. Assert: `FindAll` and `FindByTeacherUserID` were NOT called
  4. Assert: no error
  5. Assert: result length is 1, entry matches classB
- **Pass criteria**: `err == nil` AND result has exactly 1 class matching classB
- **Fail indicators**: wrong method called, result has >1 entry or wrong class, error returned

---

### TC-021: Unknown role returns ErrUnauthorized from GetClasses
- **Type**: Unit
- **Covers**: AC-3, AC-4 — GetClasses unauthorized role branch
- **Pre-conditions**: Mock repository available but should not be called
- **Test data**:
  - Input: `permission = ContextPermission{UserID: "u-004", Role: "JANITOR"}`
  - Expected output: `nil, domainAuth.ErrUnauthorized` (or `class.ErrUnauthorized`)
- **Steps**:
  1. Call `GetClasses(ctx, ContextPermission{UserID: "u-004", Role: "JANITOR"})`
  2. Assert: `errors.Is(err, ErrUnauthorized)` is true
  3. Assert: no repository method was called
- **Pass criteria**: returned error wraps/equals `ErrUnauthorized` AND no repo call made
- **Fail indicators**: nil error, different error type, repo was queried

---

### TC-022: Empty role string returns ErrUnauthorized from GetClasses
- **Type**: Unit
- **Covers**: AC-4 — boundary: empty role
- **Pre-conditions**: Mock repository available
- **Test data**:
  - Input: `permission = ContextPermission{UserID: "u-005", Role: ""}`
  - Expected output: `nil, ErrUnauthorized`
- **Steps**:
  1. Call `GetClasses(ctx, ContextPermission{UserID: "u-005", Role: ""})`
  2. Assert: error is `ErrUnauthorized`
- **Pass criteria**: `errors.Is(err, ErrUnauthorized) == true`
- **Fail indicators**: nil error, 500-class error, repo called

---

#### Group E — Class Usecase Unit Tests: GetClassByID

---

### TC-023: ADMIN finds an existing class by ID
- **Type**: Unit
- **Covers**: AC-3, AC-4 — ADMIN GetClassByID happy path
- **Pre-conditions**: Mock `FindByID("class-id-aaa")` returns classA
- **Test data**:
  - Input: `classID = "class-id-aaa"`, `permission = ContextPermission{UserID: "admin-u-001", Role: "ADMIN"}`
  - Expected output: `*ClassDTO` matching classA, `err == nil`
- **Steps**:
  1. Call `GetClassByID(ctx, "class-id-aaa", ContextPermission{Role: "ADMIN", ...})`
  2. Assert: no error
  3. Assert: returned DTO ID equals `"class-id-aaa"`, Name and Grade match classA
- **Pass criteria**: `err == nil` AND DTO fields match classA
- **Fail indicators**: error returned, wrong class returned

---

### TC-024: ADMIN gets 401 when class does not exist — NOT 404
- **Type**: Unit
- **Covers**: AC-3, AC-4 — ADMIN GetClassByID non-existent class; 401 anti-enumeration requirement
- **Pre-conditions**: Mock `FindByID` returns `nil, class.ErrClassNotFound`
- **Test data**:
  - Input: `classID = "class-id-nonexistent"`, `permission = ContextPermission{Role: "ADMIN"}`
  - Expected output: `nil, domainAuth.ErrUnauthorized` (usecase maps ErrClassNotFound → ErrUnauthorized per spec) OR `nil, class.ErrClassNotFound` that handler will map to 401
- **Steps**:
  1. Call `GetClassByID(ctx, "class-id-nonexistent", ContextPermission{Role: "ADMIN", ...})`
  2. Assert: error is `ErrUnauthorized` OR `ErrClassNotFound` (whichever the implementation uses — the handler must map it to 401, not 404)
  3. Assert: result is nil
- **Pass criteria**: non-nil error AND the error is either `ErrUnauthorized` or `ErrClassNotFound` (both must map to HTTP 401 in the handler)
- **Fail indicators**: nil error, HTTP 404 produced anywhere in the chain

---

### TC-025: TEACHER finds a class they teach
- **Type**: Unit
- **Covers**: AC-3, AC-4 — TEACHER GetClassByID happy path
- **Pre-conditions**: Mock `FindByID("class-id-aaa")` returns classA; Mock `FindByTeacherUserID("teacher-u-002")` returns `[classA, classC]`
- **Test data**:
  - Input: `classID = "class-id-aaa"`, `permission = ContextPermission{UserID: "teacher-u-002", Role: "TEACHER"}`
  - Expected output: `*ClassDTO` matching classA, `err == nil`
- **Steps**:
  1. Call `GetClassByID(ctx, "class-id-aaa", permission)`
  2. Assert: no error
  3. Assert: returned DTO matches classA
- **Pass criteria**: `err == nil` AND DTO matches classA
- **Fail indicators**: error returned, wrong class, ADMIN path taken

---

### TC-026: TEACHER gets ErrUnauthorized for a class they do not teach
- **Type**: Unit
- **Covers**: AC-3, AC-4 — TEACHER scope rejection on a class they don't own
- **Pre-conditions**: Mock `FindByID("class-id-bbb")` returns classB; Mock teacher's classes are `[classA, classC]` (not classB)
- **Test data**:
  - Input: `classID = "class-id-bbb"`, `permission = ContextPermission{UserID: "teacher-u-002", Role: "TEACHER"}`
  - Expected output: `nil, ErrUnauthorized`
- **Steps**:
  1. Call `GetClassByID(ctx, "class-id-bbb", permission)`
  2. Assert: `errors.Is(err, ErrUnauthorized)` is true
  3. Assert: result is nil
- **Pass criteria**: `ErrUnauthorized` returned AND result is nil
- **Fail indicators**: classB returned, nil error, different error type

---

### TC-027: TEACHER gets ErrUnauthorized when class does not exist
- **Type**: Unit
- **Covers**: AC-3, AC-4 — TEACHER on non-existent class must be 401, not 404
- **Pre-conditions**: Mock `FindByID` returns `nil, class.ErrClassNotFound`
- **Test data**:
  - Input: `classID = "class-id-nonexistent"`, `permission = ContextPermission{UserID: "teacher-u-002", Role: "TEACHER"}`
  - Expected output: `nil, ErrUnauthorized` (or `ErrClassNotFound` that handler maps to 401)
- **Steps**:
  1. Call `GetClassByID(ctx, "class-id-nonexistent", permission)`
  2. Assert: error is `ErrUnauthorized` or `ErrClassNotFound`
  3. Assert: result is nil
- **Pass criteria**: error is returned, result is nil, error type maps to 401 not 404
- **Fail indicators**: nil error, 404-mapped error returned

---

### TC-028: STUDENT finds their homeroom class
- **Type**: Unit
- **Covers**: AC-3, AC-4 — STUDENT GetClassByID happy path
- **Pre-conditions**: Mock `FindByStudentUserID("student-u-003")` returns classB; Mock `FindByID("class-id-bbb")` returns classB
- **Test data**:
  - Input: `classID = "class-id-bbb"`, `permission = ContextPermission{UserID: "student-u-003", Role: "STUDENT"}`
  - Expected output: `*ClassDTO` matching classB, `err == nil`
- **Steps**:
  1. Call `GetClassByID(ctx, "class-id-bbb", permission)`
  2. Assert: no error
  3. Assert: returned DTO matches classB
- **Pass criteria**: `err == nil` AND DTO matches classB
- **Fail indicators**: error returned, wrong class returned

---

### TC-029: STUDENT gets ErrUnauthorized for a class different from their homeroom
- **Type**: Unit
- **Covers**: AC-3, AC-4 — STUDENT scope rejection on different class
- **Pre-conditions**: Mock `FindByStudentUserID("student-u-003")` returns classB (ID = "class-id-bbb"); requested class is classA (ID = "class-id-aaa")
- **Test data**:
  - Input: `classID = "class-id-aaa"`, `permission = ContextPermission{UserID: "student-u-003", Role: "STUDENT"}`
  - Expected output: `nil, ErrUnauthorized`
- **Steps**:
  1. Call `GetClassByID(ctx, "class-id-aaa", permission)`
  2. Assert: `errors.Is(err, ErrUnauthorized)` is true
  3. Assert: result is nil
- **Pass criteria**: `ErrUnauthorized` returned AND result is nil
- **Fail indicators**: classA returned, nil error

---

### TC-030: STUDENT gets ErrUnauthorized when class does not exist
- **Type**: Unit
- **Covers**: AC-3, AC-4 — STUDENT on non-existent class must be 401, not 404
- **Pre-conditions**: Mock `FindByID("class-id-nonexistent")` returns `nil, ErrClassNotFound`; student's homeroom is classB
- **Test data**:
  - Input: `classID = "class-id-nonexistent"`, `permission = ContextPermission{UserID: "student-u-003", Role: "STUDENT"}`
  - Expected output: `nil, ErrUnauthorized` or `ErrClassNotFound`
- **Steps**:
  1. Call `GetClassByID(ctx, "class-id-nonexistent", permission)`
  2. Assert: error is not nil
  3. Assert: result is nil
- **Pass criteria**: non-nil error that maps to HTTP 401 in handler
- **Fail indicators**: nil error, error maps to 404

---

### TC-031: Unknown role returns ErrUnauthorized from GetClassByID
- **Type**: Unit
- **Covers**: AC-4 — GetClassByID unauthorized role branch
- **Pre-conditions**: Any mock repository
- **Test data**:
  - Input: `classID = "class-id-aaa"`, `permission = ContextPermission{UserID: "u-006", Role: "VISITOR"}`
  - Expected output: `nil, ErrUnauthorized`
- **Steps**:
  1. Call `GetClassByID(ctx, "class-id-aaa", permission)`
  2. Assert: `errors.Is(err, ErrUnauthorized)` is true
  3. Assert: no repository method was called
- **Pass criteria**: `ErrUnauthorized` returned AND no repo call made
- **Fail indicators**: nil error, repo called, class returned

---

#### Group F — Class Handler HTTP Layer Tests

---

### TC-032: GET /api/v1/classes returns 200 and class list for ADMIN
- **Type**: Integration
- **Covers**: AC-3 — handler maps usecase success to 200 with class list
- **Pre-conditions**: In-process Echo; mock `IClassUseCase.GetClasses` returns `[classA, classB]`; middleware stub injects `ContextPermission{Role: "ADMIN"}`
- **Test data**:
  - Input: `GET /api/v1/classes` with valid Bearer token for ADMIN
  - Expected output: HTTP 200, body `{"data": [{"id":"...","name":"1A1","grade":1}, {"id":"...","name":"1A2","grade":1}]}`
- **Steps**:
  1. Set up Echo with JWT middleware (stub that injects ADMIN permission) and class handler
  2. Send GET request
  3. Assert: status 200
  4. Assert: body is valid JSON array with 2 class objects
  5. Assert: each object has `id`, `name`, `grade` fields
- **Pass criteria**: HTTP 200 AND body contains exactly 2 class objects with correct fields
- **Fail indicators**: non-200 status, missing fields, wrong count

---

### TC-033: GET /api/v1/classes returns 200 and single-element list for STUDENT
- **Type**: Integration
- **Covers**: AC-3 — STUDENT sees only their class via list endpoint
- **Pre-conditions**: Mock `GetClasses` returns `[classB]` for STUDENT permission; middleware injects STUDENT permission
- **Test data**:
  - Input: `GET /api/v1/classes` with STUDENT Bearer token
  - Expected output: HTTP 200, body with array of 1 class
- **Steps**:
  1. Set up Echo with STUDENT permission injected
  2. Send GET request
  3. Assert: status 200
  4. Assert: body array has exactly 1 entry
- **Pass criteria**: HTTP 200 AND 1-element array
- **Fail indicators**: non-200 status, more than 1 class returned

---

### TC-034: GET /api/v1/classes returns 401 when usecase returns ErrUnauthorized
- **Type**: Integration
- **Covers**: AC-3, AC-4 — handler maps ErrUnauthorized → 401
- **Pre-conditions**: Mock `GetClasses` returns `ErrUnauthorized`; middleware injects some permission
- **Test data**:
  - Input: `GET /api/v1/classes` with token for unknown role
  - Expected output: HTTP 401
- **Steps**:
  1. Configure mock usecase to return `ErrUnauthorized`
  2. Send GET request
  3. Assert: status 401
- **Pass criteria**: HTTP 401
- **Fail indicators**: status 403, 500, or 200

---

### TC-035: GET /api/v1/classes returns 500 when usecase returns unexpected error
- **Type**: Integration
- **Covers**: AC-4 — handler maps non-domain errors → 500
- **Pre-conditions**: Mock `GetClasses` returns `errors.New("db timeout")`
- **Test data**:
  - Input: `GET /api/v1/classes` with any valid token
  - Expected output: HTTP 500, body `{"message":"Internal server error"}`
- **Steps**:
  1. Configure mock usecase to return a generic error
  2. Send GET request
  3. Assert: status 500
  4. Assert: body message indicates internal error
- **Pass criteria**: HTTP 500 AND body contains error indicator
- **Fail indicators**: status 200 or 401, panic, no body

---

### TC-036: GET /api/v1/classes/{classId} returns 200 for ADMIN on existing class
- **Type**: Integration
- **Covers**: AC-3 — handler maps usecase success to 200 with single class
- **Pre-conditions**: Mock `GetClassByID("class-id-aaa", ADMIN permission)` returns classA DTO
- **Test data**:
  - Input: `GET /api/v1/classes/class-id-aaa` with ADMIN token
  - Expected output: HTTP 200, body `{"id":"class-id-aaa","name":"1A1","grade":1}`
- **Steps**:
  1. Send GET request to `/api/v1/classes/class-id-aaa`
  2. Assert: status 200
  3. Assert: body contains `id`, `name`, `grade` matching classA
- **Pass criteria**: HTTP 200 AND body matches classA
- **Fail indicators**: non-200 status, wrong class in body

---

### TC-037: GET /api/v1/classes/{classId} returns 401 when class not found — NOT 404
- **Type**: Integration
- **Covers**: AC-3, AC-4 — anti-enumeration: not-found MUST be 401
- **Pre-conditions**: Mock `GetClassByID` returns `ErrUnauthorized` or `ErrClassNotFound`
- **Test data**:
  - Input: `GET /api/v1/classes/nonexistent-class-id` with ADMIN token
  - Expected output: HTTP 401 (explicitly not 404)
- **Steps**:
  1. Configure mock to return `ErrClassNotFound` (or `ErrUnauthorized`)
  2. Send GET request to `/api/v1/classes/nonexistent-class-id`
  3. Assert: status is 401
  4. Assert: status is NOT 404
- **Pass criteria**: HTTP 401 AND NOT 404
- **Fail indicators**: HTTP 404, HTTP 200, any non-401 response

---

### TC-038: GET /api/v1/classes/{classId} returns 401 when TEACHER does not teach that class
- **Type**: Integration
- **Covers**: AC-3 — TEACHER scope rejection maps to 401
- **Pre-conditions**: Mock `GetClassByID` returns `ErrUnauthorized` for this teacher-class combination
- **Test data**:
  - Input: `GET /api/v1/classes/class-id-bbb` with TEACHER token
  - Expected output: HTTP 401
- **Steps**:
  1. Configure mock to return `ErrUnauthorized`
  2. Send GET with TEACHER token
  3. Assert: status 401
- **Pass criteria**: HTTP 401
- **Fail indicators**: any other status code

---

### TC-039: GET /api/v1/classes/{classId} returns 401 when STUDENT requests a different class
- **Type**: Integration
- **Covers**: AC-3 — STUDENT scope rejection maps to 401
- **Pre-conditions**: Mock `GetClassByID` returns `ErrUnauthorized` for wrong class
- **Test data**:
  - Input: `GET /api/v1/classes/class-id-aaa` with STUDENT token (homeroom is class-id-bbb)
  - Expected output: HTTP 401
- **Steps**:
  1. Configure mock to return `ErrUnauthorized`
  2. Send GET with STUDENT token
  3. Assert: status 401
- **Pass criteria**: HTTP 401
- **Fail indicators**: HTTP 200, 403, 404

---

### TC-040: GET /api/v1/classes without Authorization header returns 401
- **Type**: Integration
- **Covers**: AC-2 — middleware blocks unauthenticated requests to protected routes
- **Pre-conditions**: Real JWT middleware registered on the route group
- **Test data**:
  - Input: `GET /api/v1/classes` with no Authorization header
  - Expected output: HTTP 401
- **Steps**:
  1. Send GET request without Authorization header
  2. Assert: status 401
  3. Assert: handler (usecase) was not called
- **Pass criteria**: HTTP 401 AND handler not reached
- **Fail indicators**: 200, 403, 500, or handler called

---

### TC-041: GET /api/v1/classes with invalid Bearer token returns 401
- **Type**: Integration
- **Covers**: AC-2 — middleware rejects invalid token before handler runs
- **Pre-conditions**: Real JWT middleware registered
- **Test data**:
  - Input: `GET /api/v1/classes` with `Authorization: Bearer garbage.token.string`
  - Expected output: HTTP 401
- **Steps**:
  1. Send GET with invalid token
  2. Assert: status 401
- **Pass criteria**: HTTP 401
- **Fail indicators**: any other status, handler called

---

#### Group G — POST /auth/google Response Update

---

### TC-042: POST /auth/google response includes token field
- **Type**: Integration
- **Covers**: AC-3, AC-4 — AuthResponse now includes `token` field containing a JWT
- **Pre-conditions**: Mock `IGoogleVerifier` returns valid claims; mock `IUserRepository` returns existing TEACHER user; `IJWTService.Sign` is real (or a controlled stub)
- **Test data**:
  - Input: `POST /auth/google` with `{"idToken": "mocked-valid-token"}`
  - Expected output: HTTP 200, body contains `"token": "<non-empty-JWT-string>"` in addition to `name`, `email`, `role`, `permissions`
- **Steps**:
  1. Send POST /auth/google with valid mock idToken
  2. Assert: status 200
  3. Assert: response body has key `"token"` with a non-empty string value
  4. Assert: the token string can be verified by the JWT service and yields the correct userID and role
- **Pass criteria**: HTTP 200 AND `token` field is present AND the token is parseable and contains correct claims
- **Fail indicators**: missing `token` field, `token` is empty string, token verifies with wrong claims, status non-200

---

### TC-043: POST /auth/google token encodes correct userID and role
- **Type**: Integration
- **Covers**: AC-4 — token claims match authenticated user
- **Pre-conditions**: Same as TC-042 but user is TEACHER
- **Test data**:
  - Input: existing TEACHER user `userID = "teacher-u-002"`, `role = "TEACHER"`
  - Expected: token decodes to `ContextPermission{UserID: "teacher-u-002", Role: "TEACHER"}`
- **Steps**:
  1. Execute POST /auth/google
  2. Extract `token` from response
  3. Call `jwtService.Verify(token)`
  4. Assert: `permission.UserID == "teacher-u-002"`
  5. Assert: `permission.Role == "TEACHER"`
- **Pass criteria**: both claims match exactly
- **Fail indicators**: wrong userID or role in token, verification fails

---

### TC-044: POST /auth/google for new user returns token with STUDENT role
- **Type**: Integration
- **Covers**: AC-4 — new users default to STUDENT in token as well as in response body
- **Pre-conditions**: Mock verifier returns claims for email not in DB; sign-up path executes
- **Test data**:
  - Input: new user email `"brandnew@example.com"`
  - Expected: token `role == "STUDENT"`
- **Steps**:
  1. Execute POST /auth/google for a new user
  2. Assert: `role` in response body is `"STUDENT"`
  3. Extract `token`, verify it
  4. Assert: token contains `role = "STUDENT"`
- **Pass criteria**: body role and token role are both `"STUDENT"`
- **Fail indicators**: role mismatch between body and token, missing token

---

### TC-045: POST /auth/google with invalid Google token still returns 401 with no token field
- **Type**: Integration
- **Covers**: AC-4 — invalid token path does not leak a JWT
- **Pre-conditions**: Mock verifier returns `ErrInvalidToken`
- **Test data**:
  - Input: `{"idToken": "invalid-google-token"}`
  - Expected: HTTP 401, body does NOT contain a `"token"` field
- **Steps**:
  1. Send POST /auth/google with invalid idToken
  2. Assert: status 401
  3. Assert: body does not contain key `"token"` (or value is absent/null)
- **Pass criteria**: HTTP 401 AND no token field in response
- **Fail indicators**: 200, token field present, token is non-empty string

---

#### Group H — Casbin Policy Verification

---

### TC-046: Policy CSV contains TEACHER GET /api/v1/classes
- **Type**: Unit
- **Covers**: AC-3 — policy file updated correctly
- **Pre-conditions**: `internal/infrastructure/auth/policies.csv` file is readable
- **Test data**:
  - Expected: line `p, TEACHER, /api/v1/classes, GET` (exact format as per existing lines)
- **Steps**:
  1. Call `parsePolicies(policiesCSV)`
  2. Filter for `role == "TEACHER"`, `path == "/api/v1/classes"`, `method == "GET"`
  3. Assert: at least one entry matches
- **Pass criteria**: matching policy row found
- **Fail indicators**: no matching row, wrong path format

---

### TC-047: Policy CSV contains TEACHER GET /api/v1/classes/:class_id
- **Type**: Unit
- **Covers**: AC-3 — policy file updated correctly
- **Pre-conditions**: `policies.csv` readable
- **Test data**:
  - Expected: line `p, TEACHER, /api/v1/classes/:class_id, GET`
- **Steps**:
  1. Parse policies
  2. Filter for `TEACHER, /api/v1/classes/:class_id, GET`
  3. Assert: entry found
- **Pass criteria**: entry found
- **Fail indicators**: no matching row

---

### TC-048: Policy CSV contains STUDENT GET /api/v1/classes
- **Type**: Unit
- **Covers**: AC-3 — STUDENT policy added
- **Pre-conditions**: `policies.csv` readable
- **Test data**:
  - Expected: `p, STUDENT, /api/v1/classes, GET`
- **Steps**:
  1. Parse policies
  2. Filter for `STUDENT, /api/v1/classes, GET`
  3. Assert: entry found
- **Pass criteria**: entry found
- **Fail indicators**: no matching row

---

### TC-049: Policy CSV contains STUDENT GET /api/v1/classes/:class_id
- **Type**: Unit
- **Covers**: AC-3 — STUDENT policy added
- **Pre-conditions**: `policies.csv` readable
- **Test data**:
  - Expected: `p, STUDENT, /api/v1/classes/:class_id, GET`
- **Steps**:
  1. Parse policies
  2. Filter for `STUDENT, /api/v1/classes/:class_id, GET`
  3. Assert: entry found
- **Pass criteria**: entry found
- **Fail indicators**: no matching row

---

#### Group I — E2E / Full Integration Tests

---

### TC-050: ADMIN full flow — sign in, get token, list all classes
- **Type**: E2E
- **Covers**: AC-3 — end-to-end ADMIN happy path
- **Pre-conditions**: In-memory SQLite seeded with 3 classes; ADMIN user exists with casbin_rule row; Google verifier mocked
- **Test data**:
  - Step 1: POST /auth/google → get `token`
  - Step 2: GET /api/v1/classes with `Authorization: Bearer <token>`
  - Expected: step 2 returns 200 with 3 classes
- **Steps**:
  1. POST /auth/google with mocked ADMIN credentials → capture `token`
  2. Assert step 1: status 200 AND `token` present
  3. GET /api/v1/classes with `Authorization: Bearer <token>`
  4. Assert: status 200 AND result has 3 classes
- **Pass criteria**: both requests succeed AND class count matches DB seed
- **Fail indicators**: 401 on step 2, wrong class count, missing token from step 1

---

### TC-051: TEACHER full flow — sign in, get token, list only own classes
- **Type**: E2E
- **Covers**: AC-3 — end-to-end TEACHER happy path
- **Pre-conditions**: DB seeded: 3 classes; TEACHER user assigned to classA and classC via `class_subjects`; TEACHER casbin_rule row exists
- **Test data**:
  - POST /auth/google → token for TEACHER
  - GET /api/v1/classes → should return 2 classes (classA, classC)
- **Steps**:
  1. POST /auth/google for TEACHER user → capture token
  2. GET /api/v1/classes with TEACHER token
  3. Assert: status 200 AND exactly 2 classes returned
  4. Assert: returned classes are classA and classC (not classB)
- **Pass criteria**: status 200, 2 classes, correct IDs
- **Fail indicators**: all 3 classes returned, 401, wrong class IDs

---

### TC-052: STUDENT full flow — sign in, get token, see only their homeroom class in list
- **Type**: E2E
- **Covers**: AC-3 — end-to-end STUDENT happy path for list
- **Pre-conditions**: DB seeded: 3 classes; STUDENT user homeroom = classB; STUDENT casbin_rule row exists
- **Test data**:
  - POST /auth/google → token for STUDENT
  - GET /api/v1/classes → should return 1 class (classB)
- **Steps**:
  1. POST /auth/google for STUDENT user → capture token
  2. GET /api/v1/classes with STUDENT token
  3. Assert: status 200 AND exactly 1 class returned
  4. Assert: returned class ID matches classB
- **Pass criteria**: status 200, 1 class, ID matches classB
- **Fail indicators**: multiple classes returned, wrong class, 401

---

### TC-053: ADMIN full flow — get single class by ID
- **Type**: E2E
- **Covers**: AC-3 — ADMIN single class happy path
- **Pre-conditions**: DB seeded with classA; ADMIN token from TC-050 flow
- **Test data**:
  - GET /api/v1/classes/{classA.ID}
  - Expected: 200 with classA data
- **Steps**:
  1. GET /api/v1/classes/{classA.ID} with ADMIN token
  2. Assert: status 200
  3. Assert: body `id == classA.ID`, `name == "1A1"`, `grade == 1`
- **Pass criteria**: 200 with correct class data
- **Fail indicators**: 401, wrong data

---

### TC-054: ADMIN gets 401 for nonexistent class ID (not 404)
- **Type**: E2E
- **Covers**: AC-3, AC-4 — critical anti-enumeration requirement
- **Pre-conditions**: DB does NOT contain class with ID `"00000000-0000-0000-0000-000000000099"`; ADMIN token valid
- **Test data**:
  - GET /api/v1/classes/00000000-0000-0000-0000-000000000099
  - Expected: HTTP 401 (NOT 404)
- **Steps**:
  1. GET /api/v1/classes/00000000-0000-0000-0000-000000000099 with valid ADMIN token
  2. Assert: status 401
  3. Assert: status is NOT 404
- **Pass criteria**: HTTP 401 AND NOT 404
- **Fail indicators**: 404, 200, 500

---

### TC-055: TEACHER gets 401 for a class they don't teach
- **Type**: E2E
- **Covers**: AC-3 — TEACHER scope rejection end-to-end
- **Pre-conditions**: DB seeded; TEACHER teaches classA and classC; classB exists; TEACHER token valid
- **Test data**:
  - GET /api/v1/classes/{classB.ID} with TEACHER token
  - Expected: HTTP 401
- **Steps**:
  1. GET /api/v1/classes/{classB.ID} with TEACHER token
  2. Assert: status 401
- **Pass criteria**: HTTP 401
- **Fail indicators**: 200 (classB returned to TEACHER who doesn't teach it), 404

---

### TC-056: STUDENT gets 401 for a class that is not their homeroom
- **Type**: E2E
- **Covers**: AC-3 — STUDENT scope rejection end-to-end
- **Pre-conditions**: STUDENT homeroom is classB; classA exists; STUDENT token valid
- **Test data**:
  - GET /api/v1/classes/{classA.ID} with STUDENT token
  - Expected: HTTP 401
- **Steps**:
  1. GET /api/v1/classes/{classA.ID} with STUDENT token
  2. Assert: status 401
- **Pass criteria**: HTTP 401
- **Fail indicators**: 200, 403, 404

---

### TC-057: STUDENT gets 200 for their own homeroom class by ID
- **Type**: E2E
- **Covers**: AC-3 — STUDENT single class happy path end-to-end
- **Pre-conditions**: STUDENT homeroom is classB; STUDENT token valid
- **Test data**:
  - GET /api/v1/classes/{classB.ID} with STUDENT token
  - Expected: HTTP 200 with classB data
- **Steps**:
  1. GET /api/v1/classes/{classB.ID} with STUDENT token
  2. Assert: status 200
  3. Assert: body ID equals classB.ID
- **Pass criteria**: HTTP 200 AND correct class returned
- **Fail indicators**: 401, wrong class, 404

---

### TC-058: Middleware is reusable — applies to both /classes and /classes/{classId} route groups
- **Type**: Integration
- **Covers**: AC-2 — reusable middleware applied consistently
- **Pre-conditions**: Real JWT middleware registered on a route group covering both endpoints
- **Test data**:
  - Input A: GET /api/v1/classes with no token → expect 401
  - Input B: GET /api/v1/classes/{anyId} with no token → expect 401
- **Steps**:
  1. Send GET /api/v1/classes without Authorization → assert 401
  2. Send GET /api/v1/classes/some-id without Authorization → assert 401
  3. Assert: both responses are 401 (not 404 or 403)
- **Pass criteria**: both requests return 401
- **Fail indicators**: either request returns non-401, middleware only applied to one route

---

## Test Data Catalogue

### Roles

| Name | Role string | Notes |
|------|-------------|-------|
| adminUser | `"ADMIN"` | UserID `"admin-u-001"` |
| teacherUser | `"TEACHER"` | UserID `"teacher-u-002"` |
| studentUser | `"STUDENT"` | UserID `"student-u-003"` |
| unknownRoleUser | `"JANITOR"` | Used for rejection tests |
| emptyRoleUser | `""` | Boundary — empty role |

### Classes (DB seed)

| Alias | Name | Grade | ID |
|-------|------|-------|----|
| classA | `"1A1"` | 1 | `"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"` |
| classB | `"1A2"` | 1 | `"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"` |
| classC | `"2B1"` | 2 | `"cccccccc-cccc-cccc-cccc-cccccccccccc"` |

### Teacher assignments (class_subjects)

| Teacher UserID | Classes |
|----------------|---------|
| `"teacher-u-002"` | classA, classC |

### Student homeroom (students table)

| Student UserID | HomeRoom |
|----------------|----------|
| `"student-u-003"` | classB |

### JWT Test Tokens

| Token alias | Description |
|-------------|-------------|
| `adminToken` | Valid JWT signed with test secret, `userID=admin-u-001, role=ADMIN` |
| `teacherToken` | Valid JWT, `userID=teacher-u-002, role=TEACHER` |
| `studentToken` | Valid JWT, `userID=student-u-003, role=STUDENT` |
| `expiredToken` | Valid JWT with `exp = time.Now() - 1s` |
| `wrongSecretToken` | JWT signed with a different secret |
| `unknownRoleToken` | Valid JWT, `role=JANITOR` |
| `emptyUserIDToken` | Valid JWT, `userID=""` |
| `emptyRoleToken` | Valid JWT, `role=""` |

### Nonexistent Class ID

`"00000000-0000-0000-0000-000000000099"` — UUID format but not present in any test database seed.

---

## Out of Scope

- **User management endpoints** (`/users`, `/auth/google` outside the token field change) — covered in GS-001.
- **Slot editing and deletion** — separate feature; `SlotOwnerMiddleware` not part of GS-002.
- **Subject endpoints** — no changes in scope.
- **Role promotion** — admin-only user management is a separate task.
- **Frontend OAuth flow** — FE code is explicitly excluded.
- **Refresh token handling** — not part of this feature.
- **Database migrations** — correctness of migration files is a DBA concern; tests use an auto-migrated in-memory DB.
- **Wire DI wiring** — `wire.go` / `wire_gen.go` correctness is verified transitively via E2E tests, not with dedicated test cases.
- **Race condition testing** — concurrent sign-ins or concurrent class fetches are not tested in this cycle.
- **ADMIN `POST /api/v1/classes`** — write endpoint for ADMIN is listed in existing policies but is not part of GS-002 implementation; no handler is being written for it.
