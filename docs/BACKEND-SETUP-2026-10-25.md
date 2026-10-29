# Hasil setup backend lokal

Tanggal: 25 Oktober 2026.

## Sudah berjalan dan diverifikasi

| Bagian | Hasil |
| --- | --- |
| Docker Desktop | Engine aktif |
| PostgreSQL | Instance `postgres-local` sehat, port loopback 5434, database Hygeia terpisah |
| Konfigurasi lokal | DATABASE_URL, DATABASE_URL_DOCKER, kunci sesi/audit, dan password demo tersedia; tidak dicetak atau disalin ke Git |
| CarePass origin | PUBLIC_APP_URL lokal disesuaikan ke `http://127.0.0.1:5188` |
| Backend container | `backend-api-1` aktif, runtime UID 100, host port `127.0.0.1:8080` |
| Migrasi | Versi 9 naik ke 10, `dirty=false` |
| Akun demo | Seed sukses, akun dan password yang sudah ada dipertahankan |
| Health/readiness | `/api/v1/health` dan `/api/v1/readyz` sehat |
| Healthcheck Compose | `docker compose up -d --wait` selesai dengan status Healthy |
| Tes Go dengan PostgreSQL | Seluruh paket source lulus dengan TEST_DATABASE_URL aktif, termasuk registrasi, atomicity sesi, koreksi append-only, dan rotasi audit |
| Analisis Go | `go vet` lulus |
| Browser live | Alur perawatan, lab, resep/apotek, consent, CarePass, emergency access, timeline, notifikasi, dan audit lulus |

Alur browser live membuat organisasi/akun dan data klinis sintetis dengan suffix
`adb4f33a`, yang dipertahankan untuk reproduksi audit sesuai desain tes. Tidak
ada data pasien nyata. Audit yang diperiksa memiliki 56 event dan `intact=true`.
Tes integrasi Go membuat dan menghapus schema sementara, terpisah dari tabel demo.

## Menjalankan ulang

Dari folder `backend`:

```powershell
docker compose up --build -d --wait
docker compose ps
```

Dari root repository:

```powershell
./scripts/Test-Backend.ps1 -UseLocalDatabase
```

Dari folder `frontend`, untuk web pada origin lokal yang sudah dikonfigurasi:

```powershell
pnpm dev --host 127.0.0.1 --port 5188
```

Jika port web diubah, sesuaikan PUBLIC_APP_URL pada `backend/.env`, lalu jalankan
ulang Compose agar QR CarePass menuju origin yang sama. Tidak perlu mengganti
kunci sesi/audit atau membuat ulang database.

## CI yang disiapkan

`.github/workflows/verify.yml` memiliki job backend dengan PostgreSQL sementara
dan job frontend dengan Chromium. Keduanya berjalan pada pull request, push ke
main, atau pemicu manual. Kredensial service CI bersifat sintetis dan hanya
untuk database sementara. Workflow dan script lokal sudah diperiksa sintaksnya;
eksekusi workflow di GitHub masih menunggu commit/push.

Konfigurasi mengikuti dokumentasi resmi [service PostgreSQL GitHub Actions](https://docs.github.com/en/actions/tutorials/use-containerized-services/create-postgresql-service-containers),
[setup-go](https://github.com/actions/setup-go),
[setup-node](https://github.com/actions/setup-node),
[pnpm action](https://github.com/pnpm/action-setup), dan
[Playwright CI](https://playwright.dev/docs/ci).

## Yang masih kurang

- **Publikasi perubahan:** file baru dan perbaikan masih di working tree; PR #20 belum mendapat commit lokal ini. CI belum memiliki hasil run di GitHub.
- **Deployment production:** host/domain, adapter frontend, HTTPS dan reverse proxy satu origin, serta pengaturan proxy SSE belum disiapkan dan diuji pada server.
- **Operasional production:** backup database beserta kunci audit, uji restore, pemantauan aplikasi, dan uji beban belum dibuktikan untuk Hygeia. Kunci lokal tidak boleh digunakan sebagai konfigurasi server baru.
- **Fitur akun lanjutan:** reset password, verifikasi email, dan MFA belum tersedia. Ini pekerjaan fitur tersendiri; tidak ditambahkan sebagai bagian setup lokal.

Setup ini membuktikan lingkungan development/demo lokal berfungsi. Ini belum
merupakan hasil verifikasi deployment production.
