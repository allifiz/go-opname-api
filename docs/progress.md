# Project Progress

## Current Phase
Material usage, versioned Chef + Accountant approval, and atomic stock OUT are implemented. Latest GitHub Actions validation is in progress.

## Last Updated
2026-08-12

## Documentation Rule
- Update this file on every codebase change.
- Update `requirements.md`, `architecture.md`, `database.md`, `api.md`, or `decisions.md` whenever the corresponding area is changed.
- The approved `rapat` meeting document remains the accepted business-flow reference and is not modified by routine implementation work.

## Completed
- Foundation, menu, inventory, procurement, PO generation, H-1 cancellation, receiving, and direct purchase implemented.
- Direct purchase batch is green through GitHub Actions run #44.
- `migrations/000008_create_material_usage.sql` added.
- Material usage has one lifecycle per scheduled menu with statuses `DRAFT`, `WAITING_APPROVAL`, `APPROVED`, `NEEDS_REVISION`.
- Usage items store server-derived `planned_qty` and user-entered `actual_qty`.
- Usage planned quantity uses the latest effective production portions, including later `ADDITIONAL_REQUIREMENT` increases.
- Usage revisions are allowed in `DRAFT` / `NEEDS_REVISION`, increment `version`, replace current usage items, and preserve historical approval rows.
- Chef + Accountant decisions are versioned and unique by usage + role + entity version.
- Approver identity is resolved against the users/roles tables; only active `CHEF` or `AKUNTAN` users may decide.
- Rejection moves usage to `NEEDS_REVISION` without changing stock.
- The second current-version approval triggers stock application atomically.
- Material stock is locked before decrement and SQL rejects decrement when stock is insufficient.
- Positive usage creates `OUT / MATERIAL_USAGE` stock movements.
- Active stock reservations for the scheduled menu/material are marked `CONSUMED` when approved usage is applied.
- If any material lacks stock, the whole second-approval transaction rolls back, including approval insertion and any earlier stock effects in that transaction.
- Material usage HTTP routes are wired in `cmd/api/main.go`.
- `docs/api.md`, `docs/database.md`, and `docs/decisions.md` synchronized with the usage design.

## Implemented Migrations
- `migrations/000001_create_foundation.sql`
- `migrations/000002_create_menu.sql`
- `migrations/000003_create_inventory.sql`
- `migrations/000004_create_procurement.sql`
- `migrations/000005_purchase_order_constraints.sql`
- `migrations/000006_create_receiving.sql`
- `migrations/000007_create_direct_purchase.sql`
- `migrations/000008_create_material_usage.sql`

## Newly Implemented HTTP Endpoints
- `POST /api/v1/scheduled-menus/:id/material-usage`
- `GET /api/v1/material-usages/:id`
- `PUT /api/v1/material-usages/:id`
- `POST /api/v1/material-usages/:id/submit`
- `POST /api/v1/material-usages/:id/decision`

Existing master/menu/procurement/PO/receiving/direct-purchase endpoints remain implemented as documented in `docs/api.md`.

## Important Implementation Notes
- Authentication is not implemented; `submitted_by` and `approver_id` temporarily come from request bodies.
- Quantities remain PostgreSQL `NUMERIC`; API decimal quantities are strings.
- Approval history is retained across revisions; old entity versions simply stop counting toward current approval.
- Usage application consumes scheduled-menu reservations after applying actual stock usage.
- Zero actual usage creates no stock movement but still consumes the reservation for that scheduled-menu/material.
- Negative stock is prevented both by row locking and an atomic `qty >= subtract_qty` SQL predicate.

## Validation Status
Previous stable batch: GREEN through GitHub Actions run #44.

Material usage batch: latest CI run pending/in progress. Canonical checks:
- PostgreSQL 17 startup
- Goose migration up
- rollback-to-zero
- migration re-apply
- `sqlc generate`
- `go test ./...`
- `go build ./...`
- generated sqlc synchronization

## In Progress
- Resolve any generated-type/compile issue from material usage until CI is green.

## Next
1. Get material usage batch green.
2. Implement `000009_stock_opname.sql`.
3. Add post-production physical stock capture and server-calculated difference.
4. Create adjustment request only when a difference exists; do not auto-change stock.
5. Add Chef + Accountant versioned approval for stock adjustment.
6. Apply approved `ADJUSTMENT_IN` / `ADJUSTMENT_OUT` atomically with negative-stock protection.
7. Add focused concurrency/transaction tests.
8. Add menu-template update flow without mutating historical scheduled-menu snapshots.
9. Add authentication/RBAC after core inventory cycle is complete.

## Blockers / TBD
- `KEPALA_SPPG` operational permissions remain TBD.
- Authentication/authorization is not implemented yet.
- Actual file-storage provider for receipt documents is not implemented yet.
- No application payment workflow is required.

## Latest Decisions
- Material usage is one versioned lifecycle per scheduled menu.
- Planned usage follows effective portions, including additional requirements.
- Usage requires both Chef and Accountant approval for the same entity version.
- Revision preserves old approvals but invalidates them by incrementing version.
- Rejection returns usage to `NEEDS_REVISION` without inventory effects.
- Second valid approval, stock OUT, ledger movement, reservation consumption, and final approval status are atomic.
- Negative stock is prohibited.
- GitHub Actions remains the canonical executable validation environment.

## Changed Files (Latest Batch)
- `migrations/000008_create_material_usage.sql`
- `internal/database/query/foundation.sql`
- `internal/database/query/menu.sql`
- `internal/database/query/inventory.sql`
- `internal/database/query/material_usage.sql`
- `internal/repository/store.go`
- `internal/service/material_usage_service.go`
- `internal/handler/http/material_usage_handler.go`
- `cmd/api/main.go`
- `docs/api.md`
- `docs/database.md`
- `docs/decisions.md`
- `docs/progress.md`

## Notes for Next Agent
Read before changing behavior/schema:
1. `docs/progress.md`
2. `docs/requirements.md`
3. `docs/architecture.md`
4. `docs/database.md`
5. `docs/decisions.md`
6. `docs/api.md`

Do not redesign approved business rules unless a new requirement explicitly supersedes them.
