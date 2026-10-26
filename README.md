<div align="center">
  <img src="frontend/static/logo.webp" alt="Logo Hygeia" width="180" />
  <h1>Hygeia</h1>
  <p><strong>Connected care. Patient-controlled access.</strong></p>
  <p>Pertukaran data kesehatan dengan persetujuan pasien<br />dan jejak akses yang dapat diverifikasi.</p>
  <p>
    <a href="https://github.com/mrizalbasri/Hygeia/actions/workflows/verify.yml"><img src="https://github.com/mrizalbasri/Hygeia/actions/workflows/verify.yml/badge.svg" alt="Status workflow Verify" /></a>
    <img src="https://img.shields.io/badge/status-prototype-475569" alt="Status: prototype" />
  </p>
  <p>
    <a href="#mulai-development">Mulai development</a> ·
    <a href="#fitur">Fitur</a> ·
    <a href="#bahasa-pemrograman">Bahasa pemrograman</a> ·
    <a href="docs/README.md">Dokumentasi</a> ·
    <a href="CONTRIBUTING.md">Kontribusi</a>
  </p>
</div>

---

Hygeia menghubungkan pasien, dokter, laboratorium, dan apotek dalam satu alur
perawatan. Pasien dapat menyetujui pemeriksaan, memilih apotek, mengatur izin baca
hasil, dan meninjau riwayat akses terhadap datanya. Penyedia layanan bekerja
melalui dashboard sesuai peran dan penugasan yang disetujui.

**Status: prototipe untuk development dan demonstrasi dengan data sintetis.**
Bukti dan batas kesiapan dicatat dalam [verifikasi backend](docs/14-BACKEND-READINESS.md).

## Bahasa pemrograman

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![TypeScript](https://img.shields.io/badge/TypeScript-6-3178C6?style=for-the-badge&logo=typescript&logoColor=white)
![JavaScript](https://img.shields.io/badge/JavaScript-F7DF1E?style=for-the-badge&logo=javascript&logoColor=black)
![HTML5](https://img.shields.io/badge/HTML5-E34F26?style=for-the-badge&logo=html5&logoColor=white)
![CSS](https://img.shields.io/badge/CSS-663399?style=for-the-badge&logo=css&logoColor=white)
![SQL](https://img.shields.io/badge/SQL-4169E1?style=for-the-badge&logo=postgresql&logoColor=white)

| Bahasa         | Penggunaan dalam proyek                                            |
| -------------- | ------------------------------------------------------------------ |
| **Go**         | REST API, aturan akses, workflow klinis, audit, migrasi, dan seed. |
| **TypeScript** | Logika frontend, integrasi API, dan komponen Svelte.               |
| **JavaScript** | Skrip pengujian browser dan konfigurasi tooling.                   |
| **HTML & CSS** | Markup serta tampilan komponen dan halaman Svelte.                 |
| **SQL**        | Skema database, migrasi, query, dan aturan integritas data.        |

## Fitur

| Peran        | Alur yang tersedia                                                                                                                                 |
| ------------ | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Pasien       | Registrasi, undangan perawatan, persetujuan lab, hasil dan koreksi, resep, pilihan apotek, izin dokter, CarePass, timeline, dan riwayat akses.     |
| Dokter       | Undangan perawatan, permintaan pemeriksaan, pembacaan hasil berotorisasi, resep multiobat, permintaan izin, penebusan CarePass, dan akses darurat. |
| Laboratorium | Antrean pemeriksaan yang disetujui, pembaruan status, hasil terstruktur, dan koreksi append-only.                                                  |
| Apotek       | Antrean resep yang disetujui, persiapan obat, status siap diambil, dan konfirmasi penyerahan.                                                      |
| Admin        | Pengelolaan akun dan organisasi, verifikasi audit, tinjauan akses darurat, dan log aktivitas.                                                      |

Fitur lintas alur mencakup inbox dan notifikasi melalui Server-Sent Events (SSE),
pagination, serta cetak detail hasil dan resep ke PDF melalui dialog cetak browser.

## Teknologi

![SvelteKit](https://img.shields.io/badge/SvelteKit-3-FF3E00?style=for-the-badge&logo=svelte&logoColor=white)
![Tailwind CSS](https://img.shields.io/badge/Tailwind_CSS-4-06B6D4?style=for-the-badge&logo=tailwindcss&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-4169E1?style=for-the-badge&logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white)
![Vite](https://img.shields.io/badge/Vite-8-646CFF?style=for-the-badge&logo=vite&logoColor=white)

| Lapisan               | Bahasa dan teknologi                         | Kegunaan                                                                |
| --------------------- | -------------------------------------------- | ----------------------------------------------------------------------- |
| Frontend              | TypeScript 6, Svelte 5, SvelteKit 3          | Landing page, SSR, autentikasi, dan dashboard per peran.                |
| Tampilan              | HTML, CSS, Tailwind CSS 4                    | Layout responsif dan tema antarmuka.                                    |
| Backend               | Go 1.26, `net/http`                          | REST API, otorisasi, workflow klinis, dan SSE.                          |
| Database              | SQL, PostgreSQL, `pgx/v5`, `golang-migrate`  | Penyimpanan relasional, transaksi, connection pool, dan migrasi.        |
| Autentikasi dan audit | Argon2id, cookie sesi, CSRF, HMAC-SHA256     | Hash password, proteksi mutasi, dan verifikasi rantai audit per pasien. |
| Pengujian             | Go testing, Vitest, Playwright               | Pengujian backend, komponen, dan alur browser.                          |
| Tooling               | Vite 8, pnpm, Docker Compose, GitHub Actions | Development, build, container backend, dan CI.                          |

Versi bersumber dari [go.mod](backend/go.mod), [package.json](frontend/package.json),
dan lockfile frontend. CI memakai PostgreSQL 18, Node.js 24, dan pnpm 11.18.0.

## Cara kerja

```text
Browser
  │
  ▼
SvelteKit · landing page dan dashboard
  │ /api melalui proxy satu origin
  ▼
Go REST API · sesi, CSRF, policy akses, workflow, audit, SSE
  │ pgx · transaksi SQL
  ▼
PostgreSQL · data klinis, persetujuan, sesi, dan rantai audit
```

Dokter mengundang pasien ke sebuah episode perawatan. Setelah pasien menerima,
dokter dapat meminta pemeriksaan dan membuat resep. Pasien menyetujui pengiriman
ke lab atau apotek yang dipilih; petugas kemudian menyelesaikan penugasannya.
Pembacaan data mengikuti kebijakan akses backend.

Audit memakai rantai HMAC-SHA256 per pasien dan ditulis secara transaksional.
Batas verifikasinya dijelaskan pada [kebijakan care episode dan audit](docs/06-BACKEND-CARE.md).

## Mulai development

### Prasyarat

- Git dan Go **1.26+**.
- Node.js **24** dan pnpm **11.18.0**, mengikuti CI.
- PostgreSQL yang dapat diakses; CI menggunakan **PostgreSQL 18**.
- Docker jika menjalankan backend melalui Compose.

Contoh perintah memakai PowerShell. Compose backend tidak membuat database:
siapkan database dan role terlebih dahulu mengikuti [panduan backend](backend/README.md#database-lokal).

### 1. Clone dan konfigurasi

```powershell
git clone https://github.com/mrizalbasri/Hygeia.git
cd Hygeia
Copy-Item backend/.env.example backend/.env
Copy-Item frontend/.env.example frontend/.env
```

Salin contoh hanya pada checkout baru yang belum mempunyai `.env`. Isi
`backend/.env` dengan konfigurasi lokal:

| Variabel         | Isi yang diperlukan                                          |
| ---------------- | ------------------------------------------------------------ |
| `DATABASE_URL`   | URL koneksi ke database Hygeia.                              |
| `APP_ENV`        | `development` untuk menjalankan seed demo.                   |
| `SESSION_SECRET` | 32 byte acak, ditulis sebagai 64 karakter hex.               |
| `AUDIT_HMAC_KEY` | Kunci acak dengan format yang sama, berbeda dari kunci sesi. |
| `DEMO_PASSWORD`  | Password akun sintetis, minimal 12 byte.                     |
| `PUBLIC_APP_URL` | Origin frontend; default `http://localhost:5173`.            |

Buat kedua kunci dengan **dua pemanggilan terpisah** dan simpan hasilnya hanya di `.env`:

```powershell
[Convert]::ToHexString([Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
```

Perintah pembangkitan kunci memerlukan PowerShell 7+. Konfigurasi Docker dan
rotasi kunci dijelaskan dalam [README backend](backend/README.md).

### 2. Jalankan backend

```powershell
cd backend
go run ./cmd/migrate up
go run ./cmd/seed
go run ./cmd/api
```

API tersedia di `http://127.0.0.1:8080`. Dari terminal lain:

```powershell
Invoke-RestMethod http://127.0.0.1:8080/api/v1/health
Invoke-RestMethod http://127.0.0.1:8080/api/v1/readyz
```

`health` memeriksa proses HTTP; `readyz` memeriksa koneksi database. Kesiapan
skema diperiksa terpisah dengan `go run ./cmd/migrate version` dari `backend`.

### 3. Jalankan frontend

Dari terminal baru di root repository:

```powershell
cd frontend
pnpm install --frozen-lockfile
pnpm dev
```

Buka URL yang ditampilkan Vite, biasanya `http://localhost:5173/`. Halaman masuk
berada di `/login`, registrasi pasien di `/register`. Browser memakai proxy
`/api`; SSR memakai `API_BASE_URL` dari `frontend/.env`.

### Akun demo

| Peran        | Email                     |
| ------------ | ------------------------- |
| Pasien       | `aulia@demo.hygeia.test`  |
| Dokter       | `maya@demo.hygeia.test`   |
| Laboratorium | `lab@demo.hygeia.test`    |
| Apotek       | `apotek@demo.hygeia.test` |
| Admin        | `admin@demo.hygeia.test`  |

Password mengikuti `DEMO_PASSWORD` pada seed pertama. Seed tidak mengganti
password akun yang sudah ada. Akun tambahan ada pada [panduan seed](backend/README.md#migrasi-dan-akun-demo).

## Pengujian

Backend, dari root repository, dengan `TEST_DATABASE_URL` diarahkan ke database development/test:

```powershell
./scripts/Test-Backend.ps1
```

Skrip menjalankan tes Go dengan integrasi database dan `go vet`, serta menolak
berjalan tanpa konfigurasi database tes. Opsi konfigurasi lokal ada di [panduan kontribusi](CONTRIBUTING.md).

Frontend, dari `frontend`:

```powershell
pnpm exec playwright install chromium
pnpm check
pnpm lint
pnpm build
pnpm test
pnpm test:e2e
```

Suite E2E fixture mengelola server tes sendiri. Untuk memakai Edge di Windows,
set `$env:PLAYWRIGHT_CHANNEL='msedge'`. Uji terhadap API demo lokal dijalankan
terpisah; lihat [panduan frontend](frontend/README.md#pengujian).
Pemeriksaan CI ada di [Verify](.github/workflows/verify.yml).

## Struktur repository

```text
Hygeia/
├── backend/
│   ├── cmd/          # Entrypoint API, migrasi, dan seed
│   ├── db/           # Migrasi dan dukungan database
│   └── internal/     # Implementasi backend
├── frontend/
│   ├── src/lib/      # Komponen dan modul bersama
│   ├── src/routes/   # Halaman, layout, dan route SvelteKit
│   ├── static/       # Logo dan aset visual
│   └── tests/        # Pengujian alur browser
├── docs/             # Kebutuhan, desain, kontrak API, dan rencana
├── scripts/          # Skrip verifikasi lokal
└── .github/          # Workflow CI
```

## Dokumentasi lanjutan

- [Indeks dokumentasi](docs/README.md) — panduan membaca berdasarkan kebutuhan.
- [Backend](backend/README.md) — konfigurasi, Docker, API, dan batas implementasi.
- [Frontend](frontend/README.md) — dashboard, proxy, SSR, dan pengujian browser.
- [Arsitektur](docs/03-ARCHITECTURE.md) dan [design system](docs/04-DESIGN.md).
- [Kontrak API](docs/05-BACKEND-API.md) dan [handoff workflow](docs/plans/2026-10-15-patient-doctor-and-provider-handoff.md).
- [Kontribusi](CONTRIBUTING.md) — alur perubahan, standar Git, dan validasi.

## Batas penggunaan

Gunakan data sintetis untuk development dan demonstrasi. Deployment produksi
memerlukan adapter frontend yang sesuai, HTTPS, proxy satu origin, dan konfigurasi
database serta operasional yang ditinjau. Verifikasi email, reset password, MFA,
dan checkpoint audit eksternal belum tersedia. Rantai HMAC tidak melindungi dari
pihak yang menguasai database sekaligus semua kunci audit.

File `.env`, kredensial, cache, dan hasil build tidak disertakan dalam Git.
