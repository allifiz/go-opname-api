# Project Progress

## Current Phase
Core inventory V1 is implemented and green through stock opname and dual-approved stock adjustment. Concurrency hardening now covers receiving, usage approval, adjustment approval, stock-out competition, reservation allocation, and shortage direct purchase.

## Last Updated
2026-08-12

## Documentation Rule
- Update this file on every codebase change.
- Update domain docs when their area changes.
- The approved `rapat` document remains read-only business-flow reference.

## Completed
- Foundation, menu, inventory, procurement, PO, H-1 cancellation, receiving, direct purchase, material usage, stock opname, and stock adjustment implemented.
- Core stock-opname batch passed GitHub Actions run #62 including migrations, rollback/re-apply, `sqlc generate`, tests, build, and generated-code synchronization.
- Added `internal/service/concurrency_integration_test.go` using the real PostgreSQL database provided by CI rather than mocks.
- Concurrent receiving for the same PO item is covered and serializes correctly, with cumulative stock/status applied once per receipt.
- Concurrent Chef + Accountant material-usage approvals are covered and apply stock OUT exactly once.
- Concurrent Chef + Accountant stock-adjustment approvals are covered and apply adjustment movement exactly once.
- Competing conditional stock OUT is covered: when stock only satisfies one request, exactly one decrement succeeds and stock never becomes negative.
- GitHub Actions run #63 passed migration up, rollback/re-apply, `sqlc generate`, all initial integration/concurrency tests via `go test ./...`, build, and generated-code synchronization.
- Added concurrent procurement stock-check coverage across two scheduled menus competing for the same physical material stock. The test requires active reservations never to exceed actual stock and aggregate net procurement to absorb the unavailable allocation.
- Added concurrent `SHORTAGE` direct-purchase coverage for two requests competing for the same remaining shortage. PO-item locking must allow only the quantity that fits the remaining shortage; the competing excess request must fail without adding stock or purchase quantity.

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

## Validation Status
Previously GREEN through GitHub Actions run #63.

Reservation + shortage race hardening batch is awaiting the latest GitHub Actions result.

Canonical checks:
- PostgreSQL 17 startup
- Goose migration up
- rollback-to-zero
- migration re-apply
- `sqlc generate`
- real PostgreSQL integration/concurrency tests through `go test ./...`
- `go build ./...`
- generated sqlc synchronization

## Next
1. Get reservation/shortage concurrency hardening green and fix any race surfaced by CI.
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
- `internal/service/concurrency_integration_test.go`
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
