# Project Progress

## Current Phase
Direct purchase (`SHORTAGE` + `ADDITIONAL_REQUIREMENT`) is implemented and undergoing GitHub Actions validation.

## Last Updated
2026-08-12

## Documentation Rule
- Update this file on every codebase change.
- Update `requirements.md`, `architecture.md`, `database.md`, `api.md`, or `decisions.md` whenever the corresponding area is changed.
- The approved `rapat` meeting document remains the accepted business-flow reference and is not modified by routine implementation work.

## Completed
- Foundation, menu, inventory, procurement, PO generation, H-1 cancellation, and receiving implemented.
- Receiving batch is green through GitHub Actions run #33.
- `migrations/000007_create_direct_purchase.sql` added.
- Direct purchase types implemented: `SHORTAGE`, `ADDITIONAL_REQUIREMENT`.
- `SHORTAGE` flow locks the PO item and calculates remaining shortage from ordered qty minus cumulative vendor receipts minus prior shortage purchases.
- `SHORTAGE` quantity cannot exceed remaining shortage; excess maps to `422`.
- Shortage direct purchase creates stock `IN` + `SHORTAGE_PURCHASE` movement atomically.
- A shortage purchase that exactly closes remaining shortage marks the PO item `FULFILLED` and recalculates PO header status.
- Additional-production demand is stored separately in `additional_requirements` / `additional_requirement_items` and does not rewrite original procurement snapshots.
- Additional requirement flow locks the scheduled menu before resolving the current effective portion count.
- Current effective portions use the latest additional requirement, falling back to original scheduled-menu planned portions.
- Additional material quantities are calculated server-side from scheduled-menu snapshot recipe × portion delta.
- Client supplies one price per calculated additional material; quantities are not client-controlled.
- Additional direct purchase creates stock `IN` + `ADDITIONAL_REQUIREMENT` movements atomically.
- Direct-purchase read/list endpoints added.
- `docs/api.md`, `docs/database.md`, and `docs/decisions.md` synchronized with the direct-purchase design.

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
- Direct purchase stock changes use the same inventory locking + movement ledger path as receiving.
- `total_amount` for direct purchase items is PostgreSQL-generated from qty × unit price.
- Original procurement gross/net values are historical and remain unchanged by production increases after PO.

## Validation Status
Receiving: GREEN through GitHub Actions run #33.

Direct purchase batch: latest CI run in progress. Canonical checks:
- PostgreSQL 17 startup
- Goose migration up
- rollback-to-zero
- migration re-apply
- `sqlc generate`
- `go test ./...`
- `go build ./...`
- generated sqlc synchronization

## In Progress
- Resolve any sqlc/generated-type or compile mismatch from the direct-purchase CI batch until green.

## Next
1. Finish direct purchase CI validation.
2. Implement `000008_material_usage.sql`.
3. Add actual material usage draft/submit flow.
4. Add Chef + Accountant versioned approvals.
5. Apply stock OUT only after both approvals for the current version.
6. Consume/release associated active reservations when approved usage is applied.
7. Enforce negative-stock rejection atomically.
8. Add focused concurrency/transaction tests.
9. Add menu-template update flow without mutating historical scheduled-menu snapshots.

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
