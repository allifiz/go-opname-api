# Project Progress

## Current Phase
Core inventory V1 is implemented through stock opname and dual-approved stock adjustment. Latest stock-opname CI validation is in progress.

## Last Updated
2026-08-12

## Documentation Rule
- Update this file on every codebase change.
- Update domain docs when their area changes.
- The approved `rapat` document remains read-only business-flow reference.

## Completed
- Foundation, menu, inventory, procurement, PO, H-1 cancellation, receiving, direct purchase, and material usage implemented.
- Material usage batch is green through GitHub Actions run #55.
- `000009_create_stock_opname.sql` added.
- Stock opname is unique per scheduled menu and snapshots current `material_stocks.qty` as `system_qty`.
- Opname input material set must exactly match the scheduled-menu material set in V1.
- `difference_qty` is PostgreSQL-generated as `physical_qty - system_qty`.
- A matching material creates no adjustment. A differing material creates a DRAFT stock adjustment; stock is not changed at opname time.
- Adjustment quantity is always derived from the opname difference and is never client-controlled.
- Revising an adjustment changes `physical_qty` + reason, recalculates difference, increments adjustment `version`, and preserves historical approvals.
- Chef + Accountant adjustment approvals are unique by adjustment + role + entity version.
- Only active `CHEF` and `AKUNTAN` users may decide.
- Rejection moves an adjustment to `NEEDS_REVISION` without inventory effects.
- The second current-version approval atomically applies `ADJUSTMENT_IN` or `ADJUSTMENT_OUT`, creates `STOCK_ADJUSTMENT` movement, and marks the adjustment approved.
- `ADJUSTMENT_OUT` uses row locking + sufficient-stock predicate, so negative stock is rejected atomically.
- Stock opname becomes `COMPLETED` after all differing-item adjustments are approved.
- Stock opname HTTP routes are wired in `cmd/api/main.go`.

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

## Newly Implemented HTTP Endpoints
- `POST /api/v1/scheduled-menus/:id/stock-opname`
- `GET /api/v1/stock-opnames/:id`
- `GET /api/v1/stock-adjustments/:id`
- `PUT /api/v1/stock-adjustments/:id`
- `POST /api/v1/stock-adjustments/:id/submit`
- `POST /api/v1/stock-adjustments/:id/decision`

Existing endpoints remain documented in `docs/api.md`.

## Validation Status
Previous stable batch: GREEN through GitHub Actions run #55.

Stock opname batch: latest CI run in progress. Canonical checks remain:
- PostgreSQL 17 startup
- Goose migration up
- rollback-to-zero
- migration re-apply
- `sqlc generate`
- `go test ./...`
- `go build ./...`
- generated sqlc synchronization

## Next After Core V1 Is Green
1. Add focused transaction/concurrency integration tests for reservation, receipt, usage, and adjustment races.
2. Implement authentication/JWT and RBAC so audit user IDs no longer come from request bodies.
3. Add menu-template update flow without mutating historical scheduled-menu snapshots.
4. Add OpenAPI/Swagger and consistent response/error contracts.
5. Add pagination/filtering where list volume can grow.
6. Implement actual object storage for receipt documents.
7. Resolve `KEPALA_SPPG` operational permissions.

## Blockers / TBD
- `KEPALA_SPPG` operational permissions remain TBD.
- Authentication/authorization is not implemented yet.
- Receipt document object storage provider is not implemented yet.
- No application payment workflow is required.

## Changed Files (Latest Batch)
- `migrations/000009_create_stock_opname.sql`
- `internal/database/query/stock_opname.sql`
- `internal/repository/store.go`
- `internal/service/stock_opname_service.go`
- `internal/handler/http/stock_opname_handler.go`
- `cmd/api/main.go`
- `docs/progress.md`
- `docs/api.md`
- `docs/database.md`
- `docs/decisions.md`

## Notes for Next Agent
Read before changing behavior/schema:
1. `docs/progress.md`
2. `docs/requirements.md`
3. `docs/architecture.md`
4. `docs/database.md`
5. `docs/decisions.md`
6. `docs/api.md`

Do not redesign approved business rules unless a new requirement explicitly supersedes them.
