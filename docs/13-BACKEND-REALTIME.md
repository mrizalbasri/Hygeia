# Realtime notifikasi (SSE)

`GET /api/v1/notifications/stream?after_id=0` memakai cookie sesi yang sama.
`Last-Event-ID` mengalahkan `after_id` saat reconnect. Cursor adalah integer
non-negatif, default 0: replay seluruh notifikasi milik pengguna dari yang
terlama, 50 per batch. Klien yang sudah memuat inbox dapat memulai dari ID
terbesar yang sudah diterima. Tidak ada selector pasien/pengguna di endpoint.

```text
retry: 3000
: connected

id: 123
event: notification
data: {"id":123,"action":"prescription.ready",...}

```

Gunakan `EventSource('/api/v1/notifications/stream')` pada origin aplikasi dan
listener `notification`. Deduplicasi berdasarkan ID saat menggabungkan dengan
inbox REST. SSE tidak menandai notifikasi dibaca. Isi event sama dengan item
REST: metadata dan pesan statis, tanpa isi klinis atau token CarePass.

Poll database setiap 2 detik; heartbeat komentar setiap 15 detik. Sesi dicek
ulang pada setiap batch/poll: logout, expiry atau akun nonaktif menghasilkan
`event: session-ended` dengan `data: {}`, lalu stream ditutup. Klien harus
menutup EventSource dan kembali ke login. Gangguan database mengirim
`event: unavailable` jika masih bisa menulis, lalu menutup koneksi agar replay
dapat dicoba ulang. Cursor bukan acknowledgement perubahan read state.

Batas per proses: 2 stream per akun dan 128 total; HTTP 429 + Retry-After 5
sebelum stream dimulai. Batas dilepas saat disconnect/error. Database connection
tidak ditahan saat menunggu client. Write deadline setiap frame 5 detik,
terpisah dari WriteTimeout request JSON. Tidak mendukung HEAD untuk streaming.
Skala multi-instance memerlukan kuota bersama jika batas global diperlukan.

Reverse proxy harus meneruskan stream tanpa buffering/cache, menjaga idle
timeout di atas 15 detik dan menggunakan origin yang sama dengan aplikasi.
Header `X-Accel-Buffering: no` membantu Nginx; pastikan konfigurasi proxy mendukung.

ID bigserial sendiri tidak menjamin urutan commit. Producer memakai advisory
transaction lock sebelum alokasi notifikasi sampai commit, agar notifikasi ID
lebih kecil tidak muncul terlambat setelah cursor lebih besar terkirim. Lock
global ini cocok untuk volume MVP; penulisan milestone menunggu transaksi
penghasil notifikasi lain, sedangkan pembacaan inbox/SSE tidak mengambil lock.
Tidak ada migrasi tambahan; gunakan migrasi 1–9 dari rangkaian PR sebelumnya.
