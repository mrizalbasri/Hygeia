# Hygeia

Prototipe pertukaran data kesehatan antara pasien, dokter, laboratorium, dan
apotek, dengan persetujuan pasien dan audit akses yang dapat diverifikasi.
Dataset demo hanya menggunakan data sintetis.

## Status awal

- Frontend SvelteKit, TypeScript, dan Tailwind CSS.
- Backend Go dengan migrasi, akun demo, sesi cookie, CSRF, login/logout, dan `/me`.
- API care episode (undangan, persetujuan, penolakan, pengakhiran), policy akses,
  audit HMAC per pasien, riwayat akses, dan verifikasi integritas.
- API laboratorium: pemilihan lab, permintaan tes, persetujuan pasien, antrean
  staf lab, pengiriman hasil terstruktur, serta pembacaan hasil terlindungi.
- Database development terpisah bernama `hygeia` pada instance PostgreSQL Docker
  lokal yang sudah tersedia.
- Dokumen kebutuhan bisnis, produk, arsitektur, dan design system.

Halaman login dan dashboard awal per peran sudah terhubung ke backend. API care
episode, laboratorium, dan audit tersedia; tampilan untuk alur tersebut belum
dihubungkan. Consent untuk satu hasil lab sudah tersedia, termasuk durasi dan
pencabutan oleh pasien. Resep dan akses darurat belum diimplementasikan.

## Menjalankan development

Frontend (pnpm):

```powershell
cd frontend
pnpm install
pnpm dev
```

Backend (Go 1.26+), pada terminal terpisah:

```powershell
cd backend
go run ./cmd/migrate up
go run ./cmd/seed
go run ./cmd/api
```

Sebelum menjalankan backend, pastikan PostgreSQL tersedia dan `backend/.env`
berisi `DATABASE_URL`, `SESSION_SECRET`, `AUDIT_HMAC_KEY`, `APP_ENV=development`, dan `DEMO_PASSWORD`
yang sesuai. Untuk komputer baru, salin
`backend/.env.example` ke `backend/.env` dan isi kredensial lokal.
File `.env`, dependensi, cache, dan hasil build tidak disertakan dalam Git.
Buka `http://localhost:5173/login`. Email akun demo tercantum pada panduan backend;
password mengikuti `DEMO_PASSWORD` saat seed pertama kali dijalankan.

Lihat [panduan backend](backend/README.md) untuk konfigurasi PostgreSQL, endpoint
health/readiness, dan cara mengarahkan cache Go ke drive D jika diperlukan.

## Dokumentasi

- [Business Requirements](docs/01-BRD.md)
- [Product Requirements](docs/02-PRD.md)
- [System Architecture](docs/03-ARCHITECTURE.md)
- [Design System](docs/04-DESIGN.md)
- [Kontrak API dan panduan frontend](docs/05-BACKEND-API.md)
- [Keputusan implementasi care episode dan audit](docs/06-BACKEND-CARE.md)
