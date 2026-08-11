# Go Opname API

Backend Go untuk alur perencanaan menu, procurement, receiving, pemakaian bahan, stock opname, dan adjustment gudang.

## Tech stack

- Go 1.25+
- Fiber
- PostgreSQL
- pgx/v5
- Goose migrations
- sqlc
- JWT HS256
- bcrypt
- Air untuk hot reload

## Status project

Core inventory V1 sudah mencakup:

```text
Menu / Scheduled Menu
→ Procurement + Reservation
→ PO + H-1 Re-check
→ Receiving
→ SHORTAGE / ADDITIONAL_REQUIREMENT
→ Material Usage + Dual Approval
→ Stock OUT
→ Stock Opname
→ Stock Adjustment + Dual Approval
```

Concurrency-sensitive flow divalidasi dengan PostgreSQL asli di GitHub Actions. Authentication/JWT + RBAC juga sudah diterapkan pada HTTP API.

## Prasyarat

```powershell
git --version
go version
postgres --version
psql --version
```

Docker tidak wajib.

## Setup lokal

### 1. Clone

```powershell
git clone https://github.com/allifiz/go-opname-api.git
cd go-opname-api
```

Jika repository sudah ada:

```powershell
git checkout main
git pull origin main
```

### 2. Buat PostgreSQL user/database

```powershell
psql -U postgres
```

```sql
CREATE USER opname WITH PASSWORD 'opname';
CREATE DATABASE opname OWNER opname;
GRANT ALL PRIVILEGES ON DATABASE opname TO opname;
\q
```

### 3. Environment

```powershell
Copy-Item .env.example .env
```

Contoh:

```env
APP_PORT=8080
DATABASE_URL=postgres://opname:opname@localhost:5432/opname?sslmode=disable
JWT_SECRET=change-this-development-secret-at-least-32-characters
```

`JWT_SECRET` wajib ada dan minimal 32 karakter. Gunakan secret berbeda dan kuat untuk production.

### 4. Dependency

```powershell
go mod tidy
```

### 5. Migration

```powershell
go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations postgres "postgres://opname:opname@localhost:5432/opname?sslmode=disable" up
```

### 6. Generate sqlc

```powershell
go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
```

### 7. Jalankan API

```powershell
go run ./cmd/api
```

### 8. Health check

```powershell
Invoke-RestMethod http://localhost:8080/health
```

```json
{
  "status": "ok"
}
```

## Authentication

Public endpoint:

```text
POST /api/v1/auth/login
```

Body:

```json
{
  "email": "user@example.com",
  "password": "your-password"
}
```

Protected endpoint menggunakan:

```text
Authorization: Bearer <JWT>
```

Untuk mengecek actor token:

```text
GET /api/v1/auth/me
```

JWT berlaku 8 jam. Password user disimpan sebagai bcrypt hash di `users.password_hash`.

### Initial user

Repository sengaja **tidak men-seed default username/password**. User awal harus diprovision secara aman ke tabel `users` dengan bcrypt password hash sampai workflow provisioning/admin khusus tersedia. Ini sedikit lebih merepotkan daripada `admin/admin`, tetapi jauh lebih sedikit merepotkan daripada menjelaskan insiden keamanan.

## Role utama

- `AHLI_GIZI`: period, menu template, scheduled menu.
- `PENGAWAS_BAHAN_BAKU`: operasi procurement/gudang, receiving, direct purchase, usage entry, stock opname.
- `AKUNTAN`: verifikasi procurement dan approval terkait.
- `CHEF`: approval material usage dan stock adjustment.
- `KEPALA_SPPG`: permission operasional masih TBD.

Detail kontrak ada di `docs/api.md` dan keputusan desain ada di `docs/decisions.md`.

## Hot reload

```powershell
go run github.com/air-verse/air@latest
```

## Docker opsional

```powershell
Copy-Item .env.example .env
go mod tidy
docker compose up -d postgres
go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations postgres "$env:DATABASE_URL" up
go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
go run github.com/air-verse/air@latest
```

## Database commands

Migration up:

```powershell
go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations postgres "$env:DATABASE_URL" up
```

Rollback satu migration:

```powershell
go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations postgres "$env:DATABASE_URL" down
```

Generate ulang sqlc:

```powershell
go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
```

## CI

GitHub Actions menggunakan PostgreSQL 17 dan menjalankan:

```text
migration up
→ rollback to 0
→ migration up
→ sqlc generate
→ go test ./...
→ go build ./...
```

Integration tests mencakup concurrency inventory serta authentication/RBAC.

## Keamanan

- Jangan commit `.env`.
- Jangan gunakan credential database development di production.
- Jangan gunakan `JWT_SECRET` contoh untuk production.
- Jangan menyimpan password plaintext; gunakan bcrypt.
- Audit actor pada HTTP write-flow berasal dari JWT, bukan UUID yang dipercaya dari body request.
