# Keputusan backend: care episode dan audit

Tanggal: 11 Oktober 2026. Lingkup yang disepakati: backend saja; frontend
diintegrasikan oleh rekan pengembang melalui kontrak `05-BACKEND-API.md`.

## Perilaku

- Dokter pada klinik yang valid membuat undangan menggunakan `patient_ref`
  persis, peka huruf besar/kecil. Tidak ada endpoint pencarian seluruh pasien.
- Satu pasangan dokter–pasien hanya boleh mempunyai satu episode `pending`
  atau `active`. Episode berbeda untuk dokter lain tetap dimungkinkan.
- Hanya pasien pemilik yang bisa menerima/menolak: `pending → active/rejected`.
- Pasien pemilik atau dokter pengundang bisa mengakhiri: `active → ended`.
- Status terminal tidak bisa diaktifkan kembali. Undangan baru diperbolehkan.
- Pembatalan undangan oleh dokter dan kedaluwarsa undangan belum termasuk tahap ini.
- Metadata episode tersedia bagi kedua peserta, termasuk setelah berakhir.
  Data klinis jalur A memerlukan episode aktif; endpoint lab mengikuti policy ini.
- Rekan satu klinik, dokter di organisasi lain, lab, apotek, dan admin tidak
  memperoleh akses episode. Dokter yang pindah klinik kehilangan akses episode lama.
- Daftar episode memuat undangan dan riwayat milik aktor; filter `active`
  dipakai frontend untuk daftar hubungan perawatan aktif.
- Pengakhiran episode membatalkan semua permintaan lab yang belum disetujui,
  dengan event audit per permintaan dalam transaksi yang sama. Penugasan lab
  yang sudah disetujui tetap bisa diselesaikan meski episode berakhir. Pasien
  tetap bisa membaca hasil; dokter kehilangan akses jalur A setelah episode berakhir.
- Consent jalur C untuk satu hasil lab merupakan izin terpisah, terikat dokter,
  klinik, dan durasi. Consent aktif tetap dapat memberi akses setelah episode
  berakhir, hingga dicabut atau habis. Pencabutan C tidak mengakhiri episode A.

## Implementasi

Memakai `net/http` dan query parameter `pgx`, mengikuti fondasi yang sudah ada.
Rencana router chi/sqlc dalam arsitektur belum diterapkan. Tidak ada perubahan UI
atau penambahan layanan AI. Paket `policy` menjadi tempat keputusan akses episode;
Modul lab menegakkan policy data klinis; modul resep nantinya harus mengikuti pola ini.

Setiap operasi domain membaca identitas/sesi aktif dari DB di dalam transaksi
dengan timeout 5 detik. Mutasi mengunci baris episode `FOR UPDATE`; partial unique
index mencegah undangan ganda. Pagination dibatasi maksimal 100 baris.

Audit ditulis dalam transaksi yang sama. Penolakan setelah menemukan episode
tetap di-commit sebagai event `deny`; aksi bisnis belum dilakukan pada jalur itu.
Kegagalan SQL atau audit membatalkan transaksi; data tidak dikirim ke klien.
Daftar dicatat sekali per pasien yang muncul pada halaman, dengan contoh ID
episode dari pasien tersebut. Rantai dikunci dalam urutan ID pasien untuk
menghindari deadlock antar halaman yang memuat beberapa pasien.

Pemeriksaan tanpa sesi, CSRF tidak valid, input tidak valid, resource tidak ada,
kode pasien tidak ditemukan, dan daftar kosong tidak menghasilkan event rantai
pasien karena belum ada akses resource pasien yang sah. Login belum diaudit.
Pembacaan ringkasan audit dan verifikasi tidak mencatat dirinya sendiri.

## Integritas audit

Satu rantai per `patients.id`, bukan user ID. Event memakai nomor urut dan
`HMAC-SHA256(AUDIT_HMAC_KEY, prev_hash || canonical(event))`.

Canonical v1 mengikuti urutan field payload pada `internal/audit/audit.go`,
dengan key context diurutkan, UTF-8 JSON dari `encoding/json`, timestamp UTC
presisi enam digit mikrodetik, dan `prev_hash` awal 32 byte nol. Nama/peran aktor
serta organisasi disimpan sebagai snapshot dan ikut dihitung dalam HMAC.

Kepala rantai dikunci `FOR UPDATE` saat append dan `FOR SHARE` saat verifikasi.
Verifikasi mengecek payload, tautan hash, urutan, dan kecocokan kepala dengan
event terakhir, termasuk penghapusan ekor. Trigger menolak UPDATE, DELETE, dan
TRUNCATE biasa pada audit events.

Pemilik DB tetap bisa menonaktifkan trigger. Akses ke DB beserta kunci HMAC
memungkinkan penulisan ulang seluruh rantai; ini bukti deteksi perubahan, bukan
jaminan tidak dapat diubah. Kunci harus disimpan dan dicadangkan terpisah dari
database. Mengganti kunci membuat rantai lama gagal diverifikasi; rotasi kunci
belum diimplementasikan. Rantai sistem dan checkpoint eksternal belum tersedia.

## Validasi yang diperlukan

Tes matriks policy, lifecycle dan otorisasi HTTP, CSRF, duplikasi undangan,
konkurensi accept/reject dan append, kegagalan audit dengan rollback, trigger
append-only, deteksi manipulasi payload dan penghapusan ekor. Tes integrasi memakai
schema acak terpisah dan membersihkannya setelah selesai; tidak memodifikasi
episode atau audit demo di schema aplikasi.

Referensi implementasi: [PostgreSQL row locking](https://www.postgresql.org/docs/current/explicit-locking.html)
dan [pgx transaction API](https://pkg.go.dev/github.com/jackc/pgx/v5#Tx).
