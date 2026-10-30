# Pemulihan flow dan pembaruan realtime

Lingkup: empat celah hasil review flow yang disetujui pengguna pada 29 Oktober
2026. Perubahan lokal; tidak mencakup commit, push, atau deployment.

- Koreksi lab menghasilkan notifikasi kepada pasien dan dokter episode aktif,
  serta milestone timeline, dalam transaksi audit yang sama. Isi klinis dan
  alasan bebas tidak disalin ke notifikasi. Panel hasil menonjolkan koreksi
  terbaru dan memisahkan seluruh versi lama sebagai riwayat.
- Lab memiliki daftar tanda terima terpaginasikan dari organisasinya sendiri.
  Daftar hanya berisi identitas permintaan/hasil, referensi pasien, nama/kode tes,
  waktu, dan status; tidak berisi nilai, catatan, atau riwayat koreksi klinis.
- Pasien pemilik, dokter episode aktif, dan penyedia yang ditugaskan boleh
  membatalkan permintaan aktif dengan alasan 5–500 karakter. Penyedia tidak
  boleh membatalkan permintaan yang belum disetujui pasien. Status selesai dan
  dibatalkan bersifat terminal; pembatalan bersaing dengan pemrosesan melalui
  urutan lock episode → resource → audit yang sudah ada. Organisasi dan waktu
  persetujuan lama tetap tersimpan. Alasan tersedia untuk pasien/dokter,
  sedangkan notifikasi hanya menyampaikan pembatalan tanpa teks bebas.
- Penggantian dilakukan sebagai permintaan/resep baru oleh dokter episode
  aktif, menggunakan detail lama sebagai awal. Lab baru dipilih dokter;
  apotek dipilih kembali oleh pasien. Persetujuan baru selalu diperlukan;
  permintaan lama tetap dibatalkan dan item resep lama tidak disunting.
- SSE yang tersedia dipakai sebagai sinyal memuat ulang data terotorisasi,
  bukan membawa nilai klinis. Satu koneksi dibagi antar komponen dalam tab.
  Daftar, ringkasan, dan detail terbuka diperbarui setelah event, reconnect,
  atau kembali ke tab. Retry bootstrap dan polling cadangan 30 detik menutup
  kegagalan koneksi. Draft form tidak dihapus akibat pembaruan daftar.
  SSE backend memeriksa database setiap 2 detik; target lokal perubahan terlihat
  dalam 5 detik saat koneksi sehat. Webhook belum diperlukan karena tidak ada
  integrasi penyedia eksternal pada lingkup ini.

Alternatif: polling saja lebih sederhana tetapi lebih lambat dan boros request;
webhook cocok untuk server eksternal tetapi tetap membutuhkan kanal browser.
Memperluas SSE mempertahankan mekanisme autentikasi, replay cursor, dan pembatasan
koneksi yang sudah diuji.

Validasi: integrasi PostgreSQL untuk notifikasi/timeline koreksi, isolasi tanda
terima, pembatalan/otorisasi/rollback/race; browser untuk hasil terbaru, refresh
tanda terima, penggantian/persetujuan ulang, SSE/reconnect/fallback dan draft form.
Jalankan check, lint, build, backend tests/vet dan suite browser secara berurutan.
