# Go Opname API

Backend API untuk pengadaan bahan baku harian berdasarkan rencana menu mingguan.

## Tech stack

- Go 1.25+
- Fiber
- PostgreSQL
- pgx
- Goose migrations
- sqlc
- Air untuk hot reload

## Status project

Fondasi lokal sudah berhasil dijalankan dan divalidasi di Windows dengan PostgreSQL lokal:

- koneksi PostgreSQL berhasil
- migration Goose berhasil
- `sqlc generate` berhasil
- API Fiber berjalan di port `8080`
- endpoint `GET /health` mengembalikan `{ "status": "ok" }`

## Prasyarat

Pastikan sudah terpasang:

```powershell
git --version
go version
postgres --version
psql --version
```

Docker tidak wajib. Panduan utama di bawah menggunakan PostgreSQL yang terpasang langsung di komputer.

## Setup lokal dengan PostgreSQL langsung

### 1. Clone repository

```powershell
git clone https://github.com/allifiz/go-opname-api.git
cd go-opname-api
```

Jika repository sudah pernah di-clone:

```powershell
git checkout main
git pull origin main
```

### 2. Buat user dan database PostgreSQL

Masuk ke PostgreSQL menggunakan user administrator:

```powershell
psql -U postgres
```

Jalankan:

```sql
CREATE USER opname WITH PASSWORD 'opname';
CREATE DATABASE opname OWNER opname;
GRANT ALL PRIVILEGES ON DATABASE opname TO opname;
```

Keluar dari `psql`:

```sql
\q
```

Jika user atau database tersebut sudah tersedia, langkah pembuatan dapat dilewati.

### 3. Buat file environment

Windows PowerShell:

```powershell
Copy-Item .env.example .env
```

Windows Command Prompt:

```cmd
copy .env.example .env
```

Isi default `.env`:

```env
APP_PORT=8080
DATABASE_URL=postgres://opname:opname@localhost:5432/opname?sslmode=disable
```

Sesuaikan username, password, host, port, atau database jika konfigurasi PostgreSQL lokal berbeda.

### 4. Download dependency Go

```powershell
go mod tidy
```

### 5. Jalankan migration

```powershell
go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations postgres "postgres://opname:opname@localhost:5432/opname?sslmode=disable" up
```

### 6. Generate kode sqlc

```powershell
go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
```

Pada eksekusi pertama, proses ini dapat memerlukan waktu karena Go mengunduh dependency `sqlc`.

### 7. Jalankan API

```powershell
go run ./cmd/api
```

Server yang berhasil berjalan akan menampilkan informasi bahwa API listening pada port `8080`.

### 8. Tes health endpoint

Gunakan browser, Postman, atau PowerShell:

```powershell
Invoke-RestMethod http://localhost:8080/health
```

Response:

```json
{
  "status": "ok"
}
```

Di Postman:

```text
Method : GET
URL    : http://localhost:8080/health
```

## Hot reload dengan Air

```powershell
go run github.com/air-verse/air@latest
```

API akan restart otomatis ketika file Go atau SQL berubah.

## Setup opsional dengan Docker

Jika Docker Desktop tersedia:

```powershell
Copy-Item .env.example .env
go mod tidy
docker compose up -d postgres
go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations postgres "postgres://opname:opname@localhost:5432/opname?sslmode=disable" up
go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
go run github.com/air-verse/air@latest
```

Jika `make` tersedia:

```powershell
make tidy
make dev
```

Perintah `make dev` menggunakan PostgreSQL dari Docker Compose.

## DBeaver

Gunakan konfigurasi berikut untuk koneksi PostgreSQL lokal:

```text
Host     : localhost
Port     : 5432
Database : opname
Username : opname
Password : opname
```

Setelah migration berjalan, refresh bagian:

```text
Schemas > public > Tables
```

Tabel awal yang tersedia antara lain:

- `users`
- `ingredients`
- `goose_db_version`

## Perintah database

Migration naik:

```powershell
go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations postgres "postgres://opname:opname@localhost:5432/opname?sslmode=disable" up
```

Rollback satu migration:

```powershell
go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations postgres "postgres://opname:opname@localhost:5432/opname?sslmode=disable" down
```

Generate ulang sqlc:

```powershell
go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
```

## Catatan keamanan

Konfigurasi `opname/opname` hanya untuk development lokal. Jangan gunakan password tersebut untuk production dan jangan commit file `.env` ke repository.
