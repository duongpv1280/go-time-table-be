# go-time-table-be RBAC System: End-to-End Technical Explanation

---

## 1. Database Schema

### Tables and Their Purpose

#### `casbin_rule`
Central authorization policy table used by the Casbin enforcer.

| Column | Type | Purpose |
|--------|------|---------|
| `id` | INT UNSIGNED PK | Auto-increment row identifier |
| `ptype` | VARCHAR(100) | Policy type: `p` (permission) or `g` (role assignment) |
| `v0`–`v5` | VARCHAR(100) | Generic value columns; meaning depends on `ptype` |
| `created_at/updated_at/deleted_at` | TIMESTAMP | Standard audit + soft-delete fields |

`g` rows assign roles to users: `g(userID, ROLE)`. `p` rows (permission policies) are **never in the DB** — they are loaded from the embedded `policies.csv` at startup.

#### `users`
All system accounts regardless of role.

| Column | Type | Purpose |
|--------|------|---------|
| `id` | UUID PK | |
| `email` | VARCHAR UNIQUE | Login identifier |
| `password_hash` | VARCHAR | BCrypt hash |
| `role` | ENUM(`ADMIN`,`TEACHER`,`STUDENT`) | Top-level role; also stored in Casbin g rows |
| `created_at/updated_at/deleted_at` | TIMESTAMP | Audit + soft-delete |

One `ADMIN` account is seeded at startup from environment config. Only the `ADMIN` role can create further user accounts.

#### `subjects`
Course offerings available in the school.

| Column | Type | Purpose |
|--------|------|---------|
| `id` | UUID PK | |
| `name` | VARCHAR | Display name, e.g. Math, Literature, English, Physics |
| `code` | VARCHAR UNIQUE | Short code, e.g. `MATH`, `LIT`, `ENG` |
| `created_at/updated_at/deleted_at` | TIMESTAMP | |

#### `classes`
Student homeroom groupings.

| Column | Type | Purpose |
|--------|------|---------|
| `id` | UUID PK | |
| `name` | VARCHAR UNIQUE | e.g. `1A1`, `1A2`, `2B3` |
| `grade` | INT | School year / grade level |
| `created_at/updated_at/deleted_at` | TIMESTAMP | |

#### `teachers`
Teacher profiles, linked 1:1 to `users`.

| Column | Type | Purpose |
|--------|------|---------|
| `id` | UUID PK | |
| `user_id` | UUID FK → users | |
| `display_name` | VARCHAR | e.g. `Mr. A`, `Ms. B` |

#### `students`
Student profiles, linked 1:1 to `users`.

| Column | Type | Purpose |
|--------|------|---------|
| `id` | UUID PK | |
| `user_id` | UUID FK → users | |
| `class_id` | UUID FK → classes | Homeroom class; fixed for the school year |
| `display_name` | VARCHAR | |

#### `class_subjects`
Declares which teacher teaches which subject in which class. This is the authority for teacher–class membership checks.

| Column | Type | Purpose |
|--------|------|---------|
| `class_id` | UUID FK → classes | |
| `subject_id` | UUID FK → subjects | |
| `teacher_id` | UUID FK → teachers | |

Composite PK `(class_id, subject_id, teacher_id)`. A teacher who teaches multiple subjects in the same class appears in multiple rows.

#### `slots`
Weekly timetable entries.

| Column | Type | Purpose |
|--------|------|---------|
| `id` | UUID PK | |
| `class_id` | UUID FK → classes | |
| `subject_id` | UUID FK → subjects | |
| `teacher_id` | UUID FK → teachers | |
| `day_of_week` | TINYINT | 1 = Monday … 7 = Sunday |
| `start_time` | TIME | e.g. `07:30:00` |
| `end_time` | TIME | e.g. `08:45:00` |
| `created_at/updated_at/deleted_at` | TIMESTAMP | Soft-delete used for day-off removal |

A teacher may update a slot's `subject_id` to move it to a different subject, or soft-delete the row to mark a day off.

### Entity Relationship

```
users
  ├─ teachers (user_id)
  │    └─ class_subjects (teacher_id, class_id, subject_id)
  │         ├─ classes
  │         └─ subjects
  └─ students (user_id)
       └─ classes (class_id — homeroom)

slots (class_id, subject_id, teacher_id, day_of_week, start_time, end_time)
  ├─ classes
  ├─ subjects
  └─ teachers

casbin_rule (ptype=g)
  v0 = userID
  v1 = role name (ADMIN | TEACHER | STUDENT)
```

---

## 2. Role Hierarchy

The system has three flat roles — no nested groups or sub-roles.

| Role | Level | Summary |
|------|-------|---------|
| `ADMIN` | System-wide | Full CRUD on all resources; the only role that can create users, teachers, subjects, and classes |
| `TEACHER` | Resource-scoped | Read students from their own classes; read all teachers and their subjects; edit or remove their own teaching slots |
| `STUDENT` | Resource-scoped | Read subjects and teachers belonging to their own homeroom class only |

`TEACHER` does not inherit `STUDENT` permissions and vice versa. `ADMIN` bypasses Casbin entirely after JWT validation.

---

## 3. Casbin Configuration

### model.conf Explained

```ini
[request_definition]
r = sub, obj, act           # (userID, URL path, HTTP method)

[policy_definition]
p = sub, obj, act           # (role, path pattern, HTTP method)

[role_definition]
g = _, _                    # maps userID → role name

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && r.act == p.act
```

The matcher has three conditions:
1. `g(r.sub, p.sub)` — the requesting user is assigned the role that the policy requires
2. `keyMatch2(r.obj, p.obj)` — the request URL matches the policy path (`:param` wildcards)
3. `r.act == p.act` — the HTTP method matches exactly

### policies.csv Structure

```csv
# Admin — full control over users, subjects, classes, teachers
p, ADMIN, /api/v1/users,                                  GET
p, ADMIN, /api/v1/users,                                  POST
p, ADMIN, /api/v1/users/:user_id,                         GET
p, ADMIN, /api/v1/users/:user_id,                         PUT
p, ADMIN, /api/v1/users/:user_id,                         DELETE
p, ADMIN, /api/v1/subjects,                               GET
p, ADMIN, /api/v1/subjects,                               POST
p, ADMIN, /api/v1/subjects/:subject_id,                   PUT
p, ADMIN, /api/v1/subjects/:subject_id,                   DELETE
p, ADMIN, /api/v1/classes,                                GET
p, ADMIN, /api/v1/classes,                                POST
p, ADMIN, /api/v1/classes/:class_id,                      PUT
p, ADMIN, /api/v1/classes/:class_id,                      DELETE
p, ADMIN, /api/v1/teachers,                               POST
p, ADMIN, /api/v1/slots,                                  POST

# Teacher — read their students/peers, manage their own slots
p, TEACHER, /api/v1/classes/:class_id/students,           GET
p, TEACHER, /api/v1/teachers,                             GET
p, TEACHER, /api/v1/teachers/:teacher_id/subjects,        GET
p, TEACHER, /api/v1/slots/:slot_id,                       PUT
p, TEACHER, /api/v1/slots/:slot_id,                       DELETE

# Student — read-only view of their own class
p, STUDENT, /api/v1/classes/:class_id/subjects,           GET
p, STUDENT, /api/v1/classes/:class_id/teachers,           GET
```

### How g Works

At startup, `loadRolePoliciesFromDB` loads all `casbin_rule` rows where `ptype='g'` and feeds them to the enforcer:

```go
// Loaded from casbin_rule table at boot:
g("user-uuid-001", "ADMIN")
g("user-uuid-002", "TEACHER")
g("user-uuid-003", "TEACHER")
g("user-uuid-004", "STUDENT")
// ...
```

When teacher `user-uuid-002` calls `GET /api/v1/teachers`:
- `g("user-uuid-002", "TEACHER")` → true ✓
- `keyMatch2("/api/v1/teachers", "/api/v1/teachers")` → true ✓
- `"GET" == "GET"` → true ✓
- **Result: allowed**

When the same user calls `GET /api/v1/classes/xyz/students`:
- `g("user-uuid-002", "TEACHER")` → true ✓
- `keyMatch2("/api/v1/classes/xyz/students", "/api/v1/classes/:class_id/students")` → true ✓
- `"GET" == "GET"` → true ✓
- **Casbin passes → ownership check follows** (see §4)

---

## 4. Authorization Flow

### Step by Step

```
1. JWT validation → extract userID and role claim
2. ADMIN shortcut → role == ADMIN bypasses all remaining checks → allow
3. Casbin.Enforce(userID, requestPath, httpMethod)
       → g(userID, role) looked up from casbin_rule rows
       → keyMatch2 against policies.csv path patterns
       → method equality check
4. Resource ownership / scope check (TEACHER and STUDENT only):
       TEACHER + GET /classes/:class_id/students
            → query class_subjects WHERE class_id=? AND teacher_id=?
            → no row → 403
       TEACHER + PUT|DELETE /slots/:slot_id
            → query slots WHERE id=? AND teacher_id=?
            → no row (or teacher mismatch) → 403
       STUDENT + /classes/:class_id/...
            → verify students.class_id == :class_id for this user
            → mismatch → 403
5. On pass: inject ContextPermission{} into echo.Context → handler executes
6. No Casbin match or ownership failure → 403
```

### What Gets Injected

```go
// pkg/helper/context.go
type ContextPermission struct {
    UserID  string
    Role    string  // ADMIN | TEACHER | STUDENT
    Permit  uint32  // bitmask of allowed HTTP methods
    IsOwner bool    // true when ownership check passed (teacher→slot, teacher→class)
}
```

Handlers call `helper.GetPermissionFromContext(ctx)` and inspect `IsOwner` before allowing state-mutating operations.

---

## 5. Permission Bitmask System

```go
// HTTP method → bitmask value
GET    = 1
POST   = 2
PUT    = 4
PATCH  = 8
DELETE = 16

// Composite constants
PermitRead  = 1   // GET only
PermitWrite = 30  // POST + PUT + PATCH + DELETE
PermitAll   = 31  // all five
```

### Ownership Constraint (Teacher → Slot)

Casbin only checks that the user holds the `TEACHER` role. A second middleware layer enforces that a teacher can only mutate **their own** slots:

```go
// middleware/slot_owner.go
slot, _ := slotRepo.FindByID(ctx, slotID)
teacher, _ := teacherRepo.FindByUserID(ctx, userID)
if slot.TeacherID != teacher.ID {
    return echo.ErrForbidden
}
ctx.Set(ContextIsOwner, true)
```

Similarly, for `GET /classes/:class_id/students`, a `ClassMemberMiddleware` checks `class_subjects` before the handler runs.

---

## 6. Resource Scoping

### Teacher Class Scope

A teacher only sees the student list for classes where they are listed in `class_subjects`. Calling `GET /api/v1/classes/:class_id/students`:
1. Casbin allows `TEACHER` role on this path pattern.
2. `ClassMemberMiddleware` queries `class_subjects` for `(class_id=:class_id, teacher_id=<their ID>)`. No row → 403.

Teachers can read the global teacher directory (`GET /api/v1/teachers`) and any teacher's subject list (`GET /api/v1/teachers/:teacher_id/subjects`) without extra scope checks — these are shared read-only views.

### Student Class Scope

A student's homeroom `class_id` is fixed in the `students` table. All endpoints under `/api/v1/classes/:class_id/...` pass through `StudentClassMiddleware`, which compares `:class_id` against `students.class_id` for the requesting user. A mismatch returns 403 before the handler is reached.

Students are permitted to read:
- `GET /api/v1/classes/:class_id/subjects` — all subjects taught in their class
- `GET /api/v1/classes/:class_id/teachers` — all teachers assigned to their class

Students cannot access slot details, student rosters, or any write endpoint.

### Teacher Slot Editing

A teacher may:
- `PUT /api/v1/slots/:slot_id` — change the `subject_id` (reschedule to a different subject) or adjust the time window for a specific day
- `DELETE /api/v1/slots/:slot_id` — soft-delete the slot to mark a day off

Both require `slot.teacher_id == requesting teacher's ID` (enforced by `SlotOwnerMiddleware`). A teacher cannot modify another teacher's slots even if they share a class.

---

## 7. Admin Seeding

One admin account is created at application startup if it does not already exist:

```go
// infrastructure/seeder/admin.go
if !userRepo.ExistsByEmail(ctx, cfg.AdminEmail) {
    user, _ := userRepo.Create(ctx, User{
        Email:        cfg.AdminEmail,
        PasswordHash: bcrypt.Hash(cfg.AdminInitialPassword),
        Role:         RoleAdmin,
    })
    casbinRepo.AddRoleForUser(ctx, user.ID, "ADMIN")
}
```

`cfg.AdminEmail` and `cfg.AdminInitialPassword` are read from environment variables at boot. No other mechanism creates admin accounts — additional admins must be created by an existing admin via `POST /api/v1/users`.

---

## Full Authorization Decision Tree

```
Request arrives
│
├── JWT invalid → 401
│
├── Role == ADMIN → allow (bypass all checks below)
│
├── Casbin.Enforce(userID, path, method)
│   └── No matching policy row → 403
│
├── Resource scope / ownership check
│   ├── TEACHER + GET /classes/:class_id/students
│   │   └── class_subjects has no (class_id, teacher_id) row → 403
│   │
│   ├── TEACHER + PUT|DELETE /slots/:slot_id
│   │   └── slots.teacher_id != teacher.id → 403
│   │
│   └── STUDENT + /classes/:class_id/...
│       └── students.class_id != :class_id → 403
│
└── → 200  (inject ContextPermission into echo.Context)
```
