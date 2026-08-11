# Project Progress

## Current Phase
Foundation, Menu, Inventory, and Procurement database layer implemented and executable validation is green in GitHub Actions.

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
- GitHub Actions CI workflow added at `.github/workflows/ci.yml`.
- CI compatibility corrected to use Go `1.26.x`, Goose `v3.27.1`, and sqlc `v1.31.1`.
- CI migration execution passed against PostgreSQL 17.
- CI migration rollback to version 0 and re-apply passed.
- `sqlc generate` passed in CI.
- `go test ./...` passed in CI.
- `go build ./...` passed in CI.
- sqlc generated code was automatically committed to `main` by `github-actions[bot]` with commit `d3d3f0028ee02b99747a2d84eff55b1b8f69440c`.
- `docs/architecture.md` synchronized with the working CI validation strategy.
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
GREEN through GitHub Actions.

Latest complete validation includes:
- PostgreSQL 17 service startup.
- Goose migration `up`.
- Goose rollback `down-to 0`.
- Goose migration `up` again.
- `sqlc generate`.
- `go test ./...`.
- `go build ./...`.

The ChatGPT execution sandbox itself still does not have direct outbound internet/DNS access. This is intentionally bypassed by using the GitHub connector for repository operations and GitHub Actions for executable validation.

## In Progress
- Start HTTP/service/repository implementation for material, period, menu template, and scheduled-menu flows.

## Next
1. Implement material repository/service/handler and routes.
2. Implement period repository/service/handler and routes.
3. Implement menu-template repository/service/handler and routes.
4. Implement scheduled-menu snapshot transaction.
5. Implement procurement stock-check + reservation transaction.
6. Add tests around reservation concurrency and negative-stock invariants.
7. Continue with `000005_receiving.sql` after the current application flow is stable.

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
- H-1 PO-item cancellation timing is enforced in the service transaction because it depends on the parent PO delivery date.
- Routine development is performed directly on `main` while repository rules permit direct writes.
- GitHub Actions is the canonical executable validation environment for migrations/sqlc/build checks initiated from this workflow.
- CI uses Go `1.26.x` for current tooling compatibility while the application minimum remains Go `1.25.0` in `go.mod`.

## Changed Files (Latest Batch)
- `.github/workflows/ci.yml`
- `internal/database/generated/*` (regenerated by CI)
- `docs/architecture.md`
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
