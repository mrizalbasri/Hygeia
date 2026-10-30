# Perbaikan flow dan realtime — 29 Oktober 2026

Perubahan lokal berdasarkan review flow dan persetujuan pengguna. Tidak ada
commit, push, perubahan kredensial, atau deployment publik. Perubahan lokal
yang sudah ada sebelum pekerjaan ini dipertahankan.

## Perilaku yang diperbaiki

1. Koreksi hasil lab menghasilkan notifikasi generik untuk pasien dan dokter
   episode aktif serta milestone timeline, atomik dengan audit. Isi klinis dan
   alasan koreksi tidak dimasukkan ke notifikasi. Waktu koreksi ditentukan
   setelah resource terkunci agar urutannya sesuai penulisan.
2. Detail hasil menonjolkan koreksi terbaru; hasil awal dan seluruh koreksi
   tersedia di riwayat. Event realtime memperbarui detail yang sedang terbuka;
   pembacaan ulang yang ditolak menghapus payload lama.
3. Menu lab **Tanda terima hasil** mengambil metadata tersimpan dari organisasi
   sendiri, mendukung pagination dan tetap tersedia setelah reload/login ulang.
   Tautan koreksi mengisi ID hasil otomatis. Hak baca nilai klinis untuk lab
   tetap tertutup.
4. Pasien pemilik, dokter episode aktif, serta penyedia yang sudah ditugaskan
   dapat membatalkan layanan aktif dengan alasan 5–500 karakter. Backend
   menyimpan organisasi/waktu persetujuan sebelumnya. Audit, timeline,
   notifikasi dan status ditulis dalam satu transaksi. Penyedia lain tidak
   berhak, dan status selesai/dibatalkan tidak dapat dibuka kembali.
5. Dokter dapat mengajukan pemeriksaan pengganti ke lab pilihan atau menerbitkan
   resep baru dengan item sebelumnya. Pasien harus menyetujui kembali dan
   memilih apotek pada resep baru. Permintaan lama tetap dibatalkan;
   penggantian merupakan resource baru melalui endpoint create biasa.

## Realtime

`realtime.ts` membagi satu koneksi SSE per tab ke daftar, ringkasan pasien/dokter/
lab, inbox, serta detail. Event memicu GET berotorisasi; nilai klinis tidak
dikirim sebagai isi event. Backend memeriksa event setiap 2 detik. Reconnect dan
kembali ke tab memuat ulang data; bootstrap yang gagal dicoba lagi setelah
5 detik; polling cadangan setiap 30 detik saat tab terlihat. Draft pembatalan
pada baris yang masih tersedia bertahan saat pembaruan sukses.

SSE melanjutkan kebijakan notifikasi yang mengecualikan pelaku aksi; tab lain
milik pelaku sendiri tetap memiliki fallback berkala. Webhook belum diperlukan
untuk komunikasi server–browser; integrasi dengan sistem eksternal belum
termasuk lingkup ini. Proxy hosting tetap harus meneruskan SSE tanpa buffering.

## Bukti pemeriksaan

| Pemeriksaan | Hasil |
| --- | --- |
| `./scripts/Test-Backend.ps1 -UseLocalDatabase` | Lulus, termasuk integrasi PostgreSQL dan `go vet` |
| Regresi backend baru | Notifikasi/timeline koreksi dan event SSE nyata; isolasi tanda terima; otorisasi/CSRF/validasi pembatalan; riwayat persetujuan; race pembatalan versus penyelesaian; rollback saat audit gagal |
| `pnpm check` | Lulus, 0 error / 0 warning |
| `pnpm lint` | Lulus |
| `pnpm test` | 2 file / 2 tes lulus |
| `pnpm build` | Lulus; adapter-auto masih belum mendeteksi target produksi lokal |
| `pnpm test:e2e` | Ketujuh suite lulus dalam satu rangkaian akhir |
| Suite recovery baru | Koreksi pada detail terbuka, penghapusan payload setelah 403, satu stream bersama, draft tetap utuh, reload tanda terima, penggantian/persetujuan ulang, reconnect/retry/fallback, cleanup dan mobile 390 px |
| `HYGEIA_LIVE_TEST=1 node tests/clinical-workflows.live.js` | Lulus terhadap API lokal yang dibangun ulang; koreksi melalui API lab memperbarui detail dokter lewat SSE tanpa reload |
| Audit live terakhir | `intact=true`, 73 event, chain `4b14695d-4430-41e1-83af-8e79f7e98c5c`, fixture sintetis `a81e84e4` |
| Migrasi lokal | Versi 11, `dirty=false` |
| `docker compose up -d --build --wait` | API lokal dibangun ulang dan sehat |
| `git diff --check` | Lulus |

Tes landing disesuaikan dengan HeroThree yang sudah ada; interaksi dropdown
diuji lewat keyboard agar tidak dipengaruhi pembukaan saat hover. Fixture
ringkasan diperbarui untuk mengizinkan bootstrap SSE. Screenshot recovery
disimpan di `frontend/.test-results/workflow-recovery/` (diabaikan Git), dan
hasil koreksi desktop serta tanda terima mobile diperiksa secara visual.

Tes live memakai browser untuk pasien/dokter dan API untuk sebagian tindakan
lab/apotek. Tes fixture melengkapi pengujian UI penyedia. Fixture dari percobaan
live tetap tersimpan untuk reproduksi audit; tidak ada reset data lama.
Peringatan Vitest `transformIndexHtml` masih muncul tanpa menggagalkan tes.
Belum menguji hosting publik, perangkat fisik, atau integrasi mitra eksternal.

## Migrasi dan operasional

Jalankan `go run ./cmd/migrate up` sebelum API baru pada lingkungan lain.
Migrasi 11 menambahkan catatan dan aturan pembatalan setelah persetujuan.
Downgrade menolak penghapusan riwayat pembatalan baru; diperlukan backup yang
kompatibel untuk kembali ke skema lama. Database/API lokal telah diperbarui.

Rancangan: [workflow recovery design](superpowers/specs/2026-10-29-workflow-recovery-design.md).
