# Project Progress

## Current Phase
Foundation, Menu, Inventory, and Procurement database layer implemented; local generator/runtime validation pending.

## Last Updated
2026-08-11

## Documentation Rule
- Update this file on every codebase change.
- Update `requirements.md`, `architecture.md`, `database.md`, `api.md`, or `decisions.md` whenever the corresponding area is changed.
- The approved `rapat` meeting document is treated as the business-flow reference and is not modified from this repository workflow.

## Completed
- Business requirement V1 discussed and approved.
- Repository documentation structure established on `main`.
- Starter foundation migration replaced with V1 foundation schema.
- `roles`, `users`, `units`, and `materials` implemented.
- Role seeds implemented: Chef, Ahli Gizi, Pengawas Bahan Baku, Akuntan, Kepala SPPG.
- Unit seeds implemented: KG, PCS, LT, IKAT, RENCENG, BOTOL.
- Menu template schema implemented.
- Scheduled-menu snapshot schema implemented.
- Two-week period constraint implemented at database level.
- Inventory current-stock snapshot implemented through `material_stocks`.
- Inventory audit ledger implemented through `stock_movements`.
- Stock reservation schema and lifecycle checks implemented.
- Negative material stock is blocked by database constraint.
- Foundation sqlc queries added.
- Menu/scheduled-menu sqlc queries added.
- Inventory/reservation sqlc queries added.
- Procurement request and PO schema implemented.
- Deferred FK from `stock_reservations.procurement_request_item_id` to `procurement_request_items.id` implemented.
- Procurement and PO sqlc queries added.
- Procurement quantity constraints and PO cancellation audit constraints implemented.
- `docs/database.md` synchronized with actual migration/query state.

## Implemented Migrations
- `migrations/000001_create_foundation.sql`
- `migrations/000002_create_menu.sql`
- `migrations/000003_create_inventory.sql`
- `migrations/000004_create_procurement.sql`

## Implemented sqlc Query Sources
- `internal/database/query/health.sql`
- `internal/database/query/foundation.sql`
- `internal/database/query/menu.sql`
- `internal/database/query/inventory.sql`
- `internal/database/query/procurement.sql`

## Important Migration Note
The original starter `000001_create_foundation.sql` was rewritten because the project is still in the initial build phase. A local development database that already ran the old starter migration must be recreated/reset before applying the new schema.

## Validation Status
Source-level review is complete for the newly added SQL.

Not yet validated in an executable local environment:
- Goose migration `up`/`down` on a clean PostgreSQL database.
- `sqlc generate`.
- `go test ./...` / `go build ./...` after regenerated database code.

The current agent runtime does not have `sqlc` or Goose installed and cannot resolve `github.com` to download them, so these checks must be run in the normal local development environment before service implementation is considered validated.

## In Progress
- Prepare clean local database validation.
- Generate sqlc output.
- Start HTTP/service/repository implementation for material and menu flows after generator validation passes.

## Next
1. Recreate/reset the local development database if it previously ran the old starter migration.
2. Run Goose migrations `000001` through `000004` against a clean PostgreSQL database.
3. Run `sqlc generate`.
4. Run `go test ./...` and/or `go build ./...`.
5. Fix any generator/schema mismatch found by those checks.
6. Implement material endpoints/service/repository.
7. Implement period and menu-template endpoints/service/repository.
8. Implement scheduled-menu snapshot transaction.
9. Implement procurement stock-check + reservation transaction.
10. Continue with `000005_receiving.sql` after the current DB layer is validated.

## Blockers / TBD
- `KEPALA_SPPG` exists as a role, but operational permissions are still TBD.
- No application payment workflow is required.
- Executable migration/sqlc validation must be run in a normal local development environment because this agent runtime cannot download the required CLI tools.

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
- Routine development is performed directly on `main` while repository rules permit direct writes.

## Changed Files (Latest Batch)
- `internal/database/query/foundation.sql`
- `internal/database/query/menu.sql`
- `internal/database/query/inventory.sql`
- `migrations/000004_create_procurement.sql`
- `internal/database/query/procurement.sql`
- `docs/database.md`
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
