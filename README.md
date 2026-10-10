# Hygeia

Prototipe pertukaran data kesehatan antara pasien, dokter, laboratorium, dan
apotek, dengan persetujuan pasien dan audit akses yang dapat diverifikasi.
Dataset demo hanya menggunakan data sintetis.

## Status awal

- Frontend SvelteKit, TypeScript, dan Tailwind CSS.
- Backend Go dengan health check dan readiness PostgreSQL.
- Database development terpisah bernama `hygeia` pada instance PostgreSQL Docker
  lokal yang sudah tersedia.
- Dokumen kebutuhan bisnis, produk, arsitektur, dan design system.

Fitur login, migrasi tabel bisnis, alur perawatan, consent, dan audit belum
diimplementasikan.

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
go run ./cmd/api
```

Sebelum menjalankan backend, pastikan PostgreSQL tersedia dan `backend/.env`
berisi `DATABASE_URL` yang sesuai. Untuk komputer baru, salin
`backend/.env.example` ke `backend/.env` dan isi kredensial lokal.
File `.env`, dependensi, cache, dan hasil build tidak disertakan dalam Git.

Lihat [panduan backend](backend/README.md) untuk konfigurasi PostgreSQL, endpoint
health/readiness, dan cara mengarahkan cache Go ke drive D jika diperlukan.

## Dokumentasi

- [Business Requirements](docs/01-BRD.md)
- [Product Requirements](docs/02-PRD.md)
- [System Architecture](docs/03-ARCHITECTURE.md)
- [Design System](docs/04-DESIGN.md)
