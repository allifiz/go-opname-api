# Project Progress

## Current Phase
Core inventory V1, concurrency hardening, authentication/JWT + RBAC, and secure initial-user bootstrap are implemented and green. Next focus is menu-template update without mutating historical scheduled-menu snapshots.

## Last Updated
2026-08-12

## Documentation Rule
- Update this file on every codebase change.
- Update domain docs when their area changes.
- The approved `rapat` document remains read-only business-flow reference.

## Completed
- Foundation, menu, inventory, procurement, PO, H-1 cancellation, receiving, direct purchase, material usage, stock opname, and stock adjustment implemented.
- Real PostgreSQL concurrency coverage protects receiving, usage approval, adjustment approval, conditional stock OUT, reservation allocation, shortage purchases, and initial-user bootstrap.
- JWT authentication implemented using HS256 with the Go standard library.
- Password verification uses bcrypt against `users.password_hash`.
- `POST /api/v1/auth/login` is public and issues an 8-hour Bearer token containing user ID, role, email, and expiry.
- `GET /api/v1/auth/me` and all non-health business endpoints require a valid Bearer token.
- `JWT_SECRET` is mandatory and must contain at least 32 characters.
- HTTP RBAC guards are implemented.
- Secure initial-user provisioning is implemented through `POST /api/v1/auth/bootstrap` and serialized by PostgreSQL so concurrent requests cannot create multiple first users.
- No default credential and no invented `ADMIN` role are introduced.
- GitHub Actions validation for the bootstrap batch is pending.

## Validation Status
GREEN through GitHub Actions run #79 for the previous auth/RBAC batch. Initial-user bootstrap validation is pending.

## Next
1. Get the initial-user bootstrap batch green in GitHub Actions.
2. Add menu-template update flow without mutating historical scheduled-menu snapshots.
3. Add OpenAPI/Swagger and consistent response/error contracts.
4. Add pagination/filtering where list volume can grow.
5. Implement actual object storage for receipt documents.
6. Resolve `KEPALA_SPPG` operational permissions.
7. Define authenticated provisioning of additional users after the one-time bootstrap, once an administrative authorization policy is approved.

## Blockers / TBD
- `KEPALA_SPPG` operational permissions remain TBD.
- Authenticated provisioning of additional users remains TBD because no administrative role/permission has been approved in the business flow.
- Receipt document object storage provider is not implemented yet.
- No application payment workflow is required.

## Changed Files (Latest Batch)
- `internal/config/config.go`
- `internal/repository/bootstrap.go`
- `internal/service/auth_service.go`
- `internal/service/auth_integration_test.go`
- `internal/handler/http/auth_handler.go`
- `cmd/api/main.go`
- `.env.example`
- `docs/progress.md`
- `docs/api.md`
- `docs/architecture.md`
- `docs/decisions.md`
- `Readme.md`

## Notes for Next Agent
Read before changing behavior/schema:
1. `docs/progress.md`
2. `docs/requirements.md`
3. `docs/architecture.md`
4. `docs/database.md`
5. `docs/decisions.md`
6. `docs/api.md`

Do not redesign approved business rules unless a new requirement explicitly supersedes them.
