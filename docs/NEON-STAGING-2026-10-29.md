# Database staging Neon — 29 Oktober 2026

## Lingkungan

- Proyek yang dipakai kembali: **Hygeia**, `calm-lab-14260245`.
- Region: AWS Singapura (`aws-ap-southeast-1`), PostgreSQL 18.
- Branch baru: `staging`, `br-withered-term-az35itfh`.
- Database aplikasi staging: `hygeia_staging`, pemilik `hygeia_owner`.
- Skema berasal dari migrasi repository 000001–000012.
- Kredensial ada di `backend/.env.staging`, diabaikan Git. Contoh tanpa secret:
  `backend/.env.staging.example`.

Branch `development` sudah ada sebelum pekerjaan ini. Database `hygeia` di
branch tersebut mencatat versi 13, tetapi berbeda dari migrasi lokal:
kolom `cancellation_note` belum tersedia dan terdapat kolom profil pasien
tambahan. Angka versi saja tidak membuktikan kesesuaian skema. Database tersebut
tidak diubah atau diturunkan versinya. Branch staging mewarisi database itu;
aplikasi hanya diarahkan ke database **baru** `hygeia_staging` yang dibangun dari
migrasi repository. Tidak ada penyalinan data pasien dari PostgreSQL lokal.

## Menjalankan dari root repository

```powershell
./scripts/Invoke-Staging.ps1 -Action Version
./scripts/Invoke-Staging.ps1 -Action Migrate
./scripts/Invoke-Staging.ps1 -Action Api
```

API memakai port 8081 agar konfigurasi lokal port 8080 tetap terpisah.
Periksa kesiapan dengan:

```powershell
Invoke-RestMethod http://127.0.0.1:8081/api/v1/readyz
```

Script memuat `.env.staging` hanya ke environment proses anak dan memulihkan
environment pemanggil setelah selesai. `backend/.env` lokal tidak diubah.
Pemanggilan `go run ./cmd/api` secara langsung tetap memakai konfigurasi lokal.
Script menolak database selain `hygeia_staging`, host selain Neon, TLS tanpa
verifikasi sertifikat, dan koneksi pooled untuk migrasi.

## Jenis koneksi

- `DATABASE_URL`: pooled, dipakai API Go melalui pool pgx yang sudah ada.
- `DATABASE_URL_UNPOOLED`: direct, dipakai migrasi dan backup/restore.
- `DATABASE_URL_DOCKER`: pooled, disediakan bila nanti API dijalankan di Docker
  dengan file environment staging yang dipilih secara eksplisit.
- Kedua koneksi memakai `sslmode=verify-full`.

Referensi: [Neon connection pooling](https://neon.com/docs/connect/connection-pooling).
Jangan menjalankan test yang bergantung pada `search_path` melalui pooler;
fixture integrasi repository membutuhkan koneksi direct dan schema acak.

Pada pemeriksaan nyata, koneksi direct mengabaikan `search_path` pada startup,
tetapi menerima pengaturan eksplisit setelah koneksi terbuka. Fixture care dan
Clerk sekarang memakai `AfterConnect` untuk mengatur `search_path` dan memeriksa
`current_schema()` sebelum menjalankan migrasi atau menulis data sintetis.

## Hasil verifikasi

- Migrasi `up` dijalankan dua kali: versi 12, `dirty=false`, tanpa duplikasi.
- API pada port 8081: `/api/v1/readyz` mengembalikan database connected,
  provider autentikasi Clerk, dan `/api/v1/me` anonim ditolak dengan HTTP 401.
- Integrasi direct PostgreSQL Neon lulus untuk
  `TestWorkflowCancellationIntegration`, `TestClerkStaffAssignmentIntegration`,
  dan `TestClerkJWTAndIdentityIntegration` setelah perbaikan isolasi fixture.
- `go test ./internal/database ./cmd/migrate -count=1` lulus;
  paket `cmd/migrate` tidak mempunyai test file.
- `go vet ./cmd/migrate ./internal/database ./internal/auth ./internal/care` lulus.
- Parser PowerShell menerima `Invoke-Staging.ps1`; environment rahasia staging
  terverifikasi diabaikan Git.
- Setelah pengujian: pengguna dan audit di schema public tetap kosong; tidak ada
  schema fixture care/Clerk yang tertinggal.
- Suite backend lengkap dan E2E browser tidak dijalankan pada pekerjaan ini.

## Batas kesiapan publik

Ini adalah database online untuk staging, dengan verifikasi API di komputer
lokal. Konfigurasi tetap memakai Clerk development dan secret aplikasi lokal
untuk pemeriksaan tersebut. Belum ada deployment API/frontend publik.

Sebelum hosting publik: siapkan domain/HTTPS, origin Clerk, konfigurasi produksi
yang sesuai, secret aplikasi terpisah, role runtime dengan hak minimum, dan
backup/restore beserta penyimpanan kunci audit. Koneksi awal ini menggunakan role
pemilik untuk migrasi dan pemeriksaan API; pemisahan role runtime masih perlu
dikerjakan sebelum penggunaan publik. Akun admin manusia belum diprovisikan.

Pengaturan suspend menggunakan default akun Neon; percobaan mengatur interval
secara eksplisit ditolak oleh batas akun. Tidak ada upgrade paket dilakukan.
