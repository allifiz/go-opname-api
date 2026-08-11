# Project Progress

## Current Phase
Core database schema implemented for Foundation, Menu, and Inventory.

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
- Stock movement quantity must be positive.
- Stock reservation quantity must be positive.
- `docs/database.md` synchronized with actual migrations.

## Implemented Migrations
- `migrations/000001_create_foundation.sql`
- `migrations/000002_create_menu.sql`
- `migrations/000003_create_inventory.sql`

## Important Migration Note
The original starter `000001_create_foundation.sql` was rewritten because the project is still in the initial build phase. A local development database that already ran the old starter migration must be recreated/reset before applying the new schema.

## In Progress
- Prepare sqlc queries for foundation/material/menu/inventory reads and writes.
- Prepare procurement migration.

## Next
1. Add sqlc queries for roles, units, materials, periods, menu templates, scheduled menus, and stock reads.
2. Add `000004_create_procurement.sql`.
3. Add the deferred FK from `stock_reservations.procurement_request_item_id` to `procurement_request_items.id`.
4. Run Goose migrations against a clean local database.
5. Run `sqlc generate`.
6. Run `go test ./...` / build validation.
7. Implement material and menu services/endpoints.
8. Continue receiving/direct-purchase workflow after procurement is stable.

## Blockers / TBD
- `KEPALA_SPPG` exists as a role, but operational permissions are still TBD.
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
- Routine development is performed directly on `main` while repository rules permit direct writes.

## Changed Files
- `migrations/000001_create_foundation.sql`
- `migrations/000002_create_menu.sql`
- `migrations/000003_create_inventory.sql`
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
