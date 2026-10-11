# Hygeia — Product Requirements Document (PRD)

| | |
|---|---|
| Versi | 0.2 (draft lomba) |
| Dokumen terkait | `01-BRD.md` (tujuan dan aturan bisnis), `03-ARCHITECTURE.md` (cara membangun) |
| Data demo | Hanya data sintetis |

---

## 1. Visi Produk

Hygeia memungkinkan pasien, dokter, laboratorium, dan apotek menyelesaikan satu perjalanan perawatan yang terhubung, dengan setiap pertukaran data sensitif yang **diizinkan, dibatasi, dan tercatat**.

## 2. Prinsip Produk

- **Pasien memegang kendali.** Berbagi data mudah dipahami dan bisa dicabut.
- **Akses seperlunya.** Setiap peran hanya melihat yang dibutuhkan.
- **Bisa dibuktikan.** Akses penting meninggalkan jejak yang integritasnya dapat diverifikasi.
- **API dulu.** Frontend hanyalah klien dari API yang sama dengan yang nanti bisa dipakai sistem eksternal.
- **Satu alur yang mulus lebih baik daripada banyak fitur setengah jadi.**

## 3. Persona

| Persona | Tujuan | Masalah | Layar utama |
|---|---|---|---|
| Pasien (Aulia) | Menjalani perawatan dan mengendalikan akses data | Tidak tahu siapa yang bisa melihat datanya | Perjalanan Saya, Izin, Riwayat Akses |
| Dokter (Dr. Maya) | Meminta tes dan melanjutkan perawatan dengan hasil yang sah | Serah terima informasi terputus | Pasien, Permintaan Lab, Hasil, Resep |
| Staf lab | Memproses permintaan dan mengembalikan hasil | Permintaan dan hasil tidak terhubung | Permintaan Masuk, Input Hasil |
| Staf apotek | Menyiapkan resep yang ditujukan ke apoteknya | Butuh detail resep tanpa data tak terkait | Antrean Resep |
| Dokter lain (Dr. Budi, Dr. Rina) | Mengakses data pasien di luar episode-nya dengan cara yang sah | Tidak ada jalur resmi yang tercatat | Minta Akses, Akses Darurat |
| Admin | Mengelola organisasi dan akun | Perlu kontrol tanpa menjadi "super-viewer" data pasien | Organisasi, Akun, Status Audit, Tinjauan Akses Darurat |

## 4. Model Akses (inti produk)

Akses ke data terlindungi hanya terjadi lewat salah satu dari empat jalur. Jika tidak ada yang berlaku, hasilnya `403`.

| Jalur | Pemicu | Penerima | Cakupan | Durasi |
|---|---|---|---|---|
| **A. Care episode** | Dokter mengundang pasien, pasien menerima | Dokter pengundang dalam klinik yang tercatat pada episode | Membuat permintaan lab dan resep, melihat hasil dari episode itu | Selama episode aktif |
| **B. Berbagi ke penyedia** | Dokter membuat permintaan lab atau resep, pasien menyetujui satu kali | Lab atau apotek tujuan | Hanya permintaan atau resep itu | Selama permintaan/resep itu aktif |
| **C. Permintaan consent lintas organisasi** | Dokter lain meminta akses ke resource tertentu | Dokter peminta | Satu resource atau satu jenis resource, aksi `view` | Dipilih pasien (misal 1 jam, 24 jam, 7 hari) |
| **D. Akses darurat** | Dokter menekan "Akses Darurat" dengan alasan | Dokter tersebut | Hasil lab dan resep aktif saja | Singkat (default 60 menit) |

Catatan desain:
- Keputusan MVP: undangan memakai kode pasien yang cocok persis. Dokter lain,
  termasuk rekan satu klinik, tidak otomatis mendapat akses dari episode tersebut.
  Detail undangan dan riwayat episode dapat dibaca pasien dan dokter pengundang;
  akses data klinis jalur A tetap memerlukan episode aktif.
- Dokter yang meminta tes **tidak** perlu consent kedua untuk membaca hasilnya sendiri (jalur A). Consent tambahan hanya dibutuhkan ketika data berpindah ke pihak yang belum punya hubungan perawatan (jalur B dan C).
- Setiap keputusan akses, baik diizinkan maupun ditolak, menghasilkan event audit.

## 5. Perjalanan Inti

1. Dokter mengundang pasien ke care episode; pasien menerima (jalur A).
2. Dokter membuat permintaan lab ke laboratorium; pasien menyetujui pengiriman ke lab itu (jalur B).
3. Lab menerima, memproses, dan mengirim hasil terstruktur.
4. Dokter membuka hasil; sistem memeriksa episode aktif lalu mencatat audit.
5. Dokter membuat resep; pasien memilih apotek (jalur B).
6. Apotek menerima resep dan memperbarui status sampai selesai.
7. Dokter lain meminta akses ke hasil lab; pasien menyetujui dengan durasi (jalur C).
8. Pada situasi darurat, dokter membuka data lewat akses darurat (jalur D); pasien langsung diberi notifikasi.
9. Pasien melihat semuanya di Perjalanan Saya dan Riwayat Akses, dan dapat memverifikasi integritas audit.

## 6. Kebutuhan Fungsional

| ID | Fitur | Kebutuhan | Prioritas |
|---|---|---|---|
| FR-01 | Autentikasi | Pengguna dapat login dan logout; rute terlindungi menolak permintaan tanpa autentikasi. | Must |
| FR-02 | Dashboard per peran | Dashboard dan navigasi berubah sesuai peran. | Must |
| FR-03 | Organisasi | Staf klinik, lab, dan apotek terhubung ke satu organisasi. | Must |
| FR-04 | Care episode | Dokter mengundang pasien; pasien menerima atau menolak; episode dapat diakhiri oleh pasien atau dokter. | Must |
| FR-05 | Permintaan lab | Dokter membuat permintaan lab ke laboratorium pilihan dalam episode aktif. | Must |
| FR-06 | Alur lab | Lab dapat menerima, memperbarui status, dan mengirim hasil terstruktur (nama tes, nilai, satuan, rentang rujukan, penanda, catatan). | Must |
| FR-07 | Persetujuan berbagi ke penyedia | Pasien menyetujui atau menolak pengiriman permintaan lab ke lab tertentu dan resep ke apotek tertentu; pasien memilih apotek. | Must |
| FR-08 | Permintaan consent lintas organisasi | Dokter di luar episode dapat meminta akses ke resource tertentu dengan tujuan tertulis. | Must |
| FR-09 | Manajemen consent | Pasien dapat menyetujui (dengan memilih durasi), menolak, mencabut, dan melihat scope dan kedaluwarsa. | Must |
| FR-10 | Hasil terlindungi | Hasil lab hanya dikembalikan jika salah satu jalur akses berlaku; selain itu `403` tanpa kebocoran isi. | Must |
| FR-11 | Resep | Dokter membuat resep (item obat, dosis, aturan pakai) dan pasien memilih apotek tujuan. | Must |
| FR-12 | Alur apotek | Apotek memindahkan resep: diterima, disiapkan, siap, selesai. | Must |
| FR-13 | Perjalanan pasien | Pasien melihat timeline kronologis semua kejadian penting. | Must |
| FR-14 | Audit log berantai hash | Event audit ditulis append-only dengan hash terkait event sebelumnya; tersedia fitur verifikasi integritas. | Must |
| FR-15 | Ringkasan akses | Riwayat akses pasien ditampilkan sebagai kalimat biasa (siapa, apa, kapan, atas dasar apa, sampai kapan). | Must |
| FR-16 | Akses darurat | Dokter dapat membuka hasil lab dan resep aktif pasien dengan kode alasan dan teks; akses berbatas waktu; pasien dinotifikasi; event ditandai di audit dan muncul di tinjauan admin. | Should |
| FR-17 | CarePass (QR) | Pasien membuat tautan/QR berbatas waktu untuk satu resource. | Could |
| FR-18 | Notifikasi | Notifikasi dalam aplikasi untuk perubahan status dan permintaan persetujuan. | Should |
| FR-19 | Pencarian dan filter | Staf dapat memfilter permintaan dan resep berdasarkan status dan referensi pasien. | Should |
| FR-20 | Pembaruan realtime | SSE atau WebSocket untuk notifikasi, jika waktu tersisa. | Could |
| FR-21 | Admin minimal | Admin mengelola organisasi dan akun, melihat status verifikasi rantai audit, dan meninjau event akses darurat; admin tidak dapat membaca isi hasil lab atau resep. | Must |

## 7. Kriteria Penerimaan

| Fitur | Kriteria |
|---|---|
| Care episode | Pasien yang menerima undangan terlihat di daftar pasien dokter; pasien yang menolak tidak muncul. Episode yang diakhiri menutup akses jalur A. |
| Berbagi ke lab | Permintaan lab tidak terlihat oleh lab sebelum pasien menyetujui. Setelah disetujui, lab hanya melihat jenis tes dan referensi pasien yang diperlukan. Jika ditolak, permintaan dibatalkan dan tercatat di timeline. |
| Hasil lab | Dokter pembuat permintaan dapat membuka hasil selama episode aktif. Dokter lain tanpa consent menerima `403` dan respons tidak memuat nilai hasil. |
| Consent | Consent memuat pasien, penerima, scope, status, waktu dibuat, tujuan, dan kedaluwarsa. Status `denied`, `expired`, dan `revoked` memblokir aksi. Pencabutan berlaku pada permintaan berikutnya. |
| Resep | Resep hanya terlihat oleh dokter pembuat, pasien, dan apotek yang dipilih pasien, setelah pasien memilih apotek. |
| Audit | Setiap akses terlindungi mencatat pelaku, aksi, resource, hasil (diizinkan/ditolak), dasar akses (jalur dan id consent bila ada), dan waktu. |
| Verifikasi audit | Tombol verifikasi mengembalikan "utuh" pada data normal dan menunjuk nomor event pertama yang tidak cocok setelah satu baris diubah langsung di database. |
| Akses darurat | Tanpa alasan, permintaan ditolak. Dengan alasan, akses berlaku sesuai batas waktu, pasien menerima notifikasi segera, event ditandai `break_glass`, dan akses berakhir otomatis. |
| Ringkasan akses | Setiap event akses terlindungi dapat dirender menjadi satu kalimat, misalnya "Dr. Maya melihat hasil Gula Darah, 14:02, karena episode perawatan aktif." |

## 8. Skenario Demo

Semua akun sintetis.

| Peran | Akun |
|---|---|
| Pasien | Aulia Rahman |
| Dokter episode | Dr. Maya, Klinik Sentra |
| Laboratorium | Lab Sentra |
| Apotek | Apotek Sentra |
| Dokter lintas organisasi | Dr. Budi, Klinik Harapan |
| Dokter IGD | Dr. Rina, IGD Harapan |

Langkah:

1. Dr. Maya mengundang Aulia ke care episode; Aulia menerima.
2. Dr. Maya membuat permintaan lab Gula Darah ke Lab Sentra; Aulia menyetujui pengiriman.
3. Lab Sentra menerima dan mengirim hasil terstruktur.
4. Dr. Maya membuka hasil (tanpa consent tambahan karena episode aktif).
5. Dr. Maya membuat resep; Aulia memilih Apotek Sentra.
6. Apotek Sentra memindahkan resep ke "Siap".
7. Dr. Budi mencoba membuka hasil dan mendapat `403`; ia mengirim permintaan akses; Aulia menyetujui 24 jam; Dr. Budi membuka hasil.
8. Aulia mencabut consent Dr. Budi; Dr. Budi mencoba lagi dan ditolak.
9. Dr. Rina menggunakan akses darurat dengan alasan "pasien tidak sadar"; Aulia menerima notifikasi.
10. Aulia membuka Perjalanan Saya dan Riwayat Akses (kalimat biasa), lalu menekan "Verifikasi Integritas" dan hasilnya utuh.
11. Penutup: satu baris audit diubah langsung di database, verifikasi diulang dan menunjuk event yang tidak cocok. Langkah ini hanya untuk demo dan dilakukan dengan skrip terpisah.

Perkiraan durasi: 6–8 menit. Siapkan versi 3 menit (langkah 1–5, 7, 10) untuk batas waktu yang ketat.

## 9. Kebutuhan Non-Fungsional

| ID | Kategori | Kebutuhan |
|---|---|---|
| NFR-01 | Keamanan | HTTPS di deployment; password di-hash (Argon2id); cookie sesi `HttpOnly`, `Secure`, `SameSite`; proteksi CSRF untuk mutasi; otorisasi sisi server di setiap endpoint terlindungi. |
| NFR-02 | Privasi | Data demo sintetis; kumpulkan field sensitif seminimal mungkin. |
| NFR-03 | Kinerja | Dashboard dan API terasa responsif pada beban demo; daftar staf dipaginasi. |
| NFR-04 | Keandalan | Tulisan multi-langkah memakai transaksi database, termasuk penulisan event audit bersama aksi yang diauditnya. |
| NFR-05 | Maintainability | Backend modular monolith dengan batas domain yang jelas. |
| NFR-06 | Observability | Log terstruktur, request ID, log error, endpoint kesehatan. |
| NFR-07 | Aksesibilitas | Form dapat dipakai dengan keyboard, label jelas, kontras terbaca, pesan consent mudah dipahami. |
| NFR-08 | Portabilitas | Layanan dalam container; variabel lingkungan terdokumentasi. |
| NFR-09 | Desain web | UI dirancang desktop-first untuk dashboard staf dan tetap dapat dipakai di layar kecil untuk pasien. |

## 10. Inventaris Layar

| Peran | Layar |
|---|---|
| Umum | Login, Notifikasi |
| Pasien | Perjalanan Saya, Undangan Episode, Persetujuan Berbagi, Izin (daftar consent aktif, riwayat, cabut), Riwayat Akses (kalimat biasa + verifikasi integritas) |
| Dokter | Daftar Pasien, Detail Pasien/Episode, Permintaan Lab, Hasil, Resep, Minta Akses, Akses Darurat |
| Lab | Permintaan Masuk, Detail dan Input Hasil |
| Apotek | Antrean Resep, Detail Resep |
| Admin | Organisasi, Akun, Status Rantai Audit, Tinjauan Akses Darurat |

Layar consent harus sesederhana mungkin: **siapa** meminta, **data apa**, **untuk apa**, **berapa lama**, tombol **Izinkan** dan **Tolak**.

## 11. Prioritas dan Batas Pemotongan

Urutan yang dijaga jika waktu mepet:

1. **Wajib ada:** FR-01 sampai FR-15 dan FR-21 (alur inti, consent, audit, verifikasi).
2. **Lalu:** FR-16 (akses darurat), FR-18 (notifikasi), FR-19 (filter).
3. **Hanya jika sisa waktu:** FR-17 (CarePass) dan FR-20 (realtime).

Jika harus memotong, potong dari bawah. Jangan potong audit berantai hash atau akses darurat demi fitur tambahan; keduanya pembeda utama Hygeia.

## 12. Di Luar Scope

Lihat BRD bagian 6.2.
