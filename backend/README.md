# Hygeia Backend

[README proyek](../README.md) · [Kontribusi](../CONTRIBUTING.md)

REST API Go (`net/http`) dengan PostgreSQL melalui connection pool `pgx`.
Memerlukan Go 1.26 atau lebih baru.

## Teknologi

| Bagian                   | Stack                                     |
| ------------------------ | ----------------------------------------- |
| Bahasa dan HTTP          | Go 1.26 dan standard library `net/http`   |
| Database                 | PostgreSQL, SQL, dan `pgx/v5`             |
| Migrasi                  | `golang-migrate` dengan SQL tertanam      |
| Autentikasi              | Argon2id, cookie sesi, dan `gorilla/csrf` |
| Audit                    | Rantai HMAC-SHA256 per pasien             |
| Distribusi dan pengujian | Docker Compose, Go testing, dan `go vet`  |

## Database lokal

Konfigurasi workstation pengembangan memakai instance PostgreSQL Docker yang
sudah ada, tanpa container baru. Checkout lain dapat memakai instance PostgreSQL
sendiri dengan menyesuaikan `DATABASE_URL`; nama container dan port di bawah
adalah contoh lingkungan lokal:

| Konfigurasi | Nilai                                   |
| ----------- | --------------------------------------- |
| Container   | `postgres-local`                        |
| Host        | `127.0.0.1`                             |
| Port host   | `5434` (port internal container `5432`) |
| Database    | `hygeia`                                |
| User        | `hygeia_app`                            |
| Password    | Tersimpan di `backend/.env` lokal       |

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
dan 9 CarePass berbatas waktu. Migrasi 10 menambah koreksi hasil lab yang
append-only. Backend inti dan fitur tambahan kini tersedia:

| Modul                               | Implementasi                                    |
| ----------------------------------- | ------------------------------------------------- |
| Admin akun/organisasi dan ledger    | [implementasi admin](internal/admin)              |
| Inbox dan read state                | [implementasi notifikasi](internal/notification) |
| Akses darurat 60 menit dan tinjauan | [implementasi akses darurat](internal/breakglass)  |
| CarePass/QR dan pencabutan          | [implementasi CarePass](internal/carepass)        |
| SSE/replay notifikasi               | [implementasi realtime](internal/notification)        |

CarePass memerlukan `PUBLIC_APP_URL` berupa origin aplikasi, default development
`http://localhost:5173`. Production wajib origin HTTPS. QR mengarah ke route
frontend `/carepass#token=...`; UI scan/redeem harus memakai kontrak backend.

## Proxy tepercaya dan kuota autentikasi

`TRUSTED_PROXY_CIDRS` opsional, berupa daftar CIDR dipisahkan koma. Nilai kosong
mengabaikan semua header forwarding. CIDR tidak valid menghentikan startup
sebelum koneksi database. Gunakan rentang sesempit mungkin milik proxy aktual,
misalnya `127.0.0.1/32,::1/128` untuk proxy loopback; jangan menyalin contoh tanpa
memeriksa alamat socket yang benar-benar dilihat API.

Hanya socket dari proxy tepercaya yang boleh memasok `X-Forwarded-For` (XFF).
Seluruh alamat divalidasi, lalu rantai ditelusuri dari kanan ke alamat pertama
yang tidak tepercaya. IPv4-mapped IPv6 dinormalisasi ke IPv4. Header kosong,
rusak, atau seluruh rantai tepercaya kembali ke IP socket. Login dan registrasi
tetap berbagi burst 5 percobaan per identitas, isi ulang 1 token per 12 detik.

Proxy wajib menimpa XFF dari pengunjung atau menyaringnya sebelum menyusun
rantai dari alamat koneksi yang diamati. Pada proxy append-only yang membiarkan
prefix rusak, fallback socket menjadi kuota bersama tambahan; klien dapat
menghabiskan kuota klien dan kuota fallback. Ini konsekuensi kebijakan fallback,
bukan pemisahan kuota penuh untuk header rusak. Tanpa XFF yang benar, pengguna
tetap berbagi kuota socket. Vite development saat ini tidak otomatis
menghasilkan XFF. Konfigurasi produksi harus diuji pada topologi hosting tujuan.

## Demo perjalanan sintetis

Perintah terpisah dari seed akun lama, dijalankan dari folder `backend` setelah
migrasi:

```powershell
go run ./cmd/seed-workflow
```

Hanya `APP_ENV=development` atau `test` yang diterima. `DATABASE_URL`,
`DEMO_PASSWORD` (12–1024 byte), dan `AUDIT_HMAC_KEY` dibaca dari konfigurasi lokal.
Perintah membuat lima akun terpisah:

| Peran  | Email                             |
| ------ | --------------------------------- |
| Pasien | `pilot.patient@demo.hygeia.test`  |
| Dokter | `pilot.doctor@demo.hygeia.test`   |
| Lab    | `pilot.lab@demo.hygeia.test`      |
| Apotek | `pilot.pharmacy@demo.hygeia.test` |
| Admin  | `pilot.admin@demo.hygeia.test`    |

Kode pasien `HYG-PILOT-SYNTH-001`. Contoh berisi satu perawatan aktif, satu hasil
lab, satu antrean pemeriksaan, satu resep siap diambil, satu antrean resep,
timeline, notifikasi, dan audit HMAC dengan basis `synthetic_demo_seed`.
Data bukan hasil diagnosis atau instruksi pengobatan. Seed tidak menambahkan
consent atau CarePass otomatis; demonstrasikan izin secara eksplisit lewat UI.

Seluruh pengisian memakai satu transaksi dan kunci seed untuk mencegah duplikasi
serentak. Pengulangan mempertahankan password, nama, dan perkembangan alur;
perintah tidak mereset demo yang sudah selesai. Konflik namespace akun/organisasi
membatalkan transaksi tanpa mengambil alih data lama. Akun demo lama seperti
Aulia tetap terpisah. Mengubah `DEMO_PASSWORD` tidak mengubah akun yang sudah ada.

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
API dan container tidak menjalankan migrasi otomatis. Jalankan migrasi sebagai
langkah terpisah sebelum merilis API, dengan koneksi direct untuk Neon.

Seed hanya diizinkan saat `APP_ENV=development` atau `test`, memakai
`DEMO_PASSWORD` dari `.env` (minimal 12 byte). Seed bisa diulang dan tidak
mengubah password atau akun yang sudah ada. Mengganti `DEMO_PASSWORD` setelah
seed tidak otomatis mengganti password akun yang sudah dibuat.

| Peran                    | Email sintetis            |
| ------------------------ | ------------------------- |
| Pasien                   | `aulia@demo.hygeia.test`  |
| Dokter episode           | `maya@demo.hygeia.test`   |
| Dokter lintas organisasi | `budi@demo.hygeia.test`   |
| Dokter IGD               | `rina@demo.hygeia.test`   |
| Laboratorium             | `lab@demo.hygeia.test`    |
| Apotek                   | `apotek@demo.hygeia.test` |
| Admin                    | `admin@demo.hygeia.test`  |

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
Untuk rotasi, pindahkan kunci aktif lama ke `AUDIT_HMAC_KEY_PREVIOUS` dan buat
`AUDIT_HMAC_KEY` baru. Daftar kunci lama dipisahkan koma; simpan semua kunci yang
masih diperlukan untuk memverifikasi riwayat. Penulisan baru memakai kunci aktif,
verifikasi mencoba kunci aktif dan kunci lama. Konfigurasi kunci lama yang tidak
valid menggagalkan startup. Jangan memakai placeholder untuk menjalankan API.

`APP_ENV=production` mengaktifkan flag cookie `Secure`; mode tersebut membutuhkan
HTTPS di reverse proxy. Development lokal menggunakan cookie tanpa `Secure`.
Gunakan satu origin dengan proxy `/api` ke backend; CORS tidak dibuka.

`sslmode=disable` digunakan untuk development lokal. Deployment harus memakai
konfigurasi TLS sesuai penyedia database. Jangan commit `.env` (sudah diabaikan
oleh `.gitignore`). Tekan Ctrl+C untuk graceful shutdown; koneksi database ditutup
setelah server menyelesaikan request aktif, dengan batas waktu 10 detik.

## Backend melalui Docker

Profil pasien mandiri memerlukan migrasi 11. Endpoint, validasi, dan perilaku data
lama dijelaskan dalam [implementasi profil pasien](internal/patientprofile).

Isi `DATABASE_URL_DOCKER` pada `backend/.env` dengan koneksi PostgreSQL host,
misalnya host `host.docker.internal` dan port `5434`, memakai kredensial lokal
yang sama. Compose memerlukan variabel ini dan tidak membuat container database.
Dari folder `backend`:

```powershell
docker compose build
docker compose run --rm api migrate up
docker compose up --build -d --wait
docker compose logs -f api
docker compose down
```

Entrypoint menjalankan API sebagai pengguna non-root tanpa migrasi/seed otomatis.
Seed opsional dijalankan eksplisit: `docker compose run --rm api seed` atau
`docker compose run --rm api seed-workflow`, hanya development/test.
Port host `127.0.0.1:8080`
meneruskan port container 8080. `.dockerignore` mengecualikan `.env` dan arsip
`bin/` dari image. Variabel sesi/audit tetap dibaca melalui environment saat run.
Compose memeriksa `/api/v1/readyz` di dalam container; `--wait` menunggu API
sehat sebelum perintah selesai. Database PostgreSQL yang sudah ada harus aktif.

Untuk Neon, Docker Hub, dan penghubung Vercel, lihat
[konfigurasi backend dan autentikasi](#backend-melalui-docker).

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

| Metode | Endpoint                | Hasil                                                             |
| ------ | ----------------------- | ----------------------------------------------------------------- |
| GET    | `/api/v1/auth/csrf`     | Cookie CSRF + JSON `{ "csrf_token": "..." }`                      |
| POST   | `/api/v1/auth/login`    | JSON `{ "email", "password" }`; cookie sesi + `{ "user": {...} }` |
| POST   | `/api/v1/auth/register` | Daftar pasien; HTTP 201, cookie sesi + `{ "user": {...} }`        |
| POST   | `/api/v1/auth/logout`   | Cabut sesi di DB dan hapus cookie; HTTP 204                       |
| GET    | `/api/v1/me`            | Identitas/peran/organisasi dari DB; HTTP 401 tanpa sesi aktif     |

Semua POST/PATCH wajib membawa cookie CSRF dan header `X-CSRF-Token` dari endpoint
CSRF. Login memerlukan `Content-Type: application/json`. Frontend sudah menyediakan
proxy Vite dan integrasi login; klien API dapat menguji
endpoint langsung dengan cookie jar. Respons autentikasi memakai `Cache-Control: no-store`.
Error berbentuk `{ "error": { "code", "message", "request_id" } }`.

Password memakai Argon2id (64 MiB, 3 iterasi, paralelisme 2, salt acak 16 byte).
Token sesi 32 byte acak disimpan sebagai SHA-256; token mentah hanya ada di cookie
`HttpOnly`, `SameSite=Lax`, path `/`. Masa berlaku absolut 8 jam. Logout mencabut
sesi, dan akun nonaktif langsung ditolak pada pembacaan identitas berikutnya.

Login dibatasi per alamat koneksi TCP: burst 5, pulih satu percobaan setiap 12
detik. Limiter berada di memori dan tidak mempercayai `X-Forwarded-For`. Di balik
proxy, klien bisa berbagi bucket; kebijakan trusted proxy/limiter terdistribusi
perlu ditentukan sebelum deployment. Pembersihan sesi kedaluwarsa berjalan setiap
jam dengan timeout 30 detik dan berhenti saat shutdown; otorisasi tetap memeriksa
waktu kedaluwarsa pada setiap request.

Registrasi pasien menerima `email`, `password` (12–1024 byte), `full_name`,
`birth_year` (1900 hingga tahun berjalan), dan `sex`
(`female|male|other|unknown`). Akun, profil pasien, dan sesi dibuat dalam satu
transaksi. Referensi `HYG-XXXX-XXXX` dibuat server. CSRF dan bucket pembatasan
login juga berlaku untuk registrasi. Formulir web tersedia pada `/register`
dan dapat dibuka dari halaman masuk pada mode lokal. Untuk Clerk, lihat bagian
autentikasi Clerk di bawah: verifikasi email, reset password, dan pengaturan MFA
ditangani oleh Clerk. Audit login berantai hash belum tersedia. Akun staf dikelola melalui API admin. Login,
dashboard per peran, dan workflow pasien/dokter tersedia pada frontend.

### Autentikasi Clerk

Produksi wajib `AUTH_PROVIDER=clerk`; mode `local` hanya development/test.
Tetapkan `CLERK_PUBLISHABLE_KEY`, `CLERK_SECRET_KEY`, dan
`CLERK_AUTHORIZED_PARTIES` (daftar exact origin browser). CLI Clerk dapat menulis
nama `VITE_CLERK_PUBLISHABLE_KEY` pada backend; nama itu juga diterima sebagai
fallback public key. Rahasia Clerk tidak boleh masuk frontend.

Middleware SDK resmi memverifikasi JWT, issuer instance, expiry, dan azp.
Sesi aktif serta email utama terverifikasi diperiksa langsung melalui API Clerk.
Context yang sama dipakai domain untuk revalidasi SSE dan lock-wait; cache hanya
berumur paling lama satu detik dalam satu request. Jika Clerk tidak tersedia,
akses ditolak. Batas rate API Clerk/latensi harus diuji pada staging sebelum
menambah pengguna; tidak ada klaim uji beban untuk mekanisme baru ini.

Token sesi internal merupakan HMAC server yang tidak pernah diberikan ke
browser. Cookie sesi lokal tidak dapat melewati middleware Clerk. Cookie JWT
`__session` dipakai untuk browser/SSE dan Bearer untuk pemeriksaan SSR.
CSRF tetap diperlukan pada mutasi. `GET /auth/config` hanya memuat provider dan
publishable key; `GET /auth/status` mengembalikan pengguna/`requires_mfa` untuk
onboarding. Admin/staf wajib mempunyai MFA aktif dan token yang menunjukkan
faktor kedua telah dilewati sebelum domain klinis/admin dapat dibuka.

Pengguna baru menjadi pasien. Metadata role/org dari Clerk tidak dipercaya.
Email yang sama dengan akun lama menghasilkan `identity_link_required`, bukan
auto-link. Setelah migrasi 13, operator dapat menautkan ID secara eksplisit:

```powershell
go run ./cmd/link-clerk-user --local-user-id LOCAL_UUID --clerk-user-id user_EXAMPLE
```

Email utama Clerk harus terverifikasi dan sama dengan akun lama. Penautan
mempertahankan ID/data lokal dan mencabut sesi lokal lama. Untuk administrator
pertama, buat/verifikasi akun di Clerk, lalu operator yang berwenang menjalankan:

```powershell
go run ./cmd/link-clerk-user --clerk-user-id user_EXAMPLE --bootstrap-admin
```

Perintah menolak jika sudah ada admin yang tertaut. Akun yang terlanjur masuk
sebagai pasien hanya dapat dialihkan bila belum memiliki episode/riwayat audit;
profil pasien kosong dihapus tanpa menghapus data klinis. Jangan gunakan flag bootstrap
untuk akun yang belum disetujui sebagai administrator. Setelah itu, admin dapat
memprovisikan akun staf lewat API admin dengan `clerk_user_id`, email terverifikasi,
nama, role, dan organisasi; tanpa password lokal. Akun Clerk harus sudah ada,
dan akun Hygeia untuk email tersebut belum ada. Konflik akun existing perlu
ditangani operator; tidak ada perubahan role pasien secara diam-diam. Admin
juga dapat memakai **Tetapkan peran staf** pada akun Clerk pasien baru yang masih
kosong. Backend menolak konversi akun dengan episode atau audit pasien, mencatat
perubahan pada aktivitas admin, dan mencabut sesi lama. Akun staf dengan riwayat
pasien harus memakai akun/email staf terpisah.

Mode lokal tetap memakai payload password lama untuk fixture sintetis. Tes
`./scripts/Test-Backend.ps1 -UseLocalDatabase` memasukkan pengujian JWT signed,
MFA, email collision, pencabutan sesi, dan kegagalan provider. Pengujian unit
tanpa database tidak menjalankan tes integrasi.

## API care episode dan audit

| Metode | Endpoint                         | Fungsi                                                       |
| ------ | -------------------------------- | ------------------------------------------------------------ |
| POST   | `/api/v1/episodes`               | Dokter mengundang dengan `{"patient_ref":"HYG-DEMO-001"}`    |
| GET    | `/api/v1/episodes`               | Daftar milik aktor; filter status dan pagination             |
| GET    | `/api/v1/episodes/{id}`          | Detail metadata episode, hanya kedua peserta                 |
| PATCH  | `/api/v1/episodes/{id}`          | `{"action":"accept"}`, `reject`, atau `end`                  |
| GET    | `/api/v1/patient/access-summary` | Riwayat akses episode dan laboratorium pasien dalam kalimat  |
| GET    | `/api/v1/audit/verify`           | Verifikasi rantai sendiri; admin wajib `?chain=<patient_id>` |

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
[implementasi API](internal).
Batas lingkup, kebijakan akses, dan format audit:
[implementasi perawatan dan audit](internal/care).

## API laboratorium

| Metode | Endpoint                               | Fungsi                                                               |
| ------ | -------------------------------------- | -------------------------------------------------------------------- |
| GET    | `/api/v1/laboratories`                 | Direktori ID/nama laboratorium, hanya dokter                         |
| POST   | `/api/v1/lab-requests`                 | Buat tes dalam episode aktif milik dokter                            |
| GET    | `/api/v1/lab-requests`                 | Daftar sesuai aktor; filter status dan pagination                    |
| GET    | `/api/v1/lab-requests/{id}`            | Detail permintaan tanpa nilai hasil                                  |
| POST   | `/api/v1/lab-requests/{id}/approval`   | Pasien approve/reject pengiriman                                     |
| PATCH  | `/api/v1/lab-requests/{id}/status`     | Lab: accepted, lalu in_progress                                      |
| POST   | `/api/v1/lab-requests/{id}/result`     | Lab mengirim satu hasil; status completed                            |
| GET    | `/api/v1/lab-results/{id}`             | Hasil untuk pasien sendiri, dokter episode aktif, atau consent aktif |
| POST   | `/api/v1/lab-results/{id}/corrections` | Lab tujuan menambah koreksi hasil yang sudah selesai                 |

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
Nilai hasil tidak disalin ke audit atau log. Koreksi menerima `reason` (5–500
karakter), `value`, `unit`, `reference_range`, `flag`, dan `note` dengan batas
yang sama seperti hasil awal. Hasil awal tetap tersimpan; pembacaan mengembalikan
array `corrections` kronologis. UPDATE/DELETE/TRUNCATE koreksi diblokir trigger.
Audit `lab_result.correct` ditulis dalam transaksi yang sama. Pada akses darurat,
catatan dan alasan koreksi disembunyikan. Pengiriman koreksi tersedia melalui
API maupun menu Koreksi hasil pada ruang laboratorium; tanda terima menyediakan
tautan yang mengisi ID hasil otomatis. Detail pasien/dokter menampilkan
riwayat koreksi dan tombol Cetak / PDF melalui dialog cetak browser.
Pembatalan permintaan aktif tersedia melalui `POST /api/v1/lab-requests/{id}/cancel`
dan `POST /api/v1/prescriptions/{id}/cancel` dengan JSON `{"reason":"..."}`
(5–500 karakter setelah trim). Pasien pemilik, dokter episode aktif, dan penyedia
yang sudah ditugaskan dapat membatalkan; admin atau organisasi lain tidak dapat.
Status selesai/dibatalkan tidak dapat dibuka kembali. Waktu persetujuan dan
organisasi sebelumnya tetap tersimpan, berikut `cancellation_reason` dan
`cancellation_note` untuk pasien/dokter. Pembatalan menutup akses pemrosesan
penyedia dan menghasilkan audit/timeline/notifikasi dalam satu transaksi.
Permintaan/resep pengganti dibuat baru oleh dokter dan memerlukan persetujuan
baru dari pasien; apotek dipilih kembali pada resep baru.

`GET /api/v1/lab-receipts?limit=20&offset=0` mengembalikan `receipts` untuk
organisasi lab pemanggil: ID hasil/permintaan, status, waktu penyerahan, kode
pasien, kode/nama tes. Nilai klinis, catatan, dan riwayat koreksi tidak disertakan;
GET hasil tetap ditolak untuk lab. Daftar ini memulihkan ID koreksi setelah
reload atau login ulang tanpa membuka akses baca hasil.

Koreksi `lab_result.correct` masuk timeline dan notifikasi pasien/dokter episode
aktif; notifikasi tidak membawa nilai atau alasan koreksi. Waktu koreksi memakai
`clock_timestamp()` setelah resource terkunci agar urutan sesuai penulisan.
Notifikasi perubahan status tersedia melalui inbox dan SSE.

Migrasi `000012_workflow_cancellation` diperlukan sebelum menjalankan API baru:

Urutan migrasi pada branch terintegrasi adalah 11 (profil pasien), 12
(pembatalan layanan), dan 13 (identitas Clerk). Database dari branch pengembangan
lama yang sudah memakai versi 11 untuk pembatalan dan versi 12 untuk Clerk
memiliki riwayat berbeda. Jangan menjalankan migrasi lanjutan atau memaksa nomor
versinya pada database tersebut. Untuk demo sintetis, gunakan database baru dan
jalankan seluruh migrasi; pemindahan data lama memerlukan rekonsiliasi skema
tersendiri.
`go run ./cmd/migrate up`. Downgrade menolak penghapusan riwayat pembatalan baru;
gunakan backup kompatibel jika benar-benar harus kembali ke skema lama.
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
[implementasi API](internal).

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
[implementasi resep](internal/prescription).

## Timeline perjalanan pasien

`GET /api/v1/patient/timeline` menyajikan kejadian bisnis sukses pada rantai
pasien: episode, laboratorium, resep, dan consent. Akses hanya untuk pasien
pemilik berdasarkan sesi. Filter `kind`, `limit` 1–100, dan cursor eksklusif
`before_seq` menggantikan offset untuk endpoint ini.

Timeline tidak memuat pembacaan data atau akses ditolak; aktivitas itu tetap
tersedia pada riwayat akses. Isi hasil lab dan obat tidak disertakan. Proyeksi
menggunakan audit yang sudah ada tanpa tabel/migrasi baru dan tanpa perubahan
format hash. Kontrak lengkap:
[implementasi timeline](internal/timeline).
Tes khusus: `go test ./internal/care -run Timeline -v` dengan `TEST_DATABASE_URL`.

## Validasi dan build

Dari root repository, jalankan seluruh tes backend termasuk integrasi database
dan `go vet` dengan satu perintah:

```powershell
./scripts/Test-Backend.ps1 -UseLocalDatabase
```

Opsi ini membaca koneksi dari `backend/.env` hanya jika `APP_ENV` bernilai
development/test. Tes memakai schema sementara dan tidak mengubah tabel demo.
Untuk database tes terpisah, atur `TEST_DATABASE_URL` lalu jalankan script tanpa
opsi. Script menolak berjalan tanpa koneksi tes agar integrasi tidak dilewati
diam-diam, dan memulihkan environment sebelumnya setelah selesai.

Workflow `.github/workflows/verify.yml` menyiapkan PostgreSQL sementara untuk
tes integrasi Go, menjalankan `go vet`/build, serta membangun image backend.
Job frontend menjalankan pemeriksaan Svelte, lint, build, unit test, dan E2E
fixture dengan Chromium. Workflow baru berjalan di GitHub setelah file di-push.

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

Tes resep memakai fixture care dengan seluruh migrasi 1–9. Jalankan
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

Frontend SvelteKit berada di `../frontend`, pada port default `5173`, dengan
proxy `/api`, halaman `/login`, dashboard, helper SSR, dan workflow pasien/dokter.
Antarmuka lab/apotek/admin sudah terintegrasi sesuai
[implementasi API](internal) dan
[alur penggunaan](../README.md#panduan-demonstrasi).

Lihat [panduan validasi dan build](#validasi-dan-build) untuk menjalankan
pemeriksaan backend. Hasil pemeriksaan CI tersedia pada GitHub Actions.

Referensi: [pgx](https://github.com/jackc/pgx),
[PostgreSQL Docker image](https://hub.docker.com/_/postgres).
