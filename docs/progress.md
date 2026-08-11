# Project Progress

## Current Phase
Direct purchase (`SHORTAGE` + `ADDITIONAL_REQUIREMENT`) is implemented and green in GitHub Actions. Next focus is material usage + dual approval + stock OUT.

## Last Updated
2026-08-12

## Documentation Rule
- Update this file on every codebase change.
- Update `requirements.md`, `architecture.md`, `database.md`, `api.md`, or `decisions.md` whenever the corresponding area is changed.
- The approved `rapat` meeting document remains the accepted business-flow reference and is not modified by routine implementation work.

## Completed
- Foundation, menu, inventory, procurement, PO generation, H-1 cancellation, receiving, and direct purchase implemented.
- Receiving batch is green through GitHub Actions run #33.
- `migrations/000007_create_direct_purchase.sql` implemented.
- Direct purchase types implemented: `SHORTAGE`, `ADDITIONAL_REQUIREMENT`.
- `SHORTAGE` locks the PO item and calculates remaining shortage from ordered qty minus cumulative vendor receipts minus prior shortage purchases.
- `SHORTAGE` quantity cannot exceed remaining shortage; excess maps to `422`.
- Shortage direct purchase creates stock `IN` + `SHORTAGE_PURCHASE` movement atomically.
- A shortage purchase that exactly closes remaining shortage marks the PO item `FULFILLED` and recalculates PO header status.
- Additional-production demand is stored separately and does not rewrite original procurement snapshots.
- Additional requirement flow locks the scheduled menu before resolving current effective portions.
- Current effective portions use the latest additional requirement, falling back to original planned portions.
- Additional material quantities are calculated server-side from scheduled-menu snapshot recipe × portion delta.
- Client supplies one price per calculated material; quantities are not client-controlled.
- Additional direct purchase creates stock `IN` + `ADDITIONAL_REQUIREMENT` movements atomically.
- Direct-purchase detail/list endpoints implemented.
- Direct purchase batch passed GitHub Actions run #44: migrations, rollback/re-apply, `sqlc generate`, `go test ./...`, `go build ./...`, and generated-code synchronization all succeeded.

## Implemented Migrations
- `migrations/000001_create_foundation.sql`
- `migrations/000002_create_menu.sql`
- `migrations/000003_create_inventory.sql`
- `migrations/000004_create_procurement.sql`
- `migrations/000005_purchase_order_constraints.sql`
- `migrations/000006_create_receiving.sql`
- `migrations/000007_create_direct_purchase.sql`

## Newly Implemented HTTP Endpoints
- `POST /api/v1/purchase-order-items/:id/direct-purchases/shortage`
- `POST /api/v1/scheduled-menus/:id/direct-purchases/additional-requirement`
- `GET /api/v1/direct-purchases/:id`
- `GET /api/v1/scheduled-menus/:id/direct-purchases`

Existing master/menu/procurement/PO/receiving endpoints remain implemented as documented in `docs/api.md`.

## Important Implementation Notes
- Authentication is not implemented; `purchased_by` temporarily comes from the request body.
- Quantities/prices are PostgreSQL `NUMERIC`; API uses string values for decimal quantities/prices.
- Direct purchase stock changes use inventory locking + movement ledger.
- `total_amount` for direct purchase items is PostgreSQL-generated from qty × unit price.
- Original procurement gross/net values remain historical and unchanged by later production increases.

## Validation Status
GREEN through GitHub Actions run #44.

Validated checks:
- PostgreSQL 17 startup
- Goose migration up
- rollback-to-zero
- migration re-apply
- `sqlc generate`
- `go test ./...`
- `go build ./...`
- generated sqlc synchronization

## In Progress
- Prepare material usage schema and dual-approval application flow.

## Next
1. Implement `000008_material_usage.sql`.
2. Add actual material usage draft/submit flow.
3. Add Chef + Accountant versioned approvals.
4. Apply stock OUT only after both approvals for the current version.
5. Consume/release associated active reservations when approved usage is applied.
6. Enforce negative-stock rejection atomically.
7. Add focused concurrency/transaction tests.
8. Add menu-template update flow without mutating historical scheduled-menu snapshots.
9. Continue to stock opname and dual-approved adjustment after usage is stable.

## Blockers / TBD
- `KEPALA_SPPG` operational permissions remain TBD.
- Authentication/authorization is not implemented yet.
- Actual file-storage provider for receipt documents is not implemented yet.
- No application payment workflow is required.

## Latest Decisions
- Vendor/source is transaction data, not master data.
- SHORTAGE uses cumulative remaining shortage and cannot exceed it.
- Fully covered shortage produces `FULFILLED`.
- ADDITIONAL_REQUIREMENT quantities are server-calculated from snapshot recipe and delta portions.
- Additional requirements do not mutate the original procurement snapshot.
- All positive direct-purchase quantities enter stock atomically with a stock movement.
- GitHub Actions is the canonical executable validation environment.

## Changed Files (Latest Batch)
- `migrations/000007_create_direct_purchase.sql`
- `internal/database/query/menu.sql`
- `internal/database/query/direct_purchase.sql`
- `internal/database/generated/*`
- `internal/repository/store.go`
- `internal/service/direct_purchase_service.go`
- `internal/handler/http/direct_purchase_handler.go`
- `internal/handler/http/core_handler.go`
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
