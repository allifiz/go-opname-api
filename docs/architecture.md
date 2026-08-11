# Architecture V1

## Stack
- Go 1.25+ for the application
- Fiber
- PostgreSQL
- pgx/v5
- Goose migrations
- sqlc
- Air for local hot reload
- GitHub Actions for executable validation
- JWT HS256 for API authentication
- bcrypt for password verification

## Application Flow
Use a pragmatic layered structure:

```text
HTTP Request
    ↓
JWT Authentication Middleware
    ↓
RBAC Guard
    ↓
HTTP Handler
    ↓
Service
    ↓
Repository
    ↓
sqlc / pgx
    ↓
PostgreSQL
```

Avoid unnecessary architecture layers unless a real requirement appears. The objective is explicit business logic, predictable transactions, testable boundaries, and authentication/authorization enforced before business handlers execute.

## Repository Layout

```text
.github/
└── workflows/
    └── ci.yml

internal/
├── config/
├── database/
│   ├── generated/       # sqlc generated code, committed by CI when changed
│   ├── query/           # sqlc SQL queries
│   └── postgres.go
├── domain/
├── repository/
├── service/
└── handler/
    └── http/
```

## Authentication and Authorization

Authentication uses the existing `users` and `roles` tables. No auth-specific migration is required.

Login flow:

```text
POST /api/v1/auth/login
    ↓
lookup active user by email
    ↓
bcrypt password verification
    ↓
issue HS256 JWT (8-hour expiry)
```

JWT claims contain:
- user UUID in `sub`
- role code
- email
- expiry

`JWT_SECRET` is required at application startup and must contain at least 32 characters.

`GET /health` and login are public. Every other `/api/v1` route passes through authentication middleware. The middleware validates JWT signature and expiry, then places the actor ID, role, and email in Fiber locals.

RBAC runs at the HTTP boundary before the business handler. Operational actor fields that previously came from request bodies are overwritten from the authenticated JWT identity. Services continue to validate role-sensitive approval identity against database users where that validation is part of the domain transaction.

Current write-role mapping:
- Ahli Gizi: period/menu/scheduled-menu writes.
- Ahli Gizi or Pengawas Bahan Baku: material writes.
- Pengawas Bahan Baku: procurement operations, receiving, direct purchase, usage entry/submission, opname and adjustment entry/submission.
- Akuntan: procurement verify/reject.
- Chef or Akuntan: usage and stock-adjustment decisions.

No public default user/password is seeded. Initial user provisioning remains a separate secure operational concern.

## Continuous Validation
GitHub Actions is the canonical executable validation environment for repository changes that need external tooling or PostgreSQL.

The ChatGPT execution sandbox cannot be granted outbound internet/DNS access from the conversation. Repository access is handled through the GitHub connector, while compilation, migration execution, and code generation run inside GitHub-hosted runners.

The CI workflow uses:
- Ubuntu GitHub-hosted runner.
- PostgreSQL 17 service.
- Go `1.26.x` toolchain for CI tooling and build validation.
- Goose `v3.27.1`.
- sqlc `v1.31.1`.

The application still declares Go `1.25.0` as its minimum version in `go.mod`. CI intentionally uses a newer compatible Go toolchain because current Goose/sqlc releases require newer Go patch/minor versions.

CI performs:
1. Checkout repository.
2. Setup Go `1.26.x`.
3. Download project dependencies.
4. Install pinned Goose and sqlc versions.
5. Apply all Goose migrations to a clean PostgreSQL database.
6. Roll migrations back to version 0.
7. Apply all migrations again.
8. Run `sqlc generate`.
9. Run `go test ./...` including PostgreSQL integration, concurrency, and authentication tests.
10. Run `go build ./...`.
11. Commit regenerated `internal/database/generated` files back to `main` when sqlc output changes.

## Key Domain Principles

### Menu Snapshot
Menu templates are reusable library entries. When a template is scheduled, its current components and materials are copied into scheduled-menu tables. Historical scheduled menus never depend on the current template contents.

### Gross vs Net Procurement
Production requirement and procurement quantity are distinct domain values:
- `gross_requirement`: production need calculated from scheduled menu.
- `available_stock`: stock that can still be allocated after existing reservations.
- `net_procurement`: quantity that actually needs purchasing.

Never overwrite gross requirement with net procurement.

### Stock Reservation
Reservation is an allocation concept, not a physical stock movement. It prevents the same current stock from offsetting multiple future procurement requirements.

Reservations are kept separate from `stock_movements` because a reservation does not physically add or remove inventory.

### Inventory Ledger
`stock_movements` is the audit source of truth for actual inventory changes. `material_stocks` is a read-optimized current snapshot maintained transactionally.

### Approval Versioning
Entities that require dual approval use a monotonically increasing `version`. Approval rows reference the entity version they approved. Any material edit increments the version and makes old approvals invalid without deleting history.

### Transaction Boundaries
Operations that change stock or reservations must use database transactions and locking where concurrent requests could violate invariants.

Examples:
- procurement stock check + reservation
- receipt + stock IN
- direct purchase + stock IN
- dual-approved material usage + stock OUT
- approved stock adjustment + stock movement

### Negative Stock
A transaction that would produce negative `material_stocks.qty` must fail and roll back.

## Numeric Types
- Quantities: PostgreSQL `NUMERIC(18,4)`.
- Monetary values: PostgreSQL `NUMERIC(18,2)`.
- Never use floating-point values for persisted quantity or monetary calculations.

## Date/Time Types
- Business production/menu date: `DATE`.
- Transaction/audit timestamps: `TIMESTAMPTZ`.

## IDs
Use UUID primary keys generated by PostgreSQL with `gen_random_uuid()`.

## Master Data Strategy
Roles and units are regular master tables instead of PostgreSQL enums. This avoids schema-type migrations merely to add a future role or unit.

Vendor is intentionally not master data in V1. Supplier/source information is stored directly on the relevant procurement/transaction record.

## Documentation Contract
- `docs/progress.md` is updated on every codebase change.
- Other docs are updated whenever their corresponding area changes.
- The approved `rapat` flow reference is not modified as part of routine implementation work.
