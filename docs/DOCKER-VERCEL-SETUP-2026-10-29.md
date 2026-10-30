# Setup Docker dan Vercel — 29 Oktober 2026

## Publikasi melalui GitHub Actions

Jalur publikasi yang dipilih adalah GitHub Actions, bukan push dari komputer
lokal. Workflow `Verify` menjalankan tes backend/PostgreSQL, vet, build Docker,
serta check/lint/build/unit/E2E frontend. Job `publish-backend` baru berjalan
setelah kedua job pengujian berhasil.

- Push pada semua branch menjalankan pemeriksaan.
- Push ke `main` yang lolos pemeriksaan otomatis membangun dan mengunggah image.
- Pull request dan push branch pengembangan tidak mengunggah image.
- Untuk publikasi branch pengembangan secara eksplisit: Actions → Verify →
  Run workflow → pilih branch → aktifkan `publish_image`.
- Image selalu mendapat tag `mrizal23/hygeia-api:sha-<commit lengkap>`.
  Hanya rilis `main` yang juga memperbarui `latest`.
- Digest hasil upload dicantumkan pada ringkasan job. Gunakan tag commit atau
  digest pada konfigurasi Azure untuk memilih versi yang pasti.

Tambahkan repository secret **DOCKERHUB_TOKEN** melalui
[GitHub Actions secrets](https://github.com/mrizalbasri/Hygeia/settings/secrets/actions).
Token harus milik akun `mrizal23` dengan izin push ke repository `hygeia-api`.
Username sudah ditetapkan pada workflow. Jangan menaruh token pada source,
file env frontend, log, atau chat. Token ditambahkan sendiri oleh pengguna;
workflow memeriksa keberadaannya sebelum mencoba login.

Azure tetap dikonfigurasi sendiri oleh pengguna. Upload image tidak otomatis
mengubah container Azure; tahap deploy Azure belum diotomatisasi.

## Pembaruan build dan environment

Docker Desktop berhasil dinyalakan pada percobaan berikutnya. Image
`hygeia-api:staging-20261029` berhasil dibangun untuk Linux amd64 dan diberi tag
`mrizal23/hygeia-api:staging-20261029`. Smoke test container read-only/non-root
lulus: migrasi versi 12, dirty=false, readiness Neon connected, respons CSRF,
akses anonim HTTP 401, dan healthcheck healthy. Tidak ada file `.env` pada `/app`.

Upload ke Docker Hub dicoba tetapi ditolak dengan `insufficient_scope`.
Image **belum terunggah**. Login pada Docker CLI dengan akun yang mempunyai
hak push ke namespace `mrizal23` dan pastikan repository `hygeia-api` tersedia.
Setelah autentikasi, lanjutkan push tag di atas; tidak perlu build ulang.

Draft import frontend ada di `frontend/.env.vercel` dan diabaikan Git. Hanya
memuat `AUTH_PROVIDER`, `PUBLIC_CLERK_PUBLISHABLE_KEY`, dan `API_BASE_URL`.
Public key yang tersedia masih Clerk development, dan URL backend masih
`https://hygeia-backend.invalid`. Ganti dengan URL HTTPS Azure nyata dan key
Clerk yang sesuai dengan backend sebelum deployment; jangan import secret backend.

Azure Container Apps dikonfigurasi sendiri oleh pengguna. Pekerjaan ini
menyiapkan file repository; tidak membuat resource Azure, memublikasikan image,
atau melakukan deployment publik.

## Docker backend dan Docker Hub

Dari root repository, dengan Docker Linux engine aktif:

```powershell
./scripts/Build-BackendImage.ps1
# Memberi tag Docker Hub tanpa mengunggah:
./scripts/Build-BackendImage.ps1 -DockerHubUser NAMA_AKUN -Tag staging-001
```

Image Linux amd64 berisi `api`, `migrate`, `seed`, `seed-workflow`, dan
`link-clerk-user`. Proses berjalan sebagai non-root pada port 8080. Default
command `api`; tidak ada migrasi atau seed otomatis pada startup. Semua `.env*`
dikecualikan dari build context. Jangan memasukkan secret melalui build arguments.

Setelah image ditinjau dan siap dipublikasikan, login dan unggah sendiri:

```powershell
docker login
docker push NAMA_AKUN/hygeia-api:staging-001
```

Gunakan tag unik atau digest untuk rilis. Akses pull repository privat perlu
kamu konfigurasikan pada runtime. Jalankan `migrate up` sekali sebelum rilis,
bukan pada setiap replica API. Command migrasi memakai `DATABASE_URL_UNPOOLED`
jika tersedia dan menolak host pooler. Seed dipanggil eksplisit; binary tetap
menolak lingkungan selain development/test.

## Docker lokal dengan Neon staging

Gunakan `backend/.env.staging` yang sudah disiapkan. Dari folder `backend`:

```powershell
docker compose -f docker-compose.staging.yml build
docker compose -f docker-compose.staging.yml run --rm api migrate up
docker compose -f docker-compose.staging.yml up -d --wait
Invoke-RestMethod http://127.0.0.1:8082/api/v1/readyz
docker compose -f docker-compose.staging.yml logs -f api
docker compose -f docker-compose.staging.yml down
```

Port host **8082**, container 8080. Compose memakai filesystem read-only dan
tanpa Linux capabilities; tidak membuat container database. Port ini berbeda
dari API lokal 8080 dan script Go staging 8081.

Untuk memakai Docker ini dari frontend lokal, pasang
`API_BASE_URL=http://127.0.0.1:8082` di `frontend/.env`, lalu restart frontend.
Gunakan hostname frontend yang konsisten (`localhost` atau `127.0.0.1`) dan
cocokkan dengan `PUBLIC_APP_URL` backend untuk pemeriksaan CSRF.

## Environment backend yang kamu pasang sendiri

Template tanpa secret: `backend/.env.production.example`.

| Variabel | Nilai |
| --- | --- |
| `PORT` | `8080` |
| `APP_ENV` | `production` untuk akses publik |
| `AUTH_PROVIDER` | `clerk` |
| `DATABASE_URL` | Neon pooled, role runtime, `sslmode=verify-full` |
| `DATABASE_URL_UNPOOLED` | Neon direct dengan role migrasi; hanya diperlukan pada langkah migrasi |
| `CLERK_PUBLISHABLE_KEY` | Public key instance Clerk yang sama dengan frontend |
| `CLERK_SECRET_KEY` | Secret key Clerk, hanya pada backend |
| `CLERK_AUTHORIZED_PARTIES` | Daftar origin HTTPS frontend tanpa wildcard |
| `PUBLIC_APP_URL` | Origin HTTPS frontend utama; dipakai CarePass dan CSRF |
| `SESSION_SECRET` | Secret 64 karakter hex |
| `AUDIT_HMAC_KEY` | Secret audit 64 karakter hex, berbeda dari secret sesi |
| `TRUSTED_PROXY_CIDRS` | Jaringan proxy aktual yang sudah diverifikasi; default kosong |

Mode production menolak Clerk development keys. `.env.staging` saat ini masih
memakai Clerk development untuk verifikasi lokal. Sebelum akses publik, siapkan
Clerk live, role database runtime dengan hak minimum, dan secret lingkungan
terpisah. Pertahankan semua kunci yang diperlukan untuk memverifikasi audit
yang sudah ada; jangan menggantinya tanpa rencana pemulihan.

Untuk runtime yang kamu konfigurasi: image menerima HTTP pada port 8080,
command normal `api`, command migrasi `migrate up`. Liveness tersedia di
`/api/v1/health`; readiness di `/api/v1/readyz`. Docker healthcheck sudah ada,
tetapi probe platform Azure tetap perlu kamu konfigurasi sendiri.

## Frontend Vercel

Gunakan root directory **frontend**, framework SvelteKit, Node.js 24.
`vercel.json` menetapkan install/build command; `packageManager` menetapkan
pnpm 11.18.0. Saat `VERCEL` tersedia, adapter Vercel memakai region `sin1`.
Build lokal tanpa `VERCEL` tetap memakai adapter-auto.

| Environment Vercel | Nilai |
| --- | --- |
| `API_BASE_URL` | Origin HTTPS backend setelah kamu setup Azure, tanpa `/api` |
| `AUTH_PROVIDER` | `clerk` |
| `PUBLIC_CLERK_PUBLISHABLE_KEY` | Public key yang sesuai dengan backend |

Jangan pasang database URL, Clerk secret key, atau kunci audit di frontend.
Pisahkan backend/database preview dari production. Origin preview yang dipakai
untuk login dan mutasi perlu diizinkan secara eksplisit, termasuk sebagai
origin utama backend preview; hostname preview dinamis tidak otomatis diizinkan.

Browser mengakses `/api/v1/*` pada origin frontend. Route server SvelteKit
meneruskan cookie sesi/CSRF, body, status error, multiple Set-Cookie, dan SSE
ke backend tetap. Cookie lain dan header forwarding dari klien dibuang.
Backend mempercayai origin frontend utama dari `PUBLIC_APP_URL` untuk CSRF,
tetapi tetap mewajibkan cookie dan token CSRF yang valid. CORS tidak dibuka.

Stream dibatasi 180 detik sebelum batas fungsi 240 detik; EventSource yang
sudah ada akan menyambung kembali. Proxy ini memakai fungsi Vercel; uji biaya,
latensi, dan koneksi realtime pada hosting nyata masih diperlukan.

## Verifikasi dan batas

- Tes CSRF lintas frontend/backend lulus: host tepat dengan token diterima;
  host lain, hostname serupa, dan token kosong ditolak.
- Tes proxy lulus: auth/cookie/CSRF, body, SSE, multiple Set-Cookie, error, dan
  kegagalan koneksi tanpa membocorkan detail backend.
- `go test` dan `go vet` pada auth/httpapi/cmd-api lulus.
- Seluruh lima binary berhasil dikompilasi silang ke Linux amd64 dengan CGO
  dimatikan, sesuai target image.
- `pnpm check`, lint, build lokal, dan Vitest server lulus; suite browser
  `clinical-workflows.e2e.js` lulus, termasuk penolakan akses dan SSE sesi berakhir.
- Smoke test route SvelteKit nyata ke API dengan database Neon lulus:
  readiness connected, cookie/token CSRF diteruskan, dan akses anonim HTTP 401.
- Docker Compose staging lolos `config --quiet`; entrypoint lolos sintaks Bash.
- Percobaan awal Docker engine gagal menyala; build dan smoke test berhasil
  pada percobaan berikutnya (lihat pembaruan di atas). Upload masih tertahan izin.
- Pengemasan adapter Vercel di Windows gagal karena izin symlink (`EPERM`);
  pengemasan deployment masih perlu diverifikasi di Linux/Vercel.
- Tidak ada resource Azure atau deployment Vercel dibuat pada pekerjaan ini.

Referensi:
[SvelteKit di Vercel](https://vercel.com/docs/frameworks/full-stack/sveltekit),
[Azure container images](https://learn.microsoft.com/en-us/azure/container-apps/containers),
[Neon pooling](https://neon.com/docs/connect/connection-pooling).
