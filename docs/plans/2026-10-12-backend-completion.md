# Penyelesaian backend Hygeia

## Cakupan yang disetujui

Admin, notifikasi dalam aplikasi, akses darurat, CarePass/QR, dan realtime.
Setiap tahap memiliki branch, tes, commit, kontrak API, dan PR; PR tetap terbuka.
Review standar dan kesesuaian spesifikasi dilakukan setelah seluruh tahap.
Perubahan frontend lokal tidak disertakan. Timeline berada pada PR #15.

## Hasil akhir

Seluruh tahap di bawah sudah diimplementasikan, dicommit, dipush, dan direview.
PR tetap terbuka. Urutan merge: [#10](https://github.com/mrizalbasri/Hygeia/pull/10)
→ [#11](https://github.com/mrizalbasri/Hygeia/pull/11)
→ [#12](https://github.com/mrizalbasri/Hygeia/pull/12)
→ [#13](https://github.com/mrizalbasri/Hygeia/pull/13)
→ [#14](https://github.com/mrizalbasri/Hygeia/pull/14)
→ [#15](https://github.com/mrizalbasri/Hygeia/pull/15).
Timeline ditempatkan terakhir dan branch-nya sudah menggabungkan seluruh tahap,
sehingga hasil integrasi dapat diuji. Detail verifikasi dan hasil review:
[backend-review](2026-10-12-backend-review.md).

## Urutan dan rancangan

1. **Admin minimal** (`feature/backend-admin`, base main): daftar/buat organisasi,
   ganti nama organisasi, daftar/buat akun, ubah nama serta nonaktifkan/aktifkan
   akun. Peran/organisasi akun tetap saat pembuatan untuk menghindari pemindahan
   izin diam-diam. Password Argon2id; akun staf wajib organisasi yang cocok;
   pasien mendapat referensi acak. Admin tidak bisa menonaktifkan dirinya.
   Nonaktifkan akun menghapus sesi; pengaktifan tidak menghidupkan sesi lama.
   Daftar kepala rantai audit memakai metadata; verifikasi memakai API yang ada.
   Mutasi admin menulis administrative activity append-only dalam transaksi;
   ledger administratif ini terpisah dari rantai HMAC pasien dan tidak memuat
   password atau isi klinis. Migrasi 6.
2. **Notifikasi** (`feature/backend-notifications`, base branch admin): daftar
   notifikasi milik sesi, jumlah belum dibaca, tandai satu sebagai dibaca.
   Audit writer meneruskan milestone sukses ke penerima melalui insert dalam
   transaksi yang sama. Pasien, dokter episode, dan staf organisasi penerima
   dipilih berdasarkan jenis kejadian; aktor sendiri tidak dinotifikasi.
   Payload hanya metadata resource dan pesan, tanpa hasil/obat/alasan bebas.
   Unique sumber audit/penerima mencegah duplikasi. Migrasi 7.
3. **Akses darurat** (`feature/backend-break-glass`, base notifikasi): dokter klinik
   memulai sesi pasien dengan kode dan teks alasan wajib; berlaku 60 menit,
   terikat dokter+klinik. Resource yang dapat dilihat hanya hasil lab selesai
   dan resep aktif. Daftar resource berupa ID; endpoint detail yang ada memakai
   policy grant darurat. Pasien mendapat notifikasi dalam transaksi pembuatan.
   Audit menyertakan ID/masa berlaku grant, tanpa teks alasan klinis. Admin
   meninjau metadata/alasan dan memberi keputusan/catatan, tanpa isi klinis.
   Pemeriksaan waktu dilakukan setelah lock agar expiry tidak terlewati.
   Migrasi 8.
4. **CarePass/QR** (`feature/backend-carepass`, base akses darurat): pasien membuat
   tautan untuk satu hasil lab/resep, TTL 15/60 menit, token acak 256-bit hanya
   disimpan sebagai hash. QR PNG dan tautan fragment dikembalikan sekali saat
   create. Dokter terautentikasi menebus token; grant terikat dokter+klinik
   penebus pertama, bukan akses anonim. Pasien dapat melihat dan mencabut pass;
   akses berikut langsung memeriksa revocation/expiry. Detail tetap melalui
   endpoint klinis, dengan basis audit CarePass. Migrasi 9.
5. **Realtime** (`feature/backend-notification-stream`, base CarePass): SSE
   notifikasi per sesi dengan cursor/Last-Event-ID, heartbeat, bounded batches,
   batas koneksi, pemeriksaan sesi ulang, serta penghentian saat disconnect.
   SSE tidak mengubah status dibaca. Deadline streaming dikelola khusus agar
   tidak diputus WriteTimeout API biasa. Tidak perlu migrasi.

## Pengujian dan review akhir

- Isolasi peran/pasien/organisasi, CSRF, validasi body, password tidak bocor,
  sesi nonaktif, dan rollback mutasi saat administrative activity gagal.
- Fanout notifikasi dan rollback transaksi, pagination/read ownership, tanpa
  payload klinis, dan tidak ada notifikasi untuk akses ditolak/read.
- Akses darurat/CarePass: scope, masa berlaku, transfer organisasi, revoke,
  race redemption/lock wait, keterikatan penerima, serta audit dan notifikasi.
- SSE: replay cursor, isolasi pengguna, autentikasi ulang, framing, dan cleanup.
- Regresi Go dengan PostgreSQL khusus tes, vet, build, dan migrasi berulang.
- Review keseluruhan terhadap origin/main dan BRD/PRD/kontrak; temuan yang nyata
  diperbaiki dan diuji sebelum laporan final. PR disajikan dalam urutan merge.
