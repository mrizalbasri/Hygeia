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

Migrasi 1 membuat `users`, `organizations`, `organization_memberships`, `patients`,
dan `sessions`. Migrasi 2 menambah `care_episodes`, `audit_chain_heads`, dan
`audit_events`. Migrasi 3 menambah `lab_requests`, `lab_results`, dan proteksi
immutability hasil. Migrasi 4 menambah `consents` untuk izin baca satu hasil lab.
Migrasi 5 menambah `prescriptions` dan `prescription_items` untuk resep dan
penugasan apotek yang disetujui pasien. Jalankan migrasi sebelum memakai API baru.

Migrasi 6 menambah ledger admin, 7 inbox notifikasi, 8 grant akses darurat,
dan 9 CarePass berbatas waktu. Backend inti dan fitur tambahan kini tersedia:

| Modul | Kontrak aktual |
| --- | --- |
| Admin akun/organisasi dan ledger | [Admin](../docs/09-BACKEND-ADMIN.md) |
| Inbox dan read state | [Notifikasi](../docs/10-BACKEND-NOTIFICATIONS.md) |
| Akses darurat 60 menit dan tinjauan | [Emergency](../docs/11-backend-emergency-api.md) |
| CarePass/QR dan pencabutan | [CarePass](../docs/12-BACKEND-CAREPASS.md) |
| SSE/replay notifikasi | [Realtime](../docs/13-BACKEND-REALTIME.md) |

CarePass memerlukan `PUBLIC_APP_URL` berupa origin aplikasi, default development
`http://localhost:5173`. Production wajib origin HTTPS. QR mengarah ke route
frontend `/carepass#token=...`; UI scan/redeem harus memakai kontrak backend.

## Migrasi dan akun demo

Dari folder `backend`, jalankan:

```powershell
go run ./cmd/migrate up
go run ./cmd/migrate version
go run ./cmd/seed
```

Migrasi menggunakan `golang-migrate` dengan SQL tertanam di binary. Perintah `up`
aman diulang; `version` menampilkan versi dan status dirty. Jangan memaksa versi
jika migrasi gagal; periksa SQL dan database terlebih dahulu. File `down.sql`
tersedia untuk pengembangan, tetapi CLI ini tidak menyediakan rollback destruktif.
API tidak menjalankan migrasi otomatis.

Seed hanya diizinkan saat `APP_ENV=development` atau `test`, memakai
`DEMO_PASSWORD` dari `.env` (minimal 12 byte). Seed bisa diulang dan tidak
mengubah password atau akun yang sudah ada. Mengganti `DEMO_PASSWORD` setelah
seed tidak otomatis mengganti password akun yang sudah dibuat.

| Peran | Email sintetis |
|---|---|
| Pasien | `aulia@demo.hygeia.test` |
| Dokter episode | `maya@demo.hygeia.test` |
| Dokter lintas organisasi | `budi@demo.hygeia.test` |
| Dokter IGD | `rina@demo.hygeia.test` |
| Laboratorium | `lab@demo.hygeia.test` |
| Apotek | `apotek@demo.hygeia.test` |
| Admin | `admin@demo.hygeia.test` |

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
APP_ENV=development
SESSION_SECRET=CHANGE_ME
AUDIT_HMAC_KEY=CHANGE_ME
DEMO_PASSWORD=CHANGE_ME
```

`SESSION_SECRET` wajib berupa 32 byte acak yang di-encode menjadi 64 karakter hex.
`AUDIT_HMAC_KEY` juga wajib berisi 32 byte acak dalam 64 karakter hex dan harus
berbeda dari `SESSION_SECRET`. Buat masing-masing dengan pemanggilan terpisah
(masukkan hasilnya hanya ke `.env`):

```powershell
[Convert]::ToHexString([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
```

Cadangkan kunci audit secara aman terpisah dari DB. Jangan mengganti kunci untuk
database yang sudah memiliki audit, karena rantai lama akan gagal diverifikasi.
Rotasi kunci belum tersedia; jangan memakai placeholder untuk menjalankan API.

`APP_ENV=production` mengaktifkan flag cookie `Secure`; mode tersebut membutuhkan
HTTPS di reverse proxy. Development lokal menggunakan cookie tanpa `Secure`.
Gunakan satu origin dengan proxy `/api` ke backend; CORS tidak dibuka.

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

## API autentikasi

| Metode | Endpoint | Hasil |
|---|---|---|
| GET | `/api/v1/auth/csrf` | Cookie CSRF + JSON `{ "csrf_token": "..." }` |
| POST | `/api/v1/auth/login` | JSON `{ "email", "password" }`; cookie sesi + `{ "user": {...} }` |
| POST | `/api/v1/auth/logout` | Cabut sesi di DB dan hapus cookie; HTTP 204 |
| GET | `/api/v1/me` | Identitas/peran/organisasi dari DB; HTTP 401 tanpa sesi aktif |

Semua POST/PATCH wajib membawa cookie CSRF dan header `X-CSRF-Token` dari endpoint
CSRF. Login memerlukan `Content-Type: application/json`. Pengembang frontend
perlu menambahkan proxy Vite dan integrasi login; klien API dapat menguji
endpoint langsung dengan cookie jar. Respons autentikasi memakai `Cache-Control: no-store`.
Error berbentuk `{ "error": { "code", "message", "request_id" } }`.

Password memakai Argon2id (64 MiB, 3 iterasi, paralelisme 2, salt acak 16 byte).
Token sesi 32 byte acak disimpan sebagai SHA-256; token mentah hanya ada di cookie
`HttpOnly`, `SameSite=Lax`, path `/`. Masa berlaku absolut 8 jam. Logout mencabut
sesi, dan akun nonaktif langsung ditolak pada pembacaan identitas berikutnya.

Login dibatasi per alamat koneksi TCP: burst 5, pulih satu percobaan setiap 12
detik. Limiter berada di memori dan tidak mempercayai `X-Forwarded-For`. Di balik
proxy, klien bisa berbagi bucket; kebijakan trusted proxy/limiter terdistribusi
perlu ditentukan sebelum deployment. Pembersihan sesi kedaluwarsa belum
dijadwalkan; otorisasi tetap memeriksa waktu kedaluwarsa pada setiap request.

Tahap ini belum mencakup registrasi publik, verifikasi email, reset password,
MFA, policy resep, atau audit login berantai hash. Frontend yang di-commit masih
halaman starter; login dan dashboard per peran belum tersedia.

## API care episode dan audit

| Metode | Endpoint | Fungsi |
|---|---|---|
| POST | `/api/v1/episodes` | Dokter mengundang dengan `{"patient_ref":"HYG-DEMO-001"}` |
| GET | `/api/v1/episodes` | Daftar milik aktor; filter status dan pagination |
| GET | `/api/v1/episodes/{id}` | Detail metadata episode, hanya kedua peserta |
| PATCH | `/api/v1/episodes/{id}` | `{"action":"accept"}`, `reject`, atau `end` |
| GET | `/api/v1/patient/access-summary` | Riwayat akses episode dan laboratorium pasien dalam kalimat |
| GET | `/api/v1/audit/verify` | Verifikasi rantai sendiri; admin wajib `?chain=<patient_id>` |

POST/PATCH memakai proteksi CSRF yang sama dengan login. Pasien saja yang dapat
accept/reject; pasien atau dokter pengundang dapat end setelah episode aktif.
Dokter lain, termasuk rekan satu klinik, tidak otomatis mendapat akses.
Undangan `pending`/`active` ganda untuk pasangan dokter–pasien ditolak `409`.
Status `rejected` dan `ended` terminal; undangan baru boleh dibuat setelahnya.
Detail riwayat episode tetap terbaca oleh peserta; akses hasil lab jalur A hanya
berlaku untuk episode aktif. Mengakhiri episode membatalkan permintaan lab yang
belum disetujui, dalam transaksi dan audit yang sama. Permintaan lab yang sudah
disetujui tetap dapat diselesaikan; pasien tetap berhak membaca hasilnya.

Audit HMAC-SHA256 ditulis transaksional, berantai per pasien dengan row lock.
Penolakan akses ke episode yang diketahui tetap dicatat tanpa mengubah episode.
Jika audit gagal, perubahan dibatalkan dan data tidak dikembalikan. Trigger
memblokir UPDATE/DELETE/TRUNCATE biasa pada audit events; pemilik DB tetap dapat
menonaktifkannya. Verifikasi mendeteksi manipulasi payload, nomor urut, tautan,
dan penghapusan ekor melalui perbandingan kepala rantai. Pemegang DB dan kunci
HMAC bisa menulis ulang seluruh rantai; checkpoint eksternal belum tersedia.

Kontrak request/response dan alur integrasi untuk frontend:
[`docs/05-BACKEND-API.md`](../docs/05-BACKEND-API.md).
Batas lingkup, kebijakan akses, dan format audit:
[`docs/06-BACKEND-CARE.md`](../docs/06-BACKEND-CARE.md).

## API laboratorium

| Metode | Endpoint | Fungsi |
|---|---|---|
| GET | `/api/v1/laboratories` | Direktori ID/nama laboratorium, hanya dokter |
| POST | `/api/v1/lab-requests` | Buat tes dalam episode aktif milik dokter |
| GET | `/api/v1/lab-requests` | Daftar sesuai aktor; filter status dan pagination |
| GET | `/api/v1/lab-requests/{id}` | Detail permintaan tanpa nilai hasil |
| POST | `/api/v1/lab-requests/{id}/approval` | Pasien approve/reject pengiriman |
| PATCH | `/api/v1/lab-requests/{id}/status` | Lab: accepted, lalu in_progress |
| POST | `/api/v1/lab-requests/{id}/result` | Lab mengirim satu hasil; status completed |
| GET | `/api/v1/lab-results/{id}` | Hasil untuk pasien sendiri, dokter episode aktif, atau consent aktif |

Alur: `pending_patient_approval → sent → accepted → in_progress → completed`.
Penolakan pasien menghasilkan `cancelled`. Lab tujuan melihat antrean setelah
persetujuan saja; responsnya hanya ID permintaan, kode pasien, kode/nama tes,
dan status. Hasil dikirim satu kali saat `in_progress`; hasil tidak dapat diedit,
dihapus, atau dibaca kembali oleh lab melalui API. Setelah selesai, permintaan
keluar dari antrean lab. Hasil tetap bisa dibaca pasien; dokter kehilangan akses
saat episode berakhir atau keanggotaan kliniknya berubah, kecuali mempunyai
consent terpisah yang masih berlaku pada kliniknya saat ini.

Semua mutasi mengunci episode sebelum permintaan. Persetujuan yang bersamaan
dengan pengakhiran episode mengikuti urutan lock: jika persetujuan menang,
penugasan lab tetap berlaku; jika pengakhiran menang, permintaan dibatalkan.
Nilai hasil tidak disalin ke audit atau log. Dukungan PDF, koreksi hasil,
penarikan persetujuan yang sudah diberikan, dan notifikasi belum tersedia.
Kontrak payload dan contoh lengkap ada di panduan API frontend.

## API consent

Dokter meminta izin baca satu hasil melalui `POST /api/v1/consents` dengan
`patient_ref`, `resource_type: "lab_result"`, `resource_id`, dan `purpose`.
`GET /api/v1/consents` dan `GET /api/v1/consents/{id}` hanya tersedia untuk
pasien pemilik atau dokter peminta pada klinik yang sesuai.

Pasien `PATCH /api/v1/consents/{id}` dengan `action: approve` serta
`duration_hours: 1`, `24`, atau `168`; atau memilih `deny` / `revoke` tanpa durasi.
Dokter membaca hasil melalui endpoint hasil yang sudah ada. Policy mengecek
scope, dokter, klinik, status, dan kedaluwarsa setiap pembacaan. Izin tidak
memberi akses ke episode, permintaan lab, atau hasil lain. Pencabutan langsung
memblokir pembacaan berikutnya; akses episode aktif tetap mempunyai basis sendiri.

Audit dan aksi ditulis dalam transaksi yang sama, termasuk pembacaan berbasis
consent. Context memuat ID izin dan waktu habis, tanpa nilai hasil. Tes khusus:
`go test ./internal/care -run Consent -v` dengan `TEST_DATABASE_URL` lokal.
Contoh payload dan panduan frontend ada di
[`docs/05-BACKEND-API.md`](../docs/05-BACKEND-API.md#consent-untuk-satu-hasil-laboratorium).

## API resep dan apotek

Dokter menerbitkan resep dalam episode aktif melalui `POST /api/v1/prescriptions`.
Pasien mengambil direktori `GET /api/v1/pharmacies` dan memilih apotek sekaligus
menyetujui pengiriman lewat `POST /api/v1/prescriptions/{id}/approval`, atau
menolak resep. Staf apotek tujuan menggunakan
`PATCH /api/v1/prescriptions/{id}/status` untuk tahapan
`sent → accepted → preparing → ready → completed`.

Daftar/detail resep mengikuti izin pasien, dokter episode aktif, atau apotek
yang ditugaskan selama penugasan aktif. Apotek hanya melihat ID, referensi
pasien, item obat, dan status. Pending dibatalkan saat episode berakhir;
penugasan yang sudah disetujui tetap dapat diselesaikan. Aksi dan audit berbagi
transaksi. Kontrak lengkap, validasi, dan contoh payload:
[`docs/07-BACKEND-PRESCRIPTIONS.md`](../docs/07-BACKEND-PRESCRIPTIONS.md).

## Timeline perjalanan pasien

`GET /api/v1/patient/timeline` menyajikan kejadian bisnis sukses pada rantai
pasien: episode, laboratorium, resep, dan consent. Akses hanya untuk pasien
pemilik berdasarkan sesi. Filter `kind`, `limit` 1–100, dan cursor eksklusif
`before_seq` menggantikan offset untuk endpoint ini.

Timeline tidak memuat pembacaan data atau akses ditolak; aktivitas itu tetap
tersedia pada riwayat akses. Isi hasil lab dan obat tidak disertakan. Proyeksi
menggunakan audit yang sudah ada tanpa tabel/migrasi baru dan tanpa perubahan
format hash. Kontrak lengkap:
[`docs/08-BACKEND-TIMELINE.md`](../docs/08-BACKEND-TIMELINE.md).
Tes khusus: `go test ./internal/care -run Timeline -v` dengan `TEST_DATABASE_URL`.

## Validasi dan build

```powershell
go test ./cmd/... ./db/... ./internal/...
go vet ./cmd/... ./db/... ./internal/...
go build -o bin/hygeia-api.exe ./cmd/api
```

Untuk tes integrasi autentikasi, isi `TEST_DATABASE_URL` dengan URL database
development lalu jalankan `go test ./internal/auth -run Integration -v`.
Tes membuat schema unik di dalam transaksi dan selalu rollback; membutuhkan
hak membuat schema. Tanpa variabel tersebut, tes integrasi dilewati. Jangan
gunakan database produksi untuk tes.

Tes care/laboratorium/audit: `go test ./internal/care -run Integration -v`. Tes membuat schema
acak `care_test_<hex>` yang di-commit untuk menguji konkurensi transaksi dan
dihapus melalui cleanup. Hanya schema tes tersebut yang diubah, termasuk tes
manipulasi audit. Memerlukan hak CREATE/DROP schema. Tanpa `TEST_DATABASE_URL`,
tes integrasi dilewati. Gunakan `go test ./cmd/... ./db/... ./internal/...` dengan variabel tersebut untuk
menjalankan seluruh tes unit dan integrasi.

Tes resep memakai fixture care dengan migrasi 1–5. Jalankan
`go test ./internal/care -run Prescription -v` untuk alur lengkap, proyeksi
apotek, pembatasan akses, race, dan rollback audit. Tes unit policy/validasi ada
di `internal/policy` dan `internal/prescription`. Daftar paket eksplisit di atas
menghindari source sementara di `bin` yang mungkin tersisa dari pekerjaan lokal.

Tes laboratorium memakai fixture care yang sama agar race dengan pengakhiran
episode diuji lintas modul. Jalankan khusus dengan
`go test ./internal/care -run Laboratory -v`. Tes unit policy ada di
`internal/policy`; tes validasi request ada di `internal/laboratory`.

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
- `db/migrations/`: migrasi skema identitas, episode, audit, laboratorium, dan consent.
- `cmd/migrate/`, `cmd/seed/`: penerapan migrasi dan data sintetis.
- `internal/auth/`: Argon2id, sesi, CSRF, limiter login, dan API identitas.
- `internal/care/`: transaksi episode, API, dan tes integrasi/concurrency.
- `internal/policy/`: keputusan akses episode, permintaan lab, dan hasil lab.
- `internal/laboratory/`: penugasan lab, hasil terstruktur, dan pembatalan saat episode berakhir.
- `internal/consent/`: izin baca satu hasil lab, keputusan pasien, durasi, dan pencabutan.
- `internal/audit/`: append HMAC transaksional, verifikasi, ringkasan kalimat.
- `internal/httpapi/`: routing, health, readiness, dan logging request JSON.

Frontend SvelteKit berada di `../frontend` dan dapat menjalankan halaman starter
pada port default `5173`. Checkout yang di-commit belum memuat proxy `/api`,
halaman `/login`, dashboard, atau helper SSR. Pengembang frontend perlu
menambahkan integrasi tersebut sesuai
[panduan API](../docs/05-BACKEND-API.md#integrasi-frontend-yang-perlu-ditambahkan).

Referensi: [pgx](https://github.com/jackc/pgx),
[PostgreSQL Docker image](https://hub.docker.com/_/postgres).
