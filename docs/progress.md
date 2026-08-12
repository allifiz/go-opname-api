# Project Progress

## Current Phase
Core inventory V1, concurrency hardening, authentication/JWT + RBAC, and secure initial-user bootstrap are implemented and green. Next focus is menu-template update without mutating historical scheduled-menu snapshots.

## Last Updated
2026-08-12

## Documentation Rule
- Update this file on every codebase change.
- Update domain docs when their area changes.
- The approved `rapat` document remains read-only business-flow reference.

## Completed
- Foundation, menu, inventory, procurement, PO, H-1 cancellation, receiving, direct purchase, material usage, stock opname, and stock adjustment implemented.
- Real PostgreSQL concurrency coverage protects receiving, usage approval, adjustment approval, conditional stock OUT, reservation allocation, shortage purchases, and initial-user bootstrap.
- JWT authentication implemented using HS256 with the Go standard library.
- Password verification uses bcrypt against `users.password_hash`.
- `POST /api/v1/auth/login` is public and issues an 8-hour Bearer token containing user ID, role, email, and expiry.
- `GET /api/v1/auth/me` and all non-health business endpoints require a valid Bearer token.
- `JWT_SECRET` is mandatory and must contain at least 32 characters.
- HTTP RBAC guards are implemented:
  - `AHLI_GIZI`: period/menu-template/scheduled-menu writes.
  - `AHLI_GIZI` or `PENGAWAS_BAHAN_BAKU`: material writes.
  - `PENGAWAS_BAHAN_BAKU`: procurement stock check/submission, PO generation/H-1 cancellation, receiving, direct purchase, material usage entry/revision/submission, stock opname/adjustment revision/submission.
  - `AKUNTAN`: procurement verification/rejection.
  - `CHEF` or `AKUNTAN`: material-usage and stock-adjustment decisions.
- Operational audit actors that previously came from request bodies are overwritten at the HTTP boundary from JWT identity.
- Secure initial-user provisioning is implemented through `POST /api/v1/auth/bootstrap`:
  - disabled unless `BOOTSTRAP_TOKEN` is configured;
  - configured bootstrap token must be at least 32 characters;
  - token is supplied in `X-Bootstrap-Token` and compared using constant-time hashed comparison;
  - name/email/password/role are validated and password is bcrypt-hashed;
  - only seeded business roles are accepted;
  - PostgreSQL transaction locks `users`, requires zero existing users, and creates exactly one first user;
  - concurrent bootstrap requests cannot create multiple first users;
  - once any user exists, bootstrap is closed by database state.
- No default credential and no invented `ADMIN` role are introduced.
- Authentication tests cover bcrypt login, valid/tampered/expired tokens, HTTP auth/RBAC, and concurrent bootstrap exclusivity.
- GitHub Actions run #80 passed PostgreSQL startup, migrations, rollback/re-apply, `sqlc generate`, all integration/concurrency/auth/bootstrap tests, build, and generated-code synchronization.

## Implemented Migrations
- `000001_create_foundation.sql`
- `000002_create_menu.sql`
- `000003_create_inventory.sql`
- `000004_create_procurement.sql`
- `000005_purchase_order_constraints.sql`
- `000006_create_receiving.sql`
- `000007_create_direct_purchase.sql`
- `000008_create_material_usage.sql`
- `000009_create_stock_opname.sql`

No new migration is required for JWT auth or initial-user bootstrap; the existing `users`, `roles`, and `password_hash` fields are used.

## Validation Status
GREEN through GitHub Actions run #80.

Validated checks:
- PostgreSQL 17 startup
- Goose migration up
- rollback-to-zero
- migration re-apply
- `sqlc generate`
- real PostgreSQL integration/concurrency/auth/bootstrap tests through `go test ./...`
- `go build ./...`
- generated sqlc synchronization

## Next
1. Add menu-template update flow without mutating historical scheduled-menu snapshots.
2. Add OpenAPI/Swagger and consistent response/error contracts.
3. Add pagination/filtering where list volume can grow.
4. Implement actual object storage for receipt documents.
5. Resolve `KEPALA_SPPG` operational permissions.
6. Define authenticated provisioning of additional users after the one-time bootstrap, once an administrative authorization policy is approved.

## Blockers / TBD
- `KEPALA_SPPG` operational permissions remain TBD.
- Authenticated provisioning of additional users remains TBD because no administrative role/permission has been approved in the business flow.
- Receipt document object storage provider is not implemented yet.
- No application payment workflow is required.

## Changed Files (Latest Batch)
- `internal/config/config.go`
- `internal/repository/bootstrap.go`
- `internal/service/auth_service.go`
- `internal/service/auth_integration_test.go`
- `internal/handler/http/auth_handler.go`
- `cmd/api/main.go`
- `.env.example`
- `docs/progress.md`
- `docs/api.md`
- `docs/architecture.md`
- `docs/decisions.md`
- `Readme.md`

## Notes for Next Agent
Read before changing behavior/schema:
1. `docs/progress.md`
2. `docs/requirements.md`
3. `docs/architecture.md`
4. `docs/database.md`
5. `docs/decisions.md`
6. `docs/api.md`

Do not redesign approved business rules unless a new requirement explicitly supersedes them.
