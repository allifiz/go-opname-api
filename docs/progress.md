# Project Progress

## Current Phase
Core inventory V1, concurrency hardening, and authentication/JWT + RBAC are implemented and green. Next focus is secure user provisioning and API/product hardening.

## Last Updated
2026-08-12

## Documentation Rule
- Update this file on every codebase change.
- Update domain docs when their area changes.
- The approved `rapat` document remains read-only business-flow reference.

## Completed
- Foundation, menu, inventory, procurement, PO, H-1 cancellation, receiving, direct purchase, material usage, stock opname, and stock adjustment implemented.
- Real PostgreSQL concurrency coverage protects receiving, usage approval, adjustment approval, conditional stock OUT, reservation allocation, and shortage purchases.
- Concurrency hardening is green through GitHub Actions run #64.
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
- Operational audit actors that previously came from request bodies are overwritten at the HTTP boundary from JWT identity (`verified_by`, `cancelled_by`, `received_by`, `purchased_by`, usage `submitted_by`/`approver_id`, opname `performed_by`, adjustment `submitted_by`/`approver_id`).
- Authentication tests cover bcrypt login against PostgreSQL, valid token claims, tampered token rejection, expired token rejection, missing-token `401`, wrong-role `403`, and allowed-role `200`.
- GitHub Actions run #79 passed PostgreSQL startup, migrations, rollback/re-apply, `sqlc generate`, all integration/concurrency/auth tests, build, and generated-code synchronization.

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

No new migration is required for JWT auth; the existing `users`, `roles`, and `password_hash` fields are used.

## Validation Status
GREEN through GitHub Actions run #79.

Validated checks:
- PostgreSQL 17 startup
- Goose migration up
- rollback-to-zero
- migration re-apply
- `sqlc generate`
- real PostgreSQL integration/concurrency/auth tests through `go test ./...`
- `go build ./...`
- generated sqlc synchronization

## Next
1. Add secure initial-user provisioning/admin workflow; do not seed a public default password in migrations.
2. Add menu-template update flow without mutating historical scheduled-menu snapshots.
3. Add OpenAPI/Swagger and consistent response/error contracts.
4. Add pagination/filtering where list volume can grow.
5. Implement actual object storage for receipt documents.
6. Resolve `KEPALA_SPPG` operational permissions.

## Blockers / TBD
- `KEPALA_SPPG` operational permissions remain TBD.
- Secure initial user provisioning is not yet implemented; no default credential is seeded intentionally.
- Receipt document object storage provider is not implemented yet.
- No application payment workflow is required.

## Changed Files (Latest Batch)
- `internal/config/config.go`
- `internal/database/query/foundation.sql`
- `internal/repository/store.go`
- `internal/service/auth_service.go`
- `internal/service/auth_integration_test.go`
- `internal/handler/http/auth_handler.go`
- `internal/handler/http/auth_handler_test.go`
- `internal/handler/http/core_handler.go`
- `internal/handler/http/procurement_handler.go`
- `internal/handler/http/receiving_handler.go`
- `internal/handler/http/direct_purchase_handler.go`
- `internal/handler/http/material_usage_handler.go`
- `internal/handler/http/stock_opname_handler.go`
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
