# Kesiapan backend Hygeia

Verifikasi: 17 Oktober 2026. Baseline kode: `8fbb18b98037bd94ee28d8c68a4e11eb404e013c`.

Backend pada baseline ini siap untuk demo sintetis dan integrasi frontend lokal.
Seluruh modul di bawah tersedia dan lolos pengujian PostgreSQL. Pemeriksaan ini
tidak menyatakan layanan sudah di-deploy atau siap untuk data pasien nyata.
Tidak ditemukan kegagalan kode pada suite yang dijalankan; perubahan penyerahan
ini memperbaiki panduan yang tertinggal dari implementasi.

## Cakupan dan bukti

| Bagian | Hasil pemeriksaan |
| --- | --- |
| Autentikasi | Login, identitas, CSRF, pembatasan login, pencabutan sesi dan akun nonaktif |
| Episode perawatan | Invite, accept, reject, end, kepemilikan, konflik, dan rollback audit |
| Laboratorium | Persetujuan pasien, antrean sesuai organisasi, accepted/in_progress, hasil immutable, akses hasil terlindungi |
| Resep/apotek | Pilihan apotek oleh pasien, antrean terbatas, accepted/preparing/ready/completed, akses ditutup setelah selesai |
| Consent | Satu hasil lab, durasi, revoke, expiry, dan klinik penerima |
| Audit/timeline | Rantai HMAC, deteksi manipulasi, transaksi, isolasi pasien, filter dan cursor |
| Admin | Akun, organisasi, ledger aktivitas, pembatasan akses dan atomicity |
| Notifikasi/SSE | Inbox, read state, isolasi pengguna, replay, urutan commit, batas koneksi dan graceful shutdown |
| Akses darurat | Scope baca terbatas, alasan, expiry setelah menunggu lock, notifikasi dan tinjauan |
| CarePass | Token terbatas, satu penebus, revoke, expiry, scope resource dan klinik |

Hasil aktual:

- PASS: `go test ./cmd/... ./db/... ./internal/... -count=1 -json` dengan
  `TEST_DATABASE_URL`: 60 tes tingkat atas, 194 hasil pass termasuk subtest,
  0 gagal, 0 tes dilewati. Paket tanpa file tes tidak dihitung sebagai tes
  dilewati. Kasus lintas modul berada di `internal/care`.
- PASS: `go vet ./cmd/... ./db/... ./internal/...`.
- PASS: build binary `cmd/api`, `cmd/migrate`, dan `cmd/seed`.
- PASS: binary migrate `up` dua kali pada database kosong terpisah;
  versi akhir 9, `dirty: false` pada kedua pemanggilan.
- PASS: binary seed dua kali; tetap 7 akun sintetis.
- PASS: server binary melalui HTTP nyata pada port 18080: health/readiness,
  login, `/me`, logout dan penolakan `/me` setelah logout untuk patient,
  doctor, lab, pharmacy, admin. Akun pasien menerima kode `HYG-DEMO-001`;
  verifikasi rantai pasien mengembalikan `intact: true`.

Lingkungan tes: Windows, Go 1.26.4, PostgreSQL 18.1, loopback port 15432.
Tes domain memakai schema acak dengan cleanup; smoke binary memakai database
`hygeia_readiness` pada cluster sementara. Database aplikasi dan `.env` lokal
tidak diubah. Proses API smoke dan PostgreSQL tes dihentikan setelah verifikasi.
Log JSON tes, binary, dan skrip smoke lokal berada di
`backend/bin/readiness-20261017-native/`, yang diabaikan Git.

## Menjalankan untuk bekerja bersama frontend

1. Aktifkan PostgreSQL aplikasi. Konfigurasi lokal terdokumentasi memakai
   container `postgres-local`, port 5434; Docker Desktop tidak aktif pada saat
   pemeriksaan ini. Alternatifnya gunakan instance development lain dan arahkan
   `DATABASE_URL` ke database Hygeia yang benar.
2. Isi `backend/.env` sesuai [panduan backend](../backend/README.md).
   Pertahankan kunci audit pada database yang sudah memiliki event.
3. Dari direktori `backend`, jalankan:

   ```powershell
   go run ./cmd/migrate up
   go run ./cmd/migrate version
   go run ./cmd/seed
   go run ./cmd/api
   ```

4. Dari terminal lain, pastikan keduanya berhasil:

   ```powershell
   Invoke-RestMethod http://127.0.0.1:8080/api/v1/health
   Invoke-RestMethod http://127.0.0.1:8080/api/v1/readyz
   ```

5. Jalankan frontend mengikuti [panduan frontend](../frontend/README.md).
   Browser memakai `/api/v1` melalui proxy satu origin. Port API default 8080;
   kredensial demo mengikuti `DEMO_PASSWORD` saat seed pertama.

Readiness saat ini hanya memeriksa koneksi DB. Pastikan CLI migrate menunjukkan
versi 9 dan `dirty: false`; HTTP 200 pada `/readyz` saja belum membuktikan skema
siap digunakan.

## Kontrak handoff lab dan apotek

Acuan request/response adalah [kontrak API](05-BACKEND-API.md),
[resep/apotek](07-BACKEND-PRESCRIPTIONS.md), dan
[rencana handoff](plans/2026-10-15-patient-doctor-and-provider-handoff.md).

| Peran | Urutan kerja | Batas data |
| --- | --- | --- |
| Lab | GET `/lab-requests` → GET detail → PATCH status accepted → in_progress → POST result | Hanya penugasan yang disetujui ke lab sendiri; receipt hasil bukan izin membaca nilai kembali |
| Apotek | GET `/prescriptions` → GET detail → PATCH status accepted → preparing → ready → completed | Hanya resep yang disetujui ke apotek sendiri; completed keluar antrean dan akses provider ditutup |

Semua path di atas diawali `/api/v1`. Setiap POST/PATCH membawa cookie CSRF dan
header `X-CSRF-Token`. Status dan izin berasal dari backend. Tangani 409 dengan
membaca status terbaru; hapus detail lama setelah 401/403. Pagination antrean
menggunakan limit/offset. Jangan memakai payload pasien/dokter sebagai proyeksi
provider. SSE hanya memberi notifikasi; baca resource kembali melalui API
yang menjalankan otorisasi.

Seluruh PR backend #10–#15 serta frontend #16, #17, dan #19 sudah merged. PR #20
menambahkan antarmuka laboratorium, apotek, dan admin di atas API yang sudah
tersedia, termasuk lifecycle antrean, penyerahan hasil, pengelolaan akun dan
organisasi, verifikasi rantai audit, serta tinjauan akses darurat. Verifikasi UI
dan E2E dicatat pada PR tersebut. Dokumen review 12 Oktober adalah catatan
historis dan tidak menggantikan status terkini ini.

## Mengulang pengujian backend

Gunakan database khusus tes dengan hak CREATE/DROP schema. Dari `backend`:

```powershell
$env:TEST_DATABASE_URL = 'postgres://USER:PASSWORD@127.0.0.1:PORT/DATABASE?sslmode=disable'
go test ./cmd/... ./db/... ./internal/... -count=1
go vet ./cmd/... ./db/... ./internal/...
go build -o bin/hygeia-api.exe ./cmd/api
```

Tanpa `TEST_DATABASE_URL`, tes integrasi dilewati. Daftar paket eksplisit
menghindari arsip lokal di `backend/bin`. Jangan memakai database produksi.
Tes lab dan resep juga dapat dijalankan terpisah:

```powershell
go test ./internal/care -run 'Laboratory|Prescription' -count=1 -v
```

## Batas deployment

Deployment belum diuji pada server. Repository belum menyediakan Dockerfile,
Compose Hygeia, atau pipeline CI backend. Draft arsitektur menjelaskan target
deployment, bukan konfigurasi yang sudah tersedia. Sebelum deployment perlu
menetapkan host, HTTPS/proxy satu origin, penyimpanan secret, database TLS,
backup dan pemulihan DB beserta kunci audit, monitoring, serta uji beban.
Proxy SSE perlu menonaktifkan buffering. Limiter login dan batas koneksi SSE
berlaku per proses; kebijakan trusted proxy/distribusi belum ditetapkan.

Registrasi publik, reset password, verifikasi email, MFA, unggahan PDF,
koreksi hasil, rotasi kunci audit, dan checkpoint audit eksternal belum tersedia.
Kesiapan demo ini tidak memperluas scope produk ke fitur-fitur tersebut.
