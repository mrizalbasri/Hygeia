# Rencana backend timeline perjalanan pasien

## Tujuan

Menambahkan `GET /api/v1/patient/timeline` untuk perjalanan pasien: undangan dan
keputusan episode, permintaan/proses/hasil lab, penerbitan dan pemenuhan resep,
serta permintaan/keputusan consent. Pekerjaan mencakup backend, kontrak API,
pengujian, commit, dan PR. UI, admin, akses darurat, dan notifikasi terpisah.

## Rancangan

Timeline merupakan proyeksi kejadian bisnis sukses pada audit HMAC pasien yang
sudah ditulis dalam transaksi aksi bisnis. Ini mempertahankan transisi historis
dan langsung mencakup data lama tanpa duplikasi tabel atau backfill. Alternatif
tabel timeline tersendiri menambah dual-write dan migrasi; merekonstruksi dari
status resource saat ini menghilangkan tahapan historis. Proyeksi audit dipilih.

- Hanya pasien pemilik; rantai diturunkan dari sesi, tanpa selector pasien.
  Staf, dokter, admin, serta sesi tidak sah ditolak.
- Allowlist eksplisit mengecualikan pembacaan/list, akses ditolak, perubahan
  status yang gagal, dan aksi tak dikenal. Consent kedaluwarsa otomatis tidak
  dibuat sebagai kejadian palsu karena tidak ada event bisnis tersimpan.
- Event memuat nomor urut, waktu, jenis (`episode`, `laboratory`, `prescription`,
  `consent`), aksi, resource type/ID, nama aktor historis, dan kalimat Indonesia.
  Isi hasil lab, obat, tujuan consent, dan metadata konteks bebas tidak disertakan.
- Urutan terbaru dahulu berdasarkan seq rantai, bukan waktu jam server.
  Filter `kind` opsional, `limit` 1–100 (default 20), dan cursor eksklusif
  `before_seq` positif untuk halaman berikut. Ambil limit+1 agar cursor hanya
  muncul jika masih ada kejadian. Cursor stabil saat event baru ditambahkan.
- Validasi menolak query duplikat, selector pasien/rantai, parameter tak dikenal,
  filter tak sah, cursor nonpositif/overflow, serta limit di luar batas.
- Modul `timeline` memiliki proyeksi dan query; `care.Service` menyelesaikan sesi
  dan pasien melalui transaksi yang sudah ada; handler care memakai validasi
  query endpoint ini. Tidak memerlukan migrasi atau perubahan format hash.
- Pembacaan timeline tidak menulis event baru, konsisten dengan pembacaan
  ringkasan akses. Timeline bukan klaim verifikasi integritas; endpoint audit
  verify tetap tersedia terpisah.

## Tahapan

1. Buat branch `codex/backend-patient-timeline` dan commit rencana.
2. Implementasikan proyeksi, endpoint, validasi, dan pagination cursor.
3. Uji perjalanan episode/lab/resep/consent, isolasi pasien/peran, penolakan
   parameter, pengecualian akses gagal/read, teks historis, pagination saat
   event baru masuk, dan rollback aksi gagal audit.
4. Jalankan regresi Go dengan PostgreSQL khusus pengujian, vet, dan build;
   dokumentasikan kontrak dan bukti hasil. Commit dan buka PR ke main.

## Kriteria selesai

Pasien melihat urutan milestone yang akurat tanpa isi klinis atau aktivitas
pasien lain; halaman berlanjut tanpa duplikasi ketika ada event baru; aksi
yang gagal tidak muncul; seluruh tes relevan lulus dan PR tidak menyertakan
perubahan frontend lokal yang sudah ada.

## Hasil verifikasi

Implementasi dan verifikasi selesai. PostgreSQL 18 khusus pengujian digunakan
dengan schema terisolasi; database aplikasi tidak diubah.

- Tes timeline lulus: seluruh 23 jenis milestone, semua filter, nama historis,
  isolasi pasien/peran, sesi kedaluwarsa, validasi query, pagination saat insert
  baru, refresh tanpa tulisan audit, rollback bisnis, dan kegagalan query.
- Allowlist menolak outcome deny, read, action baru yang belum didukung, dan
  pasangan action/resource type yang tidak cocok.
- Regresi seluruh backend lulus dengan `TEST_DATABASE_URL`:
  `go test ./cmd/... ./db/... ./internal/... -count=1`.
- `go vet ./cmd/... ./db/... ./internal/...` dan build `./cmd/api` lulus.
- Tidak ada migrasi; schema tetap versi 5. Frontend lokal tidak disertakan.
