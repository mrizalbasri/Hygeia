# Pemeriksaan kesiapan Hygeia - 28 Oktober 2026

**Snapshot pemeriksaan awal.** Implementasi tahap 1 dan hasil pengujian ulang
terbaru tersedia di [laporan tahap 1](PILOT-STAGE1-2026-10-28.md). Temuan dan
prioritas di bawah mempertahankan keadaan sebelum perbaikan, bukan status
terkini working tree.

Revisi: `7c989dcd8d7372e21a225b6d6a673d77836e8adb`. Working tree bersih saat mulai.
Pemeriksaan ini tidak mengubah kode aplikasi, commit, push, atau deployment publik.

## Kesimpulan

Alur inti siap dicoba sebagai demo lokal dengan data sintetis. Kesiapan demo
online harus diuji di hosting tujuan. Belum layak dinyatakan siap untuk data
pasien nyata atau bebas kerentanan. UI yang diperiksa rapi dan konsisten, tetapi
belum ada bukti usability dari pengguna sasaran atau audit WCAG lengkap.

## Pemeriksaan yang dijalankan

| Pemeriksaan                                                | Hasil aktual                                                                                                                                              |
| ---------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `pnpm check`                                               | Lulus, 0 error dan 0 warning                                                                                                                              |
| `pnpm lint`                                                | Lulus                                                                                                                                                     |
| `pnpm build`                                               | Lulus; warning selector CSS `:global(svg)` dan adapter-auto belum menentukan target produksi                                                              |
| `pnpm test`                                                | 2 file / 2 tes lulus dengan Chromium bawaan; percobaan Edge gagal saat meluncurkan browser sebelum tes berjalan                                           |
| `pnpm test:e2e`                                            | Kelima suite lulus: landing/dashboard, clinical, provider, registration, role-overviews                                                                   |
| `./scripts/Test-Backend.ps1 -UseLocalDatabase`             | Lulus, integrasi PostgreSQL aktif, diikuti `go vet`                                                                                                       |
| `docker compose up -d --build --wait` dari backend         | Build kode terbaru dan startup sehat                                                                                                                      |
| HTTP health / readyz                                       | API dan koneksi DB sehat; endpoint readiness tidak memverifikasi versi migrasi                                                                            |
| `HYGEIA_LIVE_TEST=1 node tests/clinical-workflows.live.js` | Lulus setelah rebuild backend; undangan, persetujuan, lab, resep/apotek, consent/revoke, CarePass/revoke, darurat, notifikasi, timeline dan responsivitas |
| Audit pada uji live terbaru                                | 56 event, `intact=true`; fixture sintetis `7a380991` dipertahankan oleh desain tes                                                                        |
| GitHub PR #20                                              | MERGED; head sama dengan revisi di atas; backend dan frontend Verify SUCCESS                                                                              |
| `pnpm audit --json`                                        | 2 moderate `@vitest/mocker`, 1 low `deepmerge`; jalur test/build, bukan bukti eksploit pada aplikasi publik                                               |

Tes unit hanya contoh kecil; cakupan perilaku terutama berasal dari integrasi
backend dan E2E. Tes live pasien/dokter menggunakan browser, sedangkan beberapa
tindakan lab/apotek/admin melalui API. Suite fixture provider menguji UI secara
terpisah; ini belum membuktikan seluruh kombinasi UI terhadap deployment publik.

Container API lama ternyata dibangun 25 Oktober, sehingga uji live diulang setelah
rebuild dari kode saat ini. Docker/PostgreSQL dan satu Vite lokal dinyalakan untuk
pemeriksaan. Tidak ada kunci atau password yang disalin ke laporan.

## Audit UI/UX dengan browser

Tujuan: pengguna memahami produk, masuk, melihat ringkasan pasien, dan menemukan
navigasi pada desktop serta ponsel. Bukti berasal dari tangkapan sesi ini.

1. **Landing - baik.** Hierarki judul jelas, aksi masuk mudah ditemukan, warna
   biru konsisten, dan status prototipe/data sintetis terlihat.
2. **Login - baik untuk demo.** Label email/password dan tautan registrasi jelas.
   Pengunjung tetap perlu memperoleh akun demo dari pengelola; petunjuk akses
   juri perlu disertakan saat pengumpulan.
3. **Ringkasan pasien desktop - baik dengan catatan.** Navigasi terkelompok,
   status aktif jelas, dan kondisi kosong memberikan penjelasan. Akun Aulia
   belum berisi perawatan sehingga manfaat produk kurang terlihat saat pertama
   masuk. Instruksi awal meminta kode pasien, sementara kartu kodenya berada
   lebih bawah; tampilkan kode dan aksi salin dekat instruksi awal.
4. **Ringkasan pasien mobile 375 x 812 - baik dengan catatan.** Konten tersusun
   satu kolom dan metrik dua kolom; tidak terlihat teks tumpang tindih pada
   tangkapan yang diterima. Banyak kartu kosong membuat informasi awal panjang.
5. **Menu mobile - baik.** State berubah dari Menu menjadi Tutup menu;
   tautan Perawatan berhasil membuka `?view=episodes` dan menu menutup kembali.
   Halaman tujuan menampilkan kode pasien, filter status, dan kondisi kosong.

Tangkapan yang sudah diperiksa tersimpan di
`frontend/.test-results/readiness-2026-10-28/` (diabaikan Git):

- `01-landing.png`
- `02-login.png`
- `03-patient-overview.png`
- `04-patient-desktop.png`
- `05-patient-mobile.png`
- `06-mobile-menu.png`

Percobaan awal capture menu ditolak karena error development. File
`06-mobile-menu.png` sudah diganti dengan capture menu yang benar pada server
tunggal dan diperiksa kembali.

![Langkah 1 - landing](../frontend/.test-results/readiness-2026-10-28/01-landing.png)
![Langkah 2 - login](../frontend/.test-results/readiness-2026-10-28/02-login.png)
![Langkah 3 - dashboard desktop](../frontend/.test-results/readiness-2026-10-28/04-patient-desktop.png)
![Langkah 4 - dashboard mobile](../frontend/.test-results/readiness-2026-10-28/05-patient-mobile.png)
![Langkah 5 - menu mobile](../frontend/.test-results/readiness-2026-10-28/06-mobile-menu.png)

Ada error SSR 500 saat beberapa server Vite/test menggunakan generated
`.svelte-kit` checkout yang sama. Setelah tes selesai dan server tunggal
dimuat ulang, dashboard tampil. Akar penyebab belum dipastikan; jangan mengklaim
bug aplikasi menetap atau sudah diperbaiki. Jalankan demo dan suite browser
secara berurutan, atau pada checkout/output terpisah, lalu verifikasi ulang.

Aksesibilitas: label, heading, skip link dan state menu tersedia pada snapshot;
suite yang lulus mencakup FAQ keyboard dan beberapa lebar layar. Kontras numerik,
pembaca layar, seluruh urutan fokus dan WCAG lengkap belum diverifikasi.

## Keamanan

Scan sumber dengan baseline independen, tinjauan arsitektur, dan pemeriksaan
frontend menemukan satu masalah sedang: limiter login/register memakai IP
socket `RemoteAddr`. Pada reverse proxy biasa, semua pengguna dari satu proxy
berbagi kuota lima percobaan dan satu token tiap 12 detik. Penyerang tanpa akun
dapat menghabiskan kuota sehingga pengguna lain terkena 429. Lokasi:
`backend/internal/auth/handler.go:77`. Tidak ada serangan DoS yang dijalankan.
Perbaikan perlu trusted-proxy policy dan identitas klien terverifikasi; jangan
langsung mempercayai `X-Forwarded-For` dari semua sumber.

Pada jalur yang diperiksa, tidak ditemukan bypass hak akses atau kebocoran
klinis terkonfirmasi. Kontrol yang ditemukan meliputi Argon2id, cookie HttpOnly,
CSRF, sesi tersimpan sebagai hash, pembatasan organisasi/resource, grant berakhir
dan dapat dicabut, serta audit transaksional. Secure cookie bergantung pada
`APP_ENV=production`. Hasil negatif ini bukan jaminan keamanan menyeluruh.

Cakupan sumber scan tetap **partial**: beberapa sumber tes integrasi besar,
komponen presentasi, CSS, harness browser, dan dokumentasi pendukung tidak
diaudit penuh. Go vulnerability database scan belum dijalankan (`govulncheck`
tidak terpasang). Advisory npm perlu ditriase sebelum upgrade terarah; tidak ada
upgrade otomatis pada pemeriksaan ini.

Referensi advisory:

- https://github.com/advisories/GHSA-qxg6-89mg-rqv8
- https://github.com/advisories/GHSA-xp26-3454-mjgp
- https://github.com/advisories/GHSA-gcqx-w48m-w2h4

Laporan scan terkelola:
`C:\Users\RizalBasri\.codex\state\plugins\codex-security\scans\Hygeia\7c989dcd8d7372e21a225b6d6a673d77836e8adb_20261028T085343Z_ue8m03hc\report.md`.
Artefak pendamping: `scan-manifest.json`, `findings.json`, `coverage.json`, dan
`threatmodel.md` pada direktori yang sama. Usage yang dikembalikan plugin:
11.647.549 total token terhitung, termasuk 11.176.064 cached input; angka ini
metrik plugin lintas lima thread dan bukan ukuran cakupan audit.

## Prioritas berikutnya

1. Perbaiki limiter login agar sesuai topology proxy dan tambahkan tes batas klien.
2. Triase/upgrade terarah dependensi test/build yang terkena advisory, lalu ulangi
   tes yang relevan.
3. Tentukan adapter SvelteKit dan proxy `/api` satu origin, HTTPS, origin CarePass,
   serta SSE tanpa buffering; uji alur di hosting tujuan.
4. Sediakan akun demo dengan satu contoh perjalanan perawatan terisi dan instruksi
   masuk untuk juri; dekatkan kode pasien ke langkah awal.
5. Perbaiki warning CSS di `frontend/src/routes/landing.css:498,669,676` dan uji
   produksi; ini tidak membuat build saat ini gagal.
6. Sebelum data nyata: validasi hak DB/runtime terpisah dari migrasi, penyimpanan
   secret, backup DB beserta kunci audit dan uji restore, monitoring, uji beban,
   tata kelola akses darurat, serta kebutuhan pemulihan akun dan privasi.
