# Hygeia tahap 1 persiapan pilot

Implementasi lokal 28 Oktober 2026 di atas revisi `7c989dc`, branch
`feature/frontend-provider-workspaces`. Perubahan belum di-commit atau di-push.
Tidak ada deployment publik atau data pasien nyata. Tahap akun produksi,
operasional hosting, dan pilot mitra belum termasuk paket ini.

## Hasil implementasi

| Bagian                     | Perubahan                                                                                                                                                                                                                                                                                                            |
| -------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Kuota login dan registrasi | `TRUSTED_PROXY_CIDRS` opsional; default abaikan forwarding. Hanya socket proxy tepercaya yang boleh memasok XFF. Seluruh rantai divalidasi dan alamat pertama yang tidak tepercaya dipilih dari kanan. IPv4-mapped IPv6 dinormalisasi. CIDR rusak menghentikan startup sebelum database.                             |
| Kuota yang dipertahankan   | Login/registrasi berbagi burst 5 dan isi ulang satu token per 12 detik, idle cleanup 10 menit, batas 10.000 identitas. Header rusak/kosong/seluruhnya tepercaya kembali ke IP socket.                                                                                                                                |
| Organisasi admin           | Pemilih 20 hasil per halaman memakai endpoint lama. Tetap dapat maju meski halaman berisi tipe lain; hasil ke-105 dapat dipilih. Pilihan dipertahankan saat pindah halaman, dihapus saat ganti peran, dan kegagalan memiliki tombol coba lagi. Cache 100 nama hanya untuk label tampilan, bukan sumber pilihan akun. |
| Koreksi lab                | Alasan setelah `TrimSpace` dihitung dengan Unicode code points, bukan byte; 5–500 diterima. Kebijakan panjang password tidak diubah.                                                                                                                                                                                 |
| Dependensi                 | Vitest dan browser provider dipatok 4.1.12; paket terkait dalam lockfile selaras. Dua advisory moderate tidak lagi dilaporkan oleh audit.                                                                                                                                                                            |
| Tampilan                   | Tiga selector stylesheet biasa menggunakan `svg`, tanpa `:global`. Petunjuk memulai perawatan, kode pasien, tombol salin dan hasil salin tampil sebelum ringkasan metrik.                                                                                                                                            |
| Demo                       | Perintah `go run ./cmd/seed-workflow` terpisah, hanya development/test, transaksi atomik, namespace terpisah, aman diulang tanpa mereset password/progres. Menyediakan lima akun, perawatan, hasil dan antrean lab, resep siap/antrean, timeline, notifikasi, audit HMAC.                                            |

Tidak ada endpoint, format respons API, migrasi, atau variabel konfigurasi
aplikasi baru selain `TRUSTED_PROXY_CIDRS`. pnpm dan arah hosting Vercel tetap.

## Bukti pemeriksaan lokal

| Pemeriksaan                                                | Hasil                                                                                                                                                                                                          |
| ---------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `pnpm check`                                               | 0 error dan 0 warning                                                                                                                                                                                          |
| `pnpm lint`                                                | Lulus                                                                                                                                                                                                          |
| `pnpm build`                                               | Lulus; tiga warning selector CSS hilang. Adapter-auto masih belum mendeteksi lingkungan produksi pada workstation lokal.                                                                                       |
| `pnpm test`                                                | 2 file, 2 tes lulus dengan Chromium, Vitest 4.1.12                                                                                                                                                             |
| `pnpm test:e2e`                                            | Enam suite lulus berurutan; suite tambahan memeriksa organisasi ke-105, halaman tanpa tipe sesuai, retensi pilihan, reset peran, dan retry.                                                                    |
| `./scripts/Test-Backend.ps1 -UseLocalDatabase`             | Integrasi PostgreSQL dan `go vet` lulus. Tambahan mencakup proxy/forgery/default policy, startup CIDR rusak, Unicode 4/5/500/501 CJK dan emoji, seed berulang, konflik namespace/rollback, penolakan produksi. |
| Seed dua kali                                              | Pengisian pertama berhasil; kedua melaporkan data sudah ada dan mempertahankan password/progres. Produksi ditolak sebelum database.                                                                            |
| `docker compose up -d --build --wait`                      | API dibangun ulang dari working tree dan sehat                                                                                                                                                                 |
| `HYGEIA_LIVE_TEST=1 pnpm test:demo`                        | Kelima peran membuka data dari backend lokal pada 1440 dan 390 px; tidak ada overflow atau error browser. Kode pasien tersalin, timeline tersedia, audit utuh.                                                 |
| `HYGEIA_LIVE_TEST=1 node tests/clinical-workflows.live.js` | Alur perawatan, lab, resep/apotek, consent, CarePass/revoke, darurat, notifikasi, timeline lulus. Rantai contoh `f86eab68-86c8-48bc-a9d6-6b05fe5f6520`, 56 event, `intact=true`, suffix sintetis `ad649dcd`.   |
| `pnpm audit --json`                                        | 0 moderate/high/critical, 1 low `deepmerge`                                                                                                                                                                    |

Server frontend dan suite browser dijalankan bergiliran. Setelah penyesuaian
spacing terakhir, pemeriksaan frontend, unit, suite dashboard yang terdampak,
dan demo lima peran diulang. Tes live pertama yang langsung mengikuti lima login
demo sempat ditolak 429; setelah kuota pulih tes diulang dan lulus, tanpa
melonggarkan limiter. Sisa fixture dari percobaan itu dipertahankan, sesuai
perilaku tes live yang tidak membersihkan data audit.

Screenshot terbaru ada di `frontend/.test-results/pilot-stage1/` (diabaikan Git).
Ini pemeriksaan Chromium dengan viewport desktop/mobile, bukan perangkat fisik,
riset pengguna sasaran, audit WCAG lengkap, atau pembuktian hosting produksi.
Tes live klinis menggunakan browser untuk pasien/dokter dan API untuk beberapa
tindakan penyedia; suite fixture dan demo lima peran melengkapi bukti tampilan.

## Batas proxy yang harus diperhatikan

Proxy produksi harus menimpa header pengunjung atau menyaringnya sebelum
menambahkan alamat koneksi yang diamati. Konfigurasi CIDR saja tidak cukup.
Kebijakan yang disetujui meminta fallback socket untuk header rusak: pada proxy
append-only yang membiarkan prefix rusak, klien dapat memakai kuota identitasnya
dan kuota fallback bersama. Tinjauan independen mengonfirmasi konsekuensi ini.
Karena itu pemisahan kuota penuh bergantung pada sanitasi XFF proxy; tidak
diklaim menutup semua denial of service pada topologi yang belum diverifikasi.
Vite development tidak otomatis memasok XFF, dan daftar proxy kosong tetap
menggunakan socket. Detail konfigurasi ada di [backend](../backend/README.md).

## Triase deepmerge

`pnpm why deepmerge` menunjukkan `deepmerge@4.3.1` melalui
`@sveltejs/vite-plugin-svelte@7.3.1`, dependensi pengembangan. Sumber terpasang
`src/utils/options.js:30,159–190,225` memanggil merge untuk default options,
konfigurasi Svelte lokal, inline plugin options, dan resolved build options.
Tidak ditemukan pemanggilan dari request pengguna atau impor langsung dalam
aplikasi. Pencarian nama pada output client/server tidak menemukan kecocokan;
ini bukti pendukung, bukan pembuktian universal ketiadaan kode terbundel.

Advisory [GHSA-gcqx-w48m-w2h4](https://github.com/advisories/GHSA-gcqx-w48m-w2h4)
menyebut prototype poisoning pada object hasil merge. Halaman advisory belum
mencantumkan patch. Feed audit menyarankan `>=4.3.2`, tetapi registry saat
pemeriksaan masih menyediakan 4.3.1; `pnpm view deepmerge@4.3.2 version` tidak
menemukan versi. Temuan tetap dilaporkan dan tidak disembunyikan. Jalur masukan
yang ditelusuri adalah konfigurasi build dari repository, sehingga data API
pasien belum terbukti mencapai fungsi ini. Lingkungan build tetap harus memakai
konfigurasi/dependensi tepercaya dan tidak menjalankan dev server publik.

Advisory Vitest
[GHSA-qxg6-89mg-rqv8](https://github.com/advisories/GHSA-qxg6-89mg-rqv8) dan
[GHSA-xp26-3454-mjgp](https://github.com/advisories/GHSA-xp26-3454-mjgp)
ditangani dengan paket 4.1.12, bukan pengecualian audit. Peringatan integrasi
`transformIndexHtml` pada Vitest/SvelteKit masih tampil meski tes lulus; ini
berbeda dari warning selector CSS yang diperbaiki.

## Yang perlu Anda lakukan

1. Jalankan API dan frontend lokal. Gunakan akun `pilot.patient@demo.hygeia.test`,
   `pilot.doctor@demo.hygeia.test`, `pilot.lab@demo.hygeia.test`,
   `pilot.pharmacy@demo.hygeia.test`, dan `pilot.admin@demo.hygeia.test`.
   Password berasal dari `DEMO_PASSWORD` lokal saat seed pertama, tidak disalin
   ke dokumen ini. Kode pasien `HYG-PILOT-SYNTH-001`.
2. Coba pemeriksaan dan resep yang berstatus terkirim, lalu tinjau perubahan
   pada pasien. Seed ulang mempertahankan kemajuan, bukan mengembalikan demo.
3. Cari calon mitra dan catat alur, peran pengguna, pengelola akun, serta
   penanggung jawab klinis dan gangguan. Belum gunakan data nyata.
4. Ikuti urutan prioritas lanjutan di bawah untuk pengamanan akun, pengujian
   bersama pengguna, staging, dan kesiapan operasional. Uji beban tetap diperlukan
   sebelum penggunaan data nyata.

## Prioritas lanjutan menuju pilot

Daftar ini adalah rencana pekerjaan berikutnya, bukan fitur yang sudah tersedia
atau bukti bahwa layanan siap digunakan dengan data pasien nyata. Penyelesaiannya
perlu dicatat beserta hasil verifikasi masing-masing.

| Urutan | Yang perlu dikerjakan | Hasil yang dituju |
| --- | --- | --- |
| **1. Pemulihan akun** | Lupa password, tautan reset sekali pakai, masa berlaku tautan | Pengguna bisa memulihkan akun sendiri |
| **2. Verifikasi email** | Konfirmasi email dan pengiriman ulang dengan pembatasan | Kepemilikan email dapat diverifikasi |
| **3. Pengelolaan sesi** | Daftar sesi aktif, keluar dari perangkat lain, pencabutan sesi setelah reset password | Pengguna bisa mengendalikan akses akunnya |
| **4. Pengamanan staf** | MFA untuk admin/staf dan prosedur pemberian akses | Akses klinis lebih terlindungi |
| **5. Uji pengalaman pengguna** | Coba alur pasien → dokter → lab → apotek bersama calon pengguna | Hambatan nyata ditemukan dan diperbaiki |
| **6. Staging online** | Frontend Vercel, backend, domain, HTTPS, proxy dan SSE | Aplikasi bisa diuji bersama melalui internet |
| **7. Backup dan pemulihan** | Backup database beserta kunci audit, lalu simulasi restore | Data dapat dipulihkan saat terjadi gangguan |
| **8. Monitoring** | Pemantauan ketersediaan, error, dan pemberitahuan gangguan | Masalah layanan cepat diketahui |
| **9. Persiapan pilot** | Mitra, penanggung jawab, aturan data, panduan staf dan penanganan masalah | Pengujian terbatas dapat dilaksanakan |

Proposal Word diperbarui hanya pada klaim yang terverifikasi. Identitas tim,
institusi, URL demo, dan hasil riset pengguna tetap menunggu data sebenarnya.
