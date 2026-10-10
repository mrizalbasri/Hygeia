# Hygeia Backend

REST API Go (`net/http`) dengan PostgreSQL melalui connection pool `pgx`.
Memerlukan Go 1.26 atau lebih baru.

## Database lokal

Hygeia memakai instance PostgreSQL Docker yang sudah ada, tanpa container baru:

| Konfigurasi | Nilai |
|---|---|
| Container | `postgres-local` |
| Host | `127.0.0.1` |
| Port host | `5434` (port internal container `5432`) |
| Database | `hygeia` |
| User | `hygeia_app` |
| Password | Tersimpan di `backend/.env` lokal |

Database dan role Hygeia terpisah dari database `tera`. Role `hygeia_app` memiliki
database Hygeia, tanpa hak superuser, membuat database, atau membuat role.
Instance dan volume tetap dikelola oleh project Tera; instance itu harus hidup
selama Hygeia dijalankan. Hygeia tidak mengelola lifecycle container tersebut.
Container telah diganti nama dari `tera-postgres-1` ke `postgres-local` melalui
`docker rename`. Grup Compose masih bernama `tera`; jika container dibuat ulang
dari `D:/Coding/Tera/compose.yaml`, nama dapat kembali mengikuti konfigurasi Compose.

Saat pindah komputer atau clone baru, siapkan database dan role pada instance
PostgreSQL yang tersedia, lalu salin `.env.example` ke `.env` dan isi koneksi.
Password lokal tidak disertakan dalam repository. Jangan mengganti `.env` lokal
yang sudah dibuat dengan nilai placeholder dari contoh.

Belum ada tabel bisnis atau migrasi domain; tahap ini menyiapkan database kosong
dan koneksi backend. Skema pengguna, organisasi, sesi, consent, dan audit akan
ditambahkan melalui migrasi pada tahap berikutnya.

## Menjalankan

Dari root project di PowerShell:

```powershell
cd backend
go run ./cmd/api
```

Server membaca `.env` dari direktori kerja. Environment shell yang sudah ada
memiliki prioritas di atas `.env`. `DATABASE_URL` wajib diisi; startup gagal jika
database tidak dapat dihubungi. Port HTTP default `8080`, dapat diubah lewat `PORT`.

Contoh koneksi (ganti password):

```dotenv
PORT=8080
DATABASE_URL=postgres://hygeia_app:CHANGE_ME@127.0.0.1:5434/hygeia?sslmode=disable
```

`sslmode=disable` digunakan untuk development lokal. Deployment harus memakai
konfigurasi TLS sesuai penyedia database. Jangan commit `.env` (sudah diabaikan
oleh `.gitignore`). Tekan Ctrl+C untuk graceful shutdown; koneksi database ditutup
setelah server menyelesaikan request aktif, dengan batas waktu 10 detik.

## Health dan readiness

```powershell
# Proses HTTP aktif
Invoke-RestMethod http://localhost:8080/api/v1/health

# PostgreSQL bisa dihubungi
Invoke-RestMethod http://localhost:8080/api/v1/readyz
```

Health mengembalikan HTTP 200 dengan `{"service":"hygeia-api","status":"ok"}`.
Readiness mengembalikan HTTP 200 dengan `{"database":"connected","status":"ok"}`,
atau HTTP 503 jika koneksi database gagal. Respons tidak mengungkap kredensial.
Readiness memeriksa koneksi, belum memeriksa versi skema atau migrasi.

## Validasi dan build

```powershell
go test ./...
go vet ./...
go build -o bin/hygeia-api.exe ./cmd/api
```

Jika drive C hampir penuh, gunakan cache dan direktori sementara di project
(drive D) sebelum menjalankan perintah Go. Jalankan dari folder `backend`:

```powershell
New-Item -ItemType Directory -Force bin/go-cache, bin/go-tmp, bin/.go-mod | Out-Null
$env:GOCACHE = Join-Path $PWD 'bin/go-cache'
$env:GOTMPDIR = Join-Path $PWD 'bin/go-tmp'
$env:GOMODCACHE = Join-Path $PWD 'bin/.go-mod'
```

Semua direktori tersebut berada di `bin/` yang diabaikan Git. Cache modul memakai
nama tersembunyi `.go-mod` agar tidak ikut ditemukan oleh `go test ./...`.
## Struktur

- `cmd/api/main.go`: konfigurasi, koneksi database, server HTTP, graceful shutdown.
- `internal/database/`: connection pool PostgreSQL (maksimal 10 koneksi).
- `internal/httpapi/`: routing, health, readiness, dan logging request JSON.

Frontend SvelteKit berada di `../frontend` dan dijalankan terpisah. Integrasi
frontend dan autentikasi belum dikonfigurasi.

Referensi: [pgx](https://github.com/jackc/pgx),
[PostgreSQL Docker image](https://hub.docker.com/_/postgres).
