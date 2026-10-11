# Rencana backend resep dan apotek

## Tujuan dan cakupan

Melengkapi alur dokter → pasien → apotek dengan modul Go/PostgreSQL yang
mengikuti pola care, laboratory, policy, dan audit yang sudah ada. Pekerjaan ini
mencakup backend, dokumentasi kontrak API, pengujian, branch, commit, dan PR.
Timeline klinis, admin, akses darurat, notifikasi, QR, realtime, dan UI resep
merupakan pekerjaan lanjutan terpisah.

## Keputusan desain

- Dokter episode aktif membuat resep berisi 1–20 item: nama obat, dosis,
  aturan pakai, dan jumlah. Semua contoh menggunakan data sintetis.
- Resep dimulai dengan `pending_patient_approval`; dokter tidak memilih apotek.
- Pasien pemilik memilih satu organisasi `pharmacy` sekaligus menyetujui
  pengiriman, atau menolak resep. Pemilihan bersifat satu kali, tanpa pemindahan
  apotek atau penyuntingan isi resep pada MVP ini.
- Status: `pending_patient_approval → sent → accepted → preparing → ready →
  completed`. Penolakan pasien atau episode yang diakhiri membatalkan resep yang
  belum disetujui. Penugasan yang sudah disetujui tetap dapat diselesaikan apotek
  setelah episode berakhir, sesuai perilaku penugasan lab yang sudah ada.
- Pasien selalu dapat melihat resepnya; dokter pembuat hanya selama episode
  aktif; staf apotek tujuan hanya selama status `sent|accepted|preparing|ready`.
  Apotek menerima proyeksi minimal: ID resep, referensi pasien, item, dan status.
  Dokter lain, rekan satu klinik, lab, admin, dan apotek lain ditolak.
- API: direktori apotek untuk pasien, buat/daftar/detail resep, persetujuan
  pasien, dan pembaruan status apotek. Daftar mendukung status dan pagination.
- Mutasi memakai sesi dan CSRF; JSON ketat, UUID dan batas item/teks divalidasi.
  Tidak ada detail obat dalam audit atau respons error.
- Setiap akses ke resep yang ada dan keputusan bisnis diizinkan/ditolak dicatat
  dalam rantai HMAC pasien. Aksi dan audit berada dalam satu transaksi; jika
  audit gagal, perubahan dibatalkan dan payload tidak dikembalikan.
- Lock episode → resep → rantai audit menyelaraskan race approval/end/status.
  Daftar mengunci episode sebelum memeriksa ulang izin agar penutupan episode
  tidak mengembalikan data klinis kepada dokter yang sudah kehilangan akses.

## Tahapan dan commit

1. Dokumentasikan rencana dan buat branch `codex/backend-prescriptions-pharmacy`.
2. Tambahkan migrasi 5 untuk prescriptions dan prescription_items; policy,
   service, handler; registrasi API; pembatalan pending saat episode berakhir;
   ringkasan audit dalam bahasa Indonesia.
3. Uji validasi, matriks izin, lifecycle HTTP lengkap dengan PostgreSQL,
   kerahasiaan proyeksi apotek, penolakan/episode berakhir, race, dan rollback
   ketika audit gagal. Jalankan regresi Go dan vet.
4. Perbarui README dan kontrak API termasuk contoh request/response, error,
   transisi, serta cara migrasi. Commit kode dan bukti pengujian.
5. Push branch dan buka PR dengan hasil pengujian dan risiko migrasi. Jangan
   memasukkan perubahan frontend yang sudah ada ke commit ini.

## Kriteria selesai

Alur resep dapat dijalankan melalui API dari episode aktif sampai selesai,
akses lintas pasien/organisasi tidak membocorkan item obat, setiap transisi sah
tercatat dan rantai utuh, kegagalan audit membatalkan transaksi, serta PR berisi
hanya pekerjaan backend resep/apotek dan dokumentasinya.

## Hasil verifikasi

Implementasi dan kontrak API selesai pada 11 Oktober 2026. PostgreSQL 18 khusus
pengujian dipakai terpisah dari database aplikasi. Pemeriksaan berikut lulus:

- Seluruh tes unit dan integrasi:
  `go test ./cmd/... ./db/... ./internal/... -count=1` dengan `TEST_DATABASE_URL`.
- `go vet ./cmd/... ./db/... ./internal/...` dan build `./cmd/api`.
- Migrasi `up` dua kali, lalu `version`: versi 5, `dirty: false`.
- Tes resep mencakup 20 item berurutan, lifecycle, akses/proyeksi, penolakan,
  episode berakhir, approval/end bersamaan, transisi ganda, pembacaan setelah
  menunggu lock, integritas rantai, dan rollback ketika audit gagal.

Perubahan frontend lokal yang sudah ada tidak dimasukkan ke commit pekerjaan
ini. Migrasi database aplikasi dan integrasi UI resep dilakukan saat penerapan
berikutnya; pengujian ini tidak mengubah database aplikasi.
