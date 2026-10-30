<div align="center">
  <img src="frontend/static/logo.webp" alt="Logo Hygeia" width="160" />
  <h1>Hygeia</h1>
  <p><strong>Perawatan terhubung, akses data dalam kendali pasien.</strong></p>
  <p>Platform pertukaran data kesehatan antara pasien, dokter, laboratorium, dan apotek<br />dengan persetujuan pasien serta jejak akses yang dapat diverifikasi.</p>
  <p>
    <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&amp;logo=go&amp;logoColor=white" alt="Go" />
    <img src="https://img.shields.io/badge/TypeScript-3178C6?style=for-the-badge&amp;logo=typescript&amp;logoColor=white" alt="TypeScript" />
    <img src="https://img.shields.io/badge/JavaScript-F7DF1E?style=for-the-badge&amp;logo=javascript&amp;logoColor=black" alt="JavaScript" />
    <img src="https://img.shields.io/badge/HTML5-E34F26?style=for-the-badge&amp;logo=html5&amp;logoColor=white" alt="HTML5" />
    <img src="https://img.shields.io/badge/CSS-663399?style=for-the-badge&amp;logo=css&amp;logoColor=white" alt="CSS" />
    <img src="https://img.shields.io/badge/SQL-4169E1?style=for-the-badge&amp;logo=postgresql&amp;logoColor=white" alt="SQL (PostgreSQL)" />
  </p>
  <p>
    <a href="https://github.com/mrizalbasri/Hygeia/actions/workflows/verify.yml"><img src="https://github.com/mrizalbasri/Hygeia/actions/workflows/verify.yml/badge.svg" alt="Status workflow Verify" /></a>
    <img src="https://img.shields.io/badge/status-prototipe-475569" alt="Status: prototipe" />
  </p>
  <p>
    <a href="#tentang-hygeia">Tentang</a> ·
    <a href="#fitur-utama">Fitur</a> ·
    <a href="#panduan-demonstrasi">Panduan demonstrasi</a> ·
    <a href="#teknologi">Teknologi</a> ·
    <a href="#instalasi-lokal">Instalasi</a>
  </p>
</div>

---

## Tentang Hygeia

Hygeia adalah prototipe aplikasi web yang menghubungkan permintaan pemeriksaan,
hasil laboratorium, dan resep dalam satu perjalanan perawatan. Pasien dapat
menyetujui pengiriman layanan, memilih apotek, membagikan data tertentu kepada
dokter, serta meninjau riwayat akses terhadap datanya.

Dokter, laboratorium, dan apotek memiliki ruang kerja sesuai peran dan
organisasinya. Backend memeriksa hak akses berdasarkan identitas pengguna,
hubungan perawatan, penugasan layanan, dan izin yang berlaku.

**Demonstrasi Hygeia menggunakan data sintetis.**

## Fitur utama

| Fitur | Fungsi |
| --- | --- |
| Hubungan perawatan | Dokter mengundang pasien melalui kode pasien; hubungan perawatan aktif setelah pasien menerima undangan. |
| Pemeriksaan laboratorium | Dokter membuat permintaan, pasien menyetujui pengiriman, dan laboratorium tujuan memproses serta menyerahkan hasil terstruktur. |
| Resep dan apotek | Dokter menerbitkan resep; pasien memilih apotek dan menyetujui pengiriman. Apotek memperbarui status hingga obat diserahkan. |
| Izin baca dan CarePass | Pasien mengelola izin baca serta membagikan satu hasil lab yang selesai atau satu resep melalui CarePass selama 15 atau 60 menit. |
| Riwayat dan notifikasi | Pasien meninjau perjalanan perawatan, perubahan status, notifikasi, serta riwayat akses. |
| Audit dan administrasi | Administrator mengelola akun dan organisasi, memverifikasi integritas audit, serta meninjau akses darurat. |

Hasil laboratorium dapat dikoreksi dengan mempertahankan riwayat sebelumnya.
Detail hasil dan resep dapat dicetak melalui dialog cetak browser, termasuk
pilihan simpan sebagai PDF.

### Ruang kerja pengguna

| Peran | Kegiatan utama |
| --- | --- |
| Pasien | Menerima undangan, memberikan persetujuan, melihat hasil dan resep, memilih apotek, serta mengelola izin data. |
| Dokter | Mengelola hubungan perawatan, meminta pemeriksaan, membaca data yang diizinkan, menerbitkan resep, dan menebus CarePass. |
| Laboratorium | Menangani antrean yang ditujukan ke organisasinya, memperbarui status, serta menyerahkan dan mengoreksi hasil. |
| Apotek | Menangani resep yang disetujui, menyiapkan obat, menandai siap diambil, dan mengonfirmasi penyerahan. |
| Administrator | Mengelola akun pengguna, peran, organisasi layanan kesehatan, dan audit aktivitas. |

## Panduan demonstrasi

### Masuk ke aplikasi

Pengguna masuk melalui halaman `/login`. Pendaftaran pasien tersedia pada
`/register`. Akun staf menggunakan peran dan organisasi yang ditetapkan
pengelola; autentikasi Clerk mewajibkan MFA untuk mengakses ruang staf.

Gunakan profil browser atau sesi terpisah untuk setiap peran saat mengikuti
alur demonstrasi.

### Alur perawatan utama

| Tahap | Tindakan | Hasil yang diamati |
| --- | --- | --- |
| 1. Hubungan perawatan | Dokter mengirim undangan menggunakan kode pasien; pasien menerima undangan. | Hubungan perawatan menjadi aktif pada kedua akun. |
| 2. Persetujuan pemeriksaan | Dokter meminta pemeriksaan ke laboratorium tujuan; pasien menyetujui pengiriman. | Permintaan masuk ke antrean laboratorium yang dituju. |
| 3. Hasil laboratorium | Petugas menerima permintaan, memulai pemeriksaan, dan menyerahkan hasil sintetis. | Pasien dan dokter yang berwenang dapat melihat hasil. |
| 4. Pengiriman resep | Dokter membuat resep; pasien memilih apotek dan menyetujui pengiriman. | Resep masuk ke antrean apotek pilihan pasien. |
| 5. Penyerahan obat | Apotek menerima resep, menyiapkan obat, menandai siap diambil, dan mengonfirmasi penyerahan. | Status resep menjadi selesai dan riwayat pasien diperbarui. |
| 6. Peninjauan riwayat | Pasien membuka riwayat perjalanan, notifikasi, serta riwayat akses. | Kegiatan yang dicatat dapat ditelusuri dari akun pasien. |

Pemeriksaan laboratorium dan penerbitan resep dapat dilakukan secara terpisah
sesuai kebutuhan pada hubungan perawatan aktif.

### Berbagi data melalui CarePass

1. Pasien memilih satu hasil laboratorium yang telah selesai atau satu resep,
   lalu membuat CarePass dengan masa berlaku **15 atau 60 menit**.
2. Pasien membagikan tautan atau QR kepada dokter tujuan. QR berisi tautan
   untuk membuka halaman penebusan CarePass. QR dapat dibuka melalui pemindai
   QR pada perangkat; dashboard dokter belum menyediakan pemindai kamera.
3. Dokter masuk, membuka tautan atau memasukkannya pada formulir, kemudian
   menekan **Tebus CarePass**. Jika tautan dibuka sebelum masuk, buka ulang
   tautan setelah proses masuk selesai.
4. Penebusan pertama mengikat izin kepada dokter dan kliniknya. Dokter dapat
   membaca satu data yang dibagikan selama izin masih berlaku.
5. Pasien dapat mencabut CarePass. Setelah dicabut atau kedaluwarsa, CarePass
   tersebut tidak lagi memberikan akses.

Pencabutan CarePass berlaku pada izin yang diberikan melalui CarePass tersebut.
Hak akses dari hubungan perawatan atau izin lain tetap mengikuti ketentuannya.

## Teknologi

Hygeia menggunakan antarmuka SvelteKit, backend Go, dan basis data PostgreSQL.
Autentikasi pengguna terintegrasi dengan Clerk.

| Komponen | Teknologi | Peran |
| --- | --- | --- |
| Antarmuka | TypeScript 6, Svelte 5, SvelteKit 3 | Halaman publik, autentikasi, dan ruang kerja pengguna. |
| Tampilan | HTML, CSS, Tailwind CSS 4 | Tata letak responsif dan komponen antarmuka. |
| Backend | Go 1.26, `net/http` | REST API, aturan akses, dan alur layanan. |
| Basis data | PostgreSQL, SQL | Penyimpanan data aplikasi. |
| Identitas | Clerk | Login dan pengaturan keamanan akun. |
| Notifikasi | Server-Sent Events (SSE) | Pembaruan notifikasi di dalam aplikasi. |
| Pengujian | Go testing, Vitest, Playwright | Pengujian backend, komponen, dan alur browser. |
| Pengembangan | Vite 8, pnpm, Docker Compose, GitHub Actions | Build, lingkungan lokal, container backend, dan pemeriksaan CI. |

Versi dependensi tercatat pada [go.mod](backend/go.mod),
[package.json](frontend/package.json), dan lockfile frontend.

## Instalasi lokal

### Prasyarat

- Git dan Go **1.26+**.
- Node.js **24** dan pnpm **11.18.0**, mengikuti CI.
- PostgreSQL yang dapat diakses; CI menggunakan **PostgreSQL 18**.
- Docker jika menjalankan backend melalui Compose.

Contoh perintah memakai PowerShell. Compose backend tidak membuat database:
siapkan database dan role terlebih dahulu mengikuti [panduan backend](backend/README.md#database-lokal).

### 1. Clone dan konfigurasi

```powershell
git clone git@github.com:mrizalbasri/Hygeia.git
cd Hygeia
Copy-Item backend/.env.example backend/.env
Copy-Item frontend/.env.example frontend/.env
```

Salin contoh hanya pada checkout baru yang belum mempunyai `.env`.
Lengkapi koneksi PostgreSQL, kunci sesi dan audit, serta password demo pada
`backend/.env` mengikuti keterangan di file contoh. Untuk demonstrasi lokal,
gunakan `APP_ENV=development` dan `AUTH_PROVIDER=local`.

Pada `frontend/.env`, gunakan `AUTH_PROVIDER=local` dan
`API_BASE_URL=http://127.0.0.1:8080`. Simpan nilai konfigurasi dan kredensial
hanya pada environment lokal. Petunjuk konfigurasi tersedia pada
[panduan backend](backend/README.md#menjalankan).

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
berada di `/login`, registrasi pasien di `/register`.

### Akun demo lokal

Akun berikut dibuat oleh perintah seed dan digunakan pada mode
`AUTH_PROVIDER=local` di lingkungan development/test. Login Clerk menggunakan
identitas yang dikelola pada instance Clerk terkait.

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
│   ├── cmd/          # API, migrasi, seed, dan utilitas pengelolaan
│   ├── db/           # Migrasi dan dukungan basis data
│   └── internal/     # Implementasi modul backend
├── frontend/
│   ├── src/lib/      # Komponen dan modul bersama
│   ├── src/routes/   # Halaman dan route SvelteKit
│   ├── static/       # Logo dan aset visual
│   └── tests/        # Pengujian alur browser
├── scripts/          # Skrip verifikasi lokal
└── .github/          # Workflow CI
```

## Panduan teknis

- [Backend](backend/README.md) — konfigurasi, basis data, Docker, dan API.
- [Frontend](frontend/README.md) — antarmuka, proxy API, dan pengujian browser.
- [Kontribusi](CONTRIBUTING.md) — alur perubahan dan pemeriksaan kode.

## Ruang lingkup

Prototipe ini mencakup alur layanan dan pertukaran data di dalam Hygeia dengan
data sintetis. Integrasi eksternal melalui SATUSEHAT atau FHIR berada di luar
lingkup implementasi saat ini.
