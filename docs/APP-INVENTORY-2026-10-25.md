# Inventaris aplikasi dan checklist penyelesaian Hygeia

Pemeriksaan lokal: 25 Oktober 2026. Acuan: kode working tree saat diperiksa,
BRD, PRD, completion spec, tes frontend, dan tes backend dengan PostgreSQL.
Perubahan lokal yang sudah ada dipertahankan. Hasil ini bukan jaminan bebas bug.

## Membuka aplikasi

- Landing: http://127.0.0.1:5188/
- Masuk: http://127.0.0.1:5188/login
- Registrasi pasien: http://127.0.0.1:5188/register
- CarePass: `/carepass#token=...`, melalui tautan/QR yang dibuat pasien.
- Backend: http://127.0.0.1:8080/api/v1/health
- Database readiness: http://127.0.0.1:8080/api/v1/readyz

Dashboard memerlukan login dan mengikuti peran akun. Mengubah URL tidak memberi
hak peran lain. Password demo mengikuti konfigurasi lokal saat seed pertama;
lihat `backend/.env` sendiri, jangan salin ke dokumen atau Git.

| Peran | Akun demo | Dashboard |
| --- | --- | --- |
| Pasien | aulia@demo.hygeia.test | `/dashboard/patient` |
| Dokter | maya@demo.hygeia.test | `/dashboard/doctor` |
| Dokter lain | budi@demo.hygeia.test | `/dashboard/doctor` |
| Dokter IGD | rina@demo.hygeia.test | `/dashboard/doctor` |
| Laboratorium | lab@demo.hygeia.test | `/dashboard/lab` |
| Apotek | apotek@demo.hygeia.test | `/dashboard/pharmacy` |
| Admin | admin@demo.hygeia.test | `/dashboard/admin` |

Untuk menyalakan ulang: aktifkan Docker Desktop dan container PostgreSQL
`postgres-local`, lalu dari `backend` jalankan `docker compose up -d --wait`.
Dari `frontend`: `pnpm dev --host 127.0.0.1 --port 5188 --strictPort`.
Origin 5188 mengikuti konfigurasi CarePass lokal. Compose backend mengandalkan
database yang sudah ada, tidak membuat database baru.

## Tampilan yang sudah dibuat

Ada empat rute publik/alur masuk dan lima dashboard per peran. Dashboard memakai
30 menu total; menu berubah melalui `?view=...`, bukan 30 file halaman terpisah.

| Ruang | Menu dan kemampuan |
| --- | --- |
| Publik | Landing, login, registrasi pasien, halaman penebusan CarePass |
| Pasien: 10 menu | Ringkasan, Perawatan, Pemeriksaan, Resep & apotek, Izin dokter, CarePass, Perjalanan, Riwayat akses, Akses darurat, Notifikasi |
| Dokter: 8 menu | Ringkasan dokter, Pasien & perawatan, Pemeriksaan lab, Resep, Permintaan akses, Tebus CarePass, Akses darurat, Notifikasi |
| Lab: 4 menu | Ringkasan lab, Permintaan masuk, Koreksi hasil, Notifikasi; detail permintaan dan input hasil melalui dialog |
| Apotek: 2 menu | Antrean resep, Notifikasi; detail obat, status penyiapan, dan konfirmasi penyerahan |
| Admin: 6 menu | Organisasi, Akun, Status audit, Tinjauan akses darurat, Log aktivitas, Notifikasi |

Sumber menu: `frontend/src/lib/workflows/api.ts`. Detail hasil/resep menyediakan
cetak melalui dialog browser, termasuk opsi simpan PDF. Ini berbeda dari upload
PDF atau layanan pembuat PDF di server. Registrasi mandiri hanya untuk pasien;
akun staf dikelola admin.

## Lingkup produk

Hygeia adalah prototipe pertukaran data perawatan berbasis persetujuan pasien.
Alur utamanya: dokter mengundang pasien → pasien menerima → dokter meminta lab
→ pasien menyetujui → lab mengirim hasil → dokter membuat resep → pasien memilih
apotek → apotek memproses sampai selesai → pasien memeriksa timeline dan audit.

| Bagian | Implementasi yang ditemukan |
| --- | --- |
| Identitas | Sesi cookie, CSRF, login/logout, registrasi pasien, pembatasan login, pembersihan sesi kedaluwarsa |
| Perawatan | Undang, terima, tolak, akhiri episode; otorisasi pasien/dokter |
| Lab | Permintaan, persetujuan, antrean organisasi, accepted/in_progress, hasil terstruktur, koreksi append-only |
| Resep/apotek | Resep multiobat, pilihan apotek oleh pasien, accepted/preparing/ready/completed |
| Consent | Permintaan akses lintas organisasi, persetujuan berdurasi, penolakan, pencabutan, kedaluwarsa |
| CarePass | QR/tautan satu resource, kedaluwarsa, penebusan, pencabutan |
| Darurat | Alasan wajib, akses terbatas waktu/cakupan, notifikasi dan tinjauan admin |
| Bukti akses | Timeline, ringkasan akses, audit HMAC berantai, verifikasi dan dukungan rotasi kunci |
| Notifikasi | Inbox, status dibaca, pembaruan SSE |
| Administrasi | Organisasi, akun, penonaktifan akun, status audit dan tinjauan akses darurat |

Di luar MVP: AI diagnosis, integrasi SATUSEHAT/BPJS/asuransi/RS nyata,
pembayaran dan klaim, rawat inap/ERP, unggah PDF hasil lab, serta penggunaan data
pasien sungguhan. Reset password, verifikasi email, dan MFA belum tersedia;
ketiganya perlu keputusan cakupan tersendiri dan bukan syarat fungsional MVP
yang tercantum pada PRD saat ini.

## Hasil pemeriksaan sesi ini

- Backend dan PostgreSQL berhasil dihidupkan; health/readiness HTTP 200.
- `scripts/Test-Backend.ps1 -UseLocalDatabase`: lulus, mencakup tes integrasi
  PostgreSQL dan `go vet`; schema tes dibersihkan oleh suite.
- `pnpm check`: percobaan awal gagal menemukan generated `./$types`; setelah
  build/sinkronisasi, pengulangan lulus dengan 0 error dan 0 warning. Penyebab
  kegagalan awal belum dipastikan; cek kembali pada checkout bersih sebelum rilis.
- `pnpm lint`: lulus sebelum perubahan tes di sesi ini.
- `pnpm build`: lulus; adapter-auto belum memilih target produksi.
- `pnpm test`: dua file, dua tes lulus. Keduanya contoh unit test; ini tidak
  menggantikan tes domain/integrasi dan E2E yang terpisah.
- Tes browser live dengan API lokal lulus: undangan, lab, resep/apotek,
  consent/cabut izin, CarePass/cabut, darurat, notifikasi, timeline, dan layout
  1440/768/414/375/320 px. Audit data sintetis: 59 event, `intact=true`.
  Data sintetis suffix `1c63832f` dipertahankan sesuai desain tes.
- Suite landing/dashboard fixture lulus, termasuk kondisi kosong/error,
  pemulihan retry, navigasi, FAQ keyboard, dan berbagai lebar layar.
- Ditemukan masalah urutan tunggu pada tes clinical: helper menunggu judul
  dashboard saat skenario sesi berakhir mengalihkan ke login. Dua langkah
  pengujian diubah untuk langsung menunggu URL login. Tes clinical ulang lulus,
  termasuk redirect SSE session-ended dan API 401. Formatting dan ESLint file
  tes yang diubah juga lulus.
- Tes fixture lab/apotek/admin dan registrasi pasien lulus secara terpisah.
- Pengulangan penuh `pnpm test:e2e` setelah perbaikan: lulus, exit code 0;
  seluruh suite landing/dashboard, clinical, provider, dan registrasi selesai.

## Checklist untuk menyelesaikan MVP

Urutan ini membatasi pekerjaan pada hasil demo yang dapat diperiksa.

- [x] Selesaikan dan catat hasil ulang seluruh suite fixture setelah perbaikan tes.
- [ ] Ulangi `pnpm check` pada checkout/install bersih untuk memeriksa generated types.
- [ ] Selaraskan BRD 6.2: CarePass dan realtime masih tertulis ditunda, padahal
  keduanya sudah diimplementasikan. Pertahankan catatan sejarah, jelaskan scope kini.
- [ ] Perbarui readiness lama yang masih menyebut migrasi 9; implementasi dan
  setup terbaru menggunakan migrasi 10. Jangan menjadikan HTTP readiness sebagai
  bukti versi skema karena endpoint saat ini hanya memeriksa koneksi DB.
- [ ] Tinjau perubahan lokal dan file baru, commit per kelompok yang bisa direview,
  lalu jalankan CI GitHub. Jangan memasukkan `.env`, hasil tes, atau data demo.
- [ ] Tentukan host/domain, adapter SvelteKit, HTTPS, proxy `/api` satu origin,
  origin CarePass, dan proxy SSE tanpa buffering; uji di server tujuan.
- [ ] Jalankan satu skenario demo lengkap dengan seluruh peran di deployment.
  Cek juga penolakan, izin dicabut/kedaluwarsa, akun nonaktif, sesi berakhir,
  konflik status, jaringan gagal, pagination, cetak hasil terkoreksi, dan mobile.
- [ ] Siapkan akun/data sintetis demo dan naskah demo 3 menit serta 6–8 menit.
  Pastikan seluruh tautan CarePass memakai origin deployment yang benar.

Untuk produksi/data nyata diperlukan pekerjaan tambahan: backup database dan
kunci audit beserta uji restore, pengelolaan secret, TLS database, monitoring,
uji beban, kebijakan limiter di balik proxy, dan penilaian privasi/keamanan.
Checkpoint audit eksternal juga belum ada. Ini belum dibuktikan oleh lulusnya
tes lokal. Tidak ada hasil deployment atau CI remote yang diklaim pada sesi ini.

## Batas kesimpulan

Alur inti dapat dijalankan lokal dan sudah diuji dengan data sintetis. Tes live
memakai browser pasien/dokter; beberapa tindakan lab/apotek/admin dilakukan via
API. Tes fixture provider memeriksa UI peran tersebut secara terpisah. Registrasi
dan koreksi lab memiliki tes integrasi backend serta fixture frontend; sesi ini
belum membuktikan seluruh kombinasi UI itu terhadap server production.

Target selesai yang dapat dinilai adalah: checklist MVP tercentang, seluruh
pemeriksaan lulus pada revisi yang sama, dan alur demo lulus pada deployment.
Pengujian mengurangi risiko; tidak membuktikan aplikasi bebas semua bug.
