# Project Progress

## Current Phase
Core database layer and phase-1/2 HTTP flows are implemented and green in GitHub Actions. Next focus is procurement stock-check and stock reservation.

## Last Updated
2026-08-12

## Documentation Rule
- Update this file on every codebase change.
- Update `requirements.md`, `architecture.md`, `database.md`, `api.md`, or `decisions.md` whenever the corresponding area is changed.
- The approved `rapat` meeting document is treated as the business-flow reference and is not modified from this repository workflow.

## Completed
- Business requirement V1 discussed and approved.
- Repository documentation structure established on `main`.
- Foundation, menu, inventory, and procurement migrations implemented.
- Foundation/menu/inventory/procurement sqlc query sources implemented.
- GitHub Actions CI added with PostgreSQL 17, Go `1.26.x`, Goose `v3.27.1`, and sqlc `v1.31.1`.
- Migration up, rollback-to-zero, re-apply, sqlc generation, tests, and build pass in CI.
- sqlc generated code is automatically synchronized by GitHub Actions.
- Core repository store implemented at `internal/repository/store.go`.
- Core service implemented at `internal/service/core_service.go`.
- Core Fiber handlers/routes implemented at `internal/handler/http/core_handler.go`.
- `cmd/api/main.go` wires repository -> service -> HTTP handler.
- Unit listing endpoint implemented.
- Material list/detail/create/update endpoints implemented.
- Period list/create endpoints implemented; service derives the fixed 14-day end date.
- Menu-template list/detail/create endpoints implemented.
- Menu-template creation is transactional for template + components + materials.
- Scheduled-menu create/detail endpoints implemented.
- Scheduled-menu creation validates date inside period and transactionally snapshots template components/materials.
- Core API batch passed GitHub Actions run #8 including migrations, sqlc generation, tests, and build.
- `docs/api.md` synchronized with the implemented HTTP surface.

## Implemented Migrations
- `migrations/000001_create_foundation.sql`
- `migrations/000002_create_menu.sql`
- `migrations/000003_create_inventory.sql`
- `migrations/000004_create_procurement.sql`

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

## Important Implementation Notes
- Authentication is not implemented yet, so `created_by` / `updated_by` remain nullable instead of using fabricated audit identities.
- `qty_per_portion` is accepted as a JSON string and parsed to PostgreSQL `NUMERIC` to avoid float precision loss.
- Scheduled menu data is copied from the selected template inside one database transaction.
- The original starter migration was rewritten during initial development; old local databases that ran it must be reset/recreated.

## Validation Status
GREEN through GitHub Actions.

Latest core API validation passed:
- PostgreSQL 17 startup.
- Goose migration `up`.
- Goose rollback `down-to 0`.
- Goose migration `up` again.
- `sqlc generate`.
- `go test ./...`.
- `go build ./...`.

## In Progress
- Design and implement procurement stock-check + stock-reservation transaction.

## Next
1. Implement procurement gross requirement calculation from scheduled-menu snapshots.
2. Lock material stock and active reservations while allocating existing stock.
3. Persist procurement request items with gross, existing, reserved, and net procurement quantities.
4. Create/update stock reservations atomically with the procurement stock check.
5. Expose procurement request detail/submit/verify/reject endpoints.
6. Generate PO from verified net procurement.
7. Add tests around reservation concurrency and negative-stock invariants.
8. Add menu-template update flow without mutating historical scheduled-menu snapshots.
9. Continue with `000005_receiving.sql` after procurement application flow is stable.

## Blockers / TBD
- `KEPALA_SPPG` exists as a role, but operational permissions are still TBD.
- Authentication/authorization is not implemented yet.
- No application payment workflow is required.

## Latest Decisions
- Vendor is not master data; supplier/source name is stored per transaction/item.
- Fixed unit price is agreed before PO and does not change at receipt.
- Vendor over-delivery is accepted in full.
- Shortage direct purchase cannot exceed remaining shortage.
- Additional portions after PO use direct purchase type `ADDITIONAL_REQUIREMENT`, not a new PO.
- Wrong material delivery is rejected; no substitution flow.
- Material quality rejection is outside V1 scope.
- No FIFO or stock-batch tracking in V1.
- No negative stock.
- Approval becomes invalid when approved data is revised; resubmission requires fresh approval.
- `material_stocks` is a fast snapshot; `stock_movements` is the inventory audit source of truth.
- Stock reservation is not a stock movement because it does not physically change quantity.
- H-1 PO-item cancellation timing is enforced in the service transaction because it depends on the parent PO delivery date.
- GitHub Actions is the canonical executable validation environment.
- CI uses Go `1.26.x` for tooling compatibility while application minimum remains Go `1.25.0` in `go.mod`.

## Changed Files (Latest Batch)
- `internal/repository/store.go`
- `internal/service/core_service.go`
- `internal/handler/http/core_handler.go`
- `cmd/api/main.go`
- `docs/api.md`
- `docs/progress.md`

## Notes for Next Agent
Read these before changing behavior or schema:
1. `docs/progress.md`
2. `docs/requirements.md`
3. `docs/architecture.md`
4. `docs/database.md`
5. `docs/decisions.md`
6. `docs/api.md` when touching HTTP contracts.

Do not redesign an approved business rule merely because another design looks cleaner. Change agreed behavior only when a new requirement explicitly supersedes it.
