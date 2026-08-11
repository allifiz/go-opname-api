# Project Progress

## Current Phase
Procurement stock-check, PO generation, and H-1 item cancellation are implemented and green in GitHub Actions. Next focus is receiving.

## Last Updated
2026-08-12

## Documentation Rule
- Update this file on every codebase change.
- Update `requirements.md`, `architecture.md`, `database.md`, `api.md`, or `decisions.md` whenever the corresponding area is changed.
- The approved `rapat` meeting document is the accepted business-flow reference and is not modified by routine implementation work.

## Completed
- Foundation, menu, inventory, procurement schema and sqlc query layers implemented.
- Core master/menu/scheduled-menu HTTP flows implemented.
- Scheduled-menu snapshot and gross-requirement calculation implemented.
- Procurement stock-check transaction implemented with material-stock locking and active reservation deduction.
- One procurement request per scheduled menu enforced.
- Procurement submit / verify / reject lifecycle implemented.
- Purchase-order generation implemented from `VERIFIED` procurement requests.
- PO item `ordered_qty` is copied server-side from positive `net_procurement_qty`; the client cannot override quantity.
- Supplier name and fixed agreed unit price are captured per PO item.
- Server-generated PO number uses `PO-YYYYMMDD-XXXXXXXXXX`.
- One purchase order per procurement request is enforced by `000005_purchase_order_constraints.sql`.
- PO detail and scheduled-menu PO listing endpoints implemented.
- H-1 PO-item cancellation implemented transactionally.
- H-1 cancellation locks the PO item and material stock, verifies currently-unreserved stock, creates replacement reservation, then cancels the PO item.
- H-1 cancellation reason is `EXISTING_STOCK_SUFFICIENT`.
- Insufficient replacement stock maps to HTTP `422`; missed cancellation deadline / invalid state maps to `409`.
- CI auto-generated sqlc push now rebases onto current `main` before pushing, preventing non-fast-forward failures when docs or other commits land during the run.
- PO/H-1 batch passed GitHub Actions run #25: migration up, rollback-to-zero, re-apply, `sqlc generate`, `go test ./...`, `go build ./...`, and generated-code commit all succeeded.

## Implemented Migrations
- `migrations/000001_create_foundation.sql`
- `migrations/000002_create_menu.sql`
- `migrations/000003_create_inventory.sql`
- `migrations/000004_create_procurement.sql`
- `migrations/000005_purchase_order_constraints.sql`

## Implemented HTTP Endpoints
- `GET /health`
- `GET /api/v1/units`
- `GET /api/v1/materials`
- `GET /api/v1/materials/:id`
- `POST /api/v1/materials`
- `PUT /api/v1/materials/:id`
- `GET /api/v1/periods`
- `POST /api/v1/periods`
- `GET /api/v1/menu-templates`
- `GET /api/v1/menu-templates/:id`
- `POST /api/v1/menu-templates`
- `GET /api/v1/scheduled-menus/:id`
- `POST /api/v1/scheduled-menus`
- `POST /api/v1/scheduled-menus/:id/procurement-stock-check`
- `GET /api/v1/scheduled-menus/:id/procurement-requests`
- `GET /api/v1/procurement-requests/:id`
- `POST /api/v1/procurement-requests/:id/submit`
- `POST /api/v1/procurement-requests/:id/verify`
- `POST /api/v1/procurement-requests/:id/reject`
- `POST /api/v1/procurement-requests/:id/purchase-order`
- `GET /api/v1/purchase-orders/:id`
- `GET /api/v1/scheduled-menus/:id/purchase-orders`
- `POST /api/v1/purchase-order-items/:id/cancel-h1`

## Important Implementation Notes
- Authentication is not implemented; `verified_by` and `cancelled_by` temporarily come from request bodies where a non-null audit FK is required.
- Persisted quantities/prices use PostgreSQL `NUMERIC`; JSON quantity/price inputs are strings to avoid float precision loss.
- H-1 date evaluation currently uses `Asia/Jakarta` business time.
- A replacement reservation created during H-1 cancellation equals the cancelled PO item's full `ordered_qty`. Combined with the original procurement reservation this protects the full scheduled-menu requirement.
- Purchase orders currently act as the scheduled-menu procurement document and may contain items from different suppliers because supplier identity is intentionally stored per item in V1.

## Validation Status
GREEN through GitHub Actions run #25.

Validated checks:
- PostgreSQL 17 startup.
- Goose migration `up`.
- Goose rollback `down-to 0`.
- Goose migration `up` again.
- `sqlc generate`.
- `go test ./...`.
- `go build ./...`.
- regenerated sqlc code committed after rebasing onto current `main`.

## In Progress
- Prepare receiving schema/application flow.

## Next
1. Add `000006_receiving.sql`.
2. Support multiple receipts per PO item and cumulative vendor receipt quantities.
3. Implement `NOT_RECEIVED`, `PARTIAL_RECEIVED`, `RECEIVED`, and `OVER_RECEIVED` based on cumulative received quantity.
4. Apply receipt + stock IN atomically.
5. Store invoice/supporting-document metadata for Accountant visibility.
6. Implement shortage and additional-requirement direct purchases.
7. Add focused transaction/concurrency tests.
8. Add menu-template update flow without mutating historical scheduled-menu snapshots.

## Blockers / TBD
- `KEPALA_SPPG` operational permissions remain TBD.
- Authentication/authorization is not implemented yet.
- No application payment workflow is required.

## Latest Decisions
- Vendor is not master data; supplier/source is per transaction/item.
- Fixed agreed unit price does not change at receipt.
- Gross requirement is not overwritten by net procurement.
- Stock reservation is allocation only, not a stock movement.
- One scheduled menu has one procurement-request lifecycle.
- One procurement request has at most one PO in V1.
- PO ordered quantity is server-controlled from verified net procurement.
- H-1 cancellation must reserve replacement stock in the same transaction before cancellation.
- H-1 cancellation is allowed only before the delivery date, with H-1 as the latest valid day.
- GitHub Actions is the canonical executable validation environment.

## Changed Files (Latest Batch)
- `migrations/000005_purchase_order_constraints.sql`
- `internal/database/query/procurement.sql`
- `internal/database/query/inventory.sql`
- `internal/database/generated/*`
- `internal/repository/store.go`
- `internal/service/purchase_order_service.go`
- `internal/handler/http/core_handler.go`
- `internal/handler/http/procurement_handler.go`
- `.github/workflows/ci.yml`
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
