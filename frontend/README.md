# Hygeia Frontend

[README proyek](../README.md) · [Dokumentasi](../docs/README.md) · [Kontribusi](../CONTRIBUTING.md)

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
[handoff pengembang](../docs/plans/2026-10-15-patient-doctor-and-provider-handoff.md).

## Menjalankan

Jalankan database/API sesuai [panduan backend](../backend/README.md), lalu:

```powershell
pnpm install
pnpm dev
```

Buka port yang dilaporkan Vite. Akun demo tersedia dari seed backend; password
berasal dari `backend/.env`. Jangan commit file `.env` atau data pasien.
Production membutuhkan adapter deployment dan reverse proxy HTTPS satu origin.

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
`pnpm test:e2e` menjalankan keenam suite fixture secara berurutan. `pnpm test`
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

Detail hasil lab menampilkan hasil awal dan riwayat koreksi dari API. Tombol
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
Lihat [kontrak profil pasien](../docs/15-PATIENT-PROFILE.md).

Tes profil dengan API/PostgreSQL nyata (database sementara saja, bukan produksi):

```powershell
$env:HYGEIA_LIVE_TEST='1'
node tests/patient-profile.live.mjs
```

Jalankan migrasi 11 dan API di `127.0.0.1:8080` terlebih dahulu. Tes membuat akun
pasien sintetis baru, menyimpan profil, memuat ulang, dan memeriksa logout.

Petugas lab membuka menu **Koreksi hasil** atau tautan **Koreksi hasil ini** pada
tanda terima. Masukkan ID hasil (bukan ID permintaan), alasan, serta seluruh
nilai pengganti. Formulir tidak mengambil hasil lama melalui GET; hak baca lab
tetap mengikuti kebijakan backend. Setelah berhasil, tampil ID koreksi dan
isian klinis dibersihkan. Kesalahan mempertahankan isian untuk diperbaiki.
Pada hasil awal dan koreksi, penanda wajib dipilih tanpa default Normal.
Tombol **Tinjau** menampilkan snapshot seluruh isian dan referensi sebelum ada
POST. Petugas dapat kembali mengedit; hanya konfirmasi akhir mengirim data.
