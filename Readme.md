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

Concurrency-sensitive flow divalidasi dengan PostgreSQL asli di GitHub Actions. Authentication/JWT + RBAC sudah diterapkan pada HTTP API, termasuk bootstrap aman untuk membuat user pertama tanpa default credential.

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
BOOTSTRAP_TOKEN=
```

`JWT_SECRET` wajib ada dan minimal 32 karakter. `BOOTSTRAP_TOKEN` opsional; jika diisi, nilainya juga wajib minimal 32 karakter dan hanya dipakai untuk membuat user pertama. Gunakan secret acak yang berbeda untuk production.

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

### 7. Buat user pertama

Sebelum API dijalankan pertama kali, isi `BOOTSTRAP_TOKEN` di `.env` dengan secret acak minimal 32 karakter. Contoh PowerShell untuk membuat token acak:

```powershell
$bytes = New-Object byte[] 32
[System.Security.Cryptography.RandomNumberGenerator]::Fill($bytes)
[Convert]::ToHexString($bytes).ToLower()
```

Simpan hasilnya ke `.env`, lalu jalankan API:

```powershell
$bootstrapToken = [Convert]::ToHexString($bytes).ToLower()
(Get-Content .env) -replace '^BOOTSTRAP_TOKEN=.*$', "BOOTSTRAP_TOKEN=$bootstrapToken" | Set-Content .env
go run ./cmd/api
```

Di terminal PowerShell lain, gunakan nilai token yang sama untuk provision user pertama:

```powershell
$bootstrapToken = "<nilai BOOTSTRAP_TOKEN dari .env>"
$headers = @{ "X-Bootstrap-Token" = $bootstrapToken }
$body = @{
  name = "Initial Accountant"
  email = "akuntan@example.com"
  password = "ganti-dengan-password-kuat"
  role = "AKUNTAN"
} | ConvertTo-Json

Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/api/v1/auth/bootstrap `
  -Headers $headers `
  -ContentType "application/json" `
  -Body $body
```

Role bootstrap yang valid: `CHEF`, `AHLI_GIZI`, `PENGAWAS_BAHAN_BAKU`, `AKUNTAN`, `KEPALA_SPPG`. Password harus 12-72 karakter.

Setelah user pertama berhasil dibuat, kosongkan/hapus `BOOTSTRAP_TOKEN` dari environment lalu restart API. Database juga menolak bootstrap berikutnya selama sudah ada user, jadi pengaman tidak hanya bergantung pada konfigurasi.

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

Public endpoint login:

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

Repository sengaja **tidak men-seed default username/password**. User pertama dibuat melalui one-time bootstrap secret seperti langkah setup di atas. Dua request bootstrap yang datang bersamaan diserialisasi oleh database, sehingga hanya satu yang dapat membuat user pertama.

Workflow untuk membuat user tambahan setelah bootstrap belum ditetapkan karena permission administratifnya belum disepakati in business flow. Tidak ada role `ADMIN` baru yang diselundupkan diam-diam hanya demi membuat endpoint terasa lengkap.

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

Integration tests mencakup concurrency inventory, authentication/RBAC, dan concurrent initial-user bootstrap.

## Keamanan

- Jangan commit `.env`.
- Jangan gunakan credential database development di production.
- Jangan gunakan `JWT_SECRET` contoh untuk production.
- Jangan commit atau membagikan `BOOTSTRAP_TOKEN`; unset setelah provisioning user pertama.
- Jangan menyimpan password plaintext; gunakan bcrypt.
- Audit actor pada HTTP write-flow berasal dari JWT, bukan UUID yang dipercaya dari body request.
