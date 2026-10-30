# Hygeia Frontend

[README proyek](../README.md) · [Kontribusi](../CONTRIBUTING.md)

## Clerk dan keamanan akun

Konfigurasi lokal sekarang memakai aplikasi Clerk development Hygeia:
`AUTH_PROVIDER=clerk` dan `PUBLIC_CLERK_PUBLISHABLE_KEY` di `frontend/.env`.
Backend memiliki secret key sendiri; jangan salin secret ke frontend.
`src/env.ts` mendeklarasikan env memakai API SvelteKit 3.

Halaman `/login` dan `/register` memakai komponen resmi ClerkJS. Pemulihan
password dan verifikasi email mengikuti flow Clerk. `/account/security`
menampilkan UserProfile untuk email/password, MFA/recovery codes, serta sesi
aktif. Admin/staf harus mengaktifkan MFA, lalu masuk dengan faktor kedua sebelum
dashboard dapat dibuka. Tombol **Keamanan akun** tersedia pada dashboard Clerk.

Identitas Clerk tidak menentukan role klinis. Backend memberikan akun pasien
untuk pengguna baru dan mempertahankan role/organisasi/consent di database
Hygeia. Email yang sudah ada memerlukan penautan eksplisit oleh pengelola.
Administrator pertama perlu diprovisikan dengan perintah operator pada
[panduan backend](../backend/README.md#autentikasi-clerk). Akun yang sudah masuk
sebagai pasien harus masih kosong sebelum dapat dialihkan. Tidak ada akun manusia yang otomatis diberi
peran admin oleh integrasi ini.

Mode lokal tetap tersedia hanya untuk development/test sintetis:
tetapkan `AUTH_PROVIDER=local` pada frontend **dan** backend. Fixture E2E
memilih mode ini secara eksplisit. Tes live klinis lama menggunakan password
demo lokal dan tidak boleh dijalankan ke API mode Clerk.

Untuk memeriksa tampilan Clerk development tanpa mengirim email atau membuat
akun pengguna, jalankan `HYGEIA_CLERK_LIVE=1 node tests/clerk-ui.live.js` dengan
API lokal aktif. Skrip memakai port 5188, yang perlu masuk authorized parties.
Tes ini hanya membuktikan UI login/registrasi dimuat; bukan bukti login/reset/MFA
akun nyata atau instance produksi sudah lolos.

## Teknologi

| Bagian            | Stack                               |
| ----------------- | ----------------------------------- |
| Bahasa            | TypeScript 6, Svelte, HTML, dan CSS |
| Framework         | Svelte 5 dan SvelteKit 3            |
| Styling dan build | Tailwind CSS 4 dan Vite 8           |
| Pengujian         | Vitest dan Playwright               |
| Lingkungan CI     | Node.js 24 dan pnpm 11.18.0         |

## Cakupan aplikasi

SvelteKit dengan landing Hygeia, login cookie sesi, dan dashboard sesuai peran.
Backend Go di `127.0.0.1:8080`; browser memakai proxy `/api` satu origin.
SSR meneruskan cookie melalui `API_BASE_URL` (lihat `src/env.ts`).

Pasien: ringkasan, kode pasien, undangan/perawatan, persetujuan pemeriksaan,
hasil lab, resep dan pilihan apotek, izin dokter, CarePass, timeline, riwayat
akses/verifikasi audit, riwayat akses darurat, dan notifikasi langsung.
Dokter: undangan, pemeriksaan, hasil, resep multiobat, permintaan izin,
penebusan CarePass, akses darurat, dan notifikasi.
Laboratorium: antrean permintaan yang sudah disetujui pasien, transisi
sent→accepted→in_progress, dan penyerahan hasil terstruktur sekali pakai.
Apotek: antrean resep, transisi sent→accepted→preparing→ready→completed, dan
konfirmasi penyerahan. Admin: organisasi, akun, status audit HMAC, tinjauan
akses darurat, dan log aktivitas. Kontrak dan urutan kerja ada di
[alur penggunaan](../README.md#panduan-demonstrasi).

## Menjalankan

Jalankan database/API sesuai [panduan backend](../backend/README.md), lalu:

```powershell
pnpm install
pnpm dev
```

Buka port yang dilaporkan Vite. Akun demo tersedia dari seed backend; password
berasal dari `backend/.env`. Jangan commit file `.env` atau data pasien.
Jalur `/api/v1` diteruskan oleh route server SvelteKit ke `API_BASE_URL`, baik
pada development maupun Vercel. Adapter Vercel dipilih ketika `VERCEL` tersedia;
build lokal memakai adapter-auto. Lihat [konfigurasi backend dan autentikasi](../backend/README.md#backend-melalui-docker).

## Pengujian

```powershell
pnpm check
pnpm lint
pnpm build
$env:PLAYWRIGHT_CHANNEL='msedge'
pnpm test
pnpm test:e2e
```

Tes E2E fixture menjalankan server sendiri dan menutupnya setelah selesai.
`pnpm test:e2e` menjalankan sembilan suite fixture secara berurutan, termasuk
pagination organisasi setelah hasil ke-100, pemilihan sesuai tipe, reset peran,
dan pemulihan kegagalan pemuatan. `pnpm test`
sudah mengaktifkan mode run; jangan menambahkan `-- --run` lagi.
Tes mencakup landing/mobile, persetujuan/penolakan, CSRF, konflik 409, detail 403,
503/retry, cursor bigint, API 401 dan SSE session-ended, lifecycle antrean
lab/apotek, serta pengelolaan akun/organisasi dan tinjauan akses darurat admin.
Screenshot di `.test-results/` diabaikan Git. Tes memakai Chromium Playwright
secara default; set `PLAYWRIGHT_CHANNEL='msedge'` untuk memakai Edge. Deteksi
kesiapan server memakai polling HTTP karena Vite 8 tidak selalu mencetak URL
dev server pada stdout non-TTY.

Uji integrasi API demo lokal (opt-in):

```powershell
$env:HYGEIA_LIVE_TEST='1'
node tests/clinical-workflows.live.js
```

Tes membaca `DEMO_PASSWORD` dari `backend/.env`, memerlukan API aktif di :8080,
lalu menjalankan Vite di :5185. Tes membuat organisasi/akun sintetis unik dan
menyelesaikan alur dokter/pasien melalui browser. Tindakan lab/apotek dilakukan
melalui API sampai frontend petugas selesai. Data sintetis sengaja dipertahankan
untuk reproduksi audit; tidak mengubah akun lama. Jangan jalankan pada produksi.
Kesiapan server pada tes live juga diperiksa melalui polling HTTP, sehingga
tidak bergantung pada keluaran URL Vite di terminal.

Setelah menjalankan `go run ./cmd/seed-workflow` dari backend, periksa demo lima
peran dengan data tersimpan pada API lokal:

```powershell
$env:HYGEIA_LIVE_TEST='1'
pnpm test:demo
```

Tes ini membuka kelima akun `pilot.*@demo.hygeia.test` pada desktop 1440 px dan
mobile 390 px, memeriksa data API, timeline, integritas audit, dan tombol salin
kode pasien. Tes memakai port 5186, membuat sesi dan audit baca, tetapi tidak
mereset atau memajukan perawatan. Screenshot ada di `.test-results/pilot-stage1/`.
Password akun harus masih sama dengan `DEMO_PASSWORD` lokal. Jalankan server
demo, check/build, unit, dan suite browser bergiliran agar generated files
SvelteKit tidak ditulis oleh beberapa proses sekaligus.

Modul shared berada di `src/lib/workflows`: `api.ts` untuk CSRF/error/cursor,
`PagedList` untuk pagination, `DirectoryPicker` untuk direktori, `ClinicalDetail`
untuk pembacaan berotorisasi, serta `Notifications` untuk inbox.
`notification-state.svelte.ts` menyediakan satu pengelola SSE per dashboard
melalui context, termasuk badge belum dibaca, reconnect, dan retry manual.
Event menginvalidasi daftar/ringkasan agar dibaca ulang melalui API; tidak
dipakai sebagai sumber nilai klinis. Setiap mutasi memuat ulang daftar; server
tetap menentukan izin dan status.

Ringkasan pasien memakai `/patient/work-summary` untuk seluruh jumlah pekerjaan,
termasuk izin dokter. Daftar terbaru tetap dibatasi enam entri dan tidak
dipakai untuk menghitung total. Kegagalan menampilkan tanda pisah dan retry.
`dashboard-reliability.e2e.mjs` memeriksa badge, satu transport lintas menu,
reconnect/retry, dan session expiry memakai transport deterministik; suite
clinical juga memeriksa event `session-ended` melalui EventSource native.

Detail hasil lab menampilkan koreksi terbaru dan riwayat hasil dari API. Tombol
Cetak / PDF pada detail hasil/resep membuka dialog cetak browser; pilih Simpan
sebagai PDF untuk mengekspor. Tombol hanya aktif setelah data berhasil dimuat.
Mode cetak menyembunyikan navigasi dan kontrol, serta menampilkan seluruh isi
dialog tanpa batas tinggi.

Registrasi pasien tersedia di `/register`, melalui tautan pada halaman masuk.
Formulir meminta identitas, tahun lahir, jenis kelamin, dan konfirmasi password;
setelah berhasil, sesi langsung mengarah ke dashboard pasien. Kesalahan API
ditampilkan tanpa menghapus isian, dan pengiriman ganda diblokir selama proses.

Pasien dapat mengubah data diri melalui menu **Profil saya**. Email dan nomor
pasien hanya baca; tanggal lahir lengkap serta kontak bersifat opsional.
Data lama yang hanya berisi tahun lahir tidak diubah menjadi tanggal buatan.
Lihat [implementasi profil pasien](../backend/internal/patientprofile).

Tes profil dengan API/PostgreSQL nyata (database sementara saja, bukan produksi):

```powershell
$env:HYGEIA_LIVE_TEST='1'
node tests/patient-profile.live.mjs
```

Jalankan seluruh migrasi dan API di `127.0.0.1:8080` terlebih dahulu. Tes membuat akun
pasien sintetis baru, menyimpan profil, memuat ulang, dan memeriksa logout.

Petugas lab membuka menu **Tanda terima hasil** untuk menemukan kembali hasil
yang sudah diserahkan, lalu **Koreksi hasil ini** mengisi ID otomatis. Daftar ini
hanya metadata dan tetap tersedia setelah reload. Menu **Koreksi hasil** juga
menerima ID hasil (bukan ID permintaan), alasan, serta seluruh
nilai pengganti. Formulir tidak mengambil hasil lama melalui GET; hak baca lab
tetap mengikuti kebijakan backend. Setelah berhasil, tampil ID koreksi dan
isian klinis dibersihkan. Kesalahan mempertahankan isian untuk diperbaiki.
Penanda hasil wajib dipilih, lalu petugas meninjau isian sebelum konfirmasi akhir.

Pasien/dokter dapat membuka **Batalkan layanan**, sedangkan lab/apotek memakai
**Tidak dapat melayani**, lalu mengisi alasan dan mengonfirmasi. Dokter dapat
mengajukan pemeriksaan pengganti atau resep baru dengan detail sebelumnya.
Pasien menyetujui kembali lab dan memilih kembali apotek pada permintaan baru.
Tidak ada penugasan penyedia baru yang langsung aktif tanpa persetujuan.

`node tests/workflow-recovery.e2e.js` memeriksa pembatalan, penggantian,
pemulihan tanda terima, koreksi realtime, draft yang tetap utuh,
reconnect/retry/fallback, dan pembersihan stream. Suite ini masuk `pnpm test:e2e`.
