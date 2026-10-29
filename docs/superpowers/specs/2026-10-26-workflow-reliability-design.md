# Keandalan ringkasan, hasil lab, dan notifikasi

## Lingkup yang disetujui

Tahap pertama mempertahankan identitas visual dan aturan akses Hygeia. Tidak
menambahkan integrasi eksternal, pemulihan akun, atau konfigurasi deployment.

## Ringkasan pasien

`GET /api/v1/patient/work-summary` mengembalikan jumlah seluruh perawatan aktif,
lab dalam proses, resep siap diambil, serta persetujuan tertunda per kategori:
undangan, pemeriksaan, resep, dan izin dokter. Identitas hanya dari sesi pasien.
Satu query SQL membaca snapshot yang sama, tanpa pagination dan tanpa nilai
klinis. Peran lain ditolak. Daftar terbaru tetap dibatasi enam entri; kartu
tindakan menampilkan jumlah per kategori dan tautan ke workflow yang sesuai.
Kegagalan ringkasan ditampilkan sebagai galat dan tanda pisah, bukan nol.

## Pengiriman hasil dan koreksi

Penanda tidak memiliki nilai awal. Petugas harus memilih secara eksplisit.
Submit pertama memvalidasi input dan menampilkan tinjauan seluruh nilai serta
konteks pemeriksaan/pasien atau ID hasil koreksi. Tidak ada mutasi pada langkah
ini. Petugas dapat kembali mengedit. Hanya konfirmasi akhir mengirim snapshot
yang ditinjau. Kegagalan mempertahankan input; koreksi tetap append-only.

## Notifikasi

Satu pengelola berbasis context Svelte dimiliki kerangka dashboard per akun.
Koneksi tidak dimulai saat SSR dan dibersihkan saat akun berubah atau keluar.
Header menampilkan jumlah belum dibaca. Halaman Notifikasi memakai pengelola
yang sama, bukan membuka SSE kedua. Event hanya menginvalidasi data; nilai
klinis tetap diminta melalui API berotorisasi. Reconnect memperbarui jumlah
dan revisi untuk mengejar perubahan saat koneksi putus. Ada pemulihan manual
jika inisialisasi gagal, dan sesi berakhir mengalihkan pengguna ke login.

## Verifikasi

- Integrasi backend: lebih dari satu halaman, isolasi pasien, seluruh kategori,
  izin aktif/kedaluwarsa tidak masuk pending, peran salah, sesi berakhir.
- Browser fixture: ringkasan tidak tergantung enam entri terbaru, kegagalan dan
  retry, penanda wajib, tinjauan tanpa POST, edit ulang, kegagalan pengiriman.
- Browser fixture realtime: satu SSE lintas menu, badge, event/reconnect,
  kegagalan koneksi, session-ended, serta layar desktop dan mobile.
- Jalankan check, lint, build, unit test, E2E frontend dan tes/vet backend.
  Catat tes database yang tidak dapat dijalankan; jangan menganggap skip sebagai
  bukti integrasi.
