# Hygeia

Prototipe pertukaran data kesehatan antara pasien, dokter, laboratorium, dan
apotek, dengan persetujuan pasien dan audit akses yang dapat diverifikasi.
Dataset demo hanya menggunakan data sintetis.

## Status implementasi

- Frontend SvelteKit, TypeScript, dan Tailwind CSS.
- Backend Go dengan migrasi, akun demo, sesi cookie, CSRF, login/logout, dan `/me`.
- API care episode (undangan, persetujuan, penolakan, pengakhiran), policy akses,
  audit HMAC per pasien, riwayat akses, dan verifikasi integritas.
- API laboratorium: pemilihan lab, permintaan tes, persetujuan pasien, antrean
  staf lab, pengiriman hasil terstruktur, serta pembacaan hasil terlindungi.
- API resep dan apotek: resep dalam episode aktif, pasien memilih apotek dan
  menyetujui pengiriman, antrean apotek sampai selesai, serta audit setiap akses.
- Timeline perjalanan pasien: kejadian episode, lab, resep, dan consent;
  filter jenis dan pagination cursor, hanya untuk pasien pemilik.
- Database development terpisah bernama `hygeia` pada instance PostgreSQL Docker
  lokal yang sudah tersedia.
- Dokumen kebutuhan bisnis, produk, arsitektur, dan design system.

Frontend menyediakan landing page, login/logout melalui proxy `/api`, dashboard
per peran, serta workflow pasien, dokter, laboratorium, apotek, dan admin:
perawatan, lab, resep, consent, CarePass, akses darurat, timeline, audit,
notifikasi, antrean lab/apotek, pengelolaan akun dan organisasi, serta tinjauan
akses darurat. Kontrak backend dan
[rencana handoff](docs/plans/2026-10-15-patient-doctor-and-provider-handoff.md)
menjadi acuan integrasi. Bukti dan batas kesiapan dicatat pada
[hasil verifikasi backend](docs/14-BACKEND-READINESS.md).

## Menjalankan development

Backend (Go 1.26+), dari root repository:

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
Setelah API berjalan, periksa koneksi backend dari terminal lain:

```powershell
Invoke-RestMethod http://127.0.0.1:8080/api/v1/health
Invoke-RestMethod http://127.0.0.1:8080/api/v1/readyz
```

Login web tersedia pada `/login`; frontend mengambil CSRF dan mengirim login
ke backend melalui proxy satu origin. Email akun demo tercantum pada panduan
backend; password mengikuti `DEMO_PASSWORD` saat seed pertama kali dijalankan.

Frontend dapat dijalankan secara terpisah:

```powershell
cd frontend
pnpm install
pnpm dev
```

Buka URL root yang ditampilkan Vite (default `http://localhost:5173/`) untuk
landing page, atau `/login` untuk masuk ke dashboard. Proxy development menuju
backend pada `127.0.0.1:8080`. Untuk produksi, arahkan `/api` ke backend pada
origin yang sama. Lihat [panduan frontend](frontend/README.md) untuk cakupan dan
pengujian serta [kontrak API](docs/05-BACKEND-API.md) untuk integrasi lanjutan.

Lihat [panduan backend](backend/README.md) untuk konfigurasi PostgreSQL, endpoint
health/readiness, dan cara mengarahkan cache Go ke drive D jika diperlukan.

## Dokumentasi

- [Business Requirements](docs/01-BRD.md)
- [Product Requirements](docs/02-PRD.md)
- [System Architecture](docs/03-ARCHITECTURE.md)
- [Design System](docs/04-DESIGN.md)
- [Kontrak API dan panduan frontend](docs/05-BACKEND-API.md)
- [Keputusan implementasi care episode dan audit](docs/06-BACKEND-CARE.md)
- [Kontrak resep dan alur apotek](docs/07-BACKEND-PRESCRIPTIONS.md)
- [Kontrak timeline perjalanan pasien](docs/08-BACKEND-TIMELINE.md)
- [Rencana implementasi resep dan apotek](docs/plans/2026-10-11-prescriptions-pharmacy.md)
