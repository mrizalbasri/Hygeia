# Hygeia — Business Requirements Document (BRD)

**Pertukaran data kesehatan lintas institusi, atas izin pasien**

| | |
|---|---|
| Versi | 0.2 (draft lomba) |
| Tanggal | Oktober 2026 |
| Lomba | UINIC 8.0 — Web Development Competition |
| Tema / subtema | *Engineering Interconnected Web Solutions to Empower Digital Ecosystems* / Kesehatan |
| Tahap | MVP / prototipe lomba |
| Data demo | Hanya data sintetis. Bukan untuk penggunaan klinis nyata. |

> Dokumen ini bagian pertama dari tiga: **BRD** (masalah dan tujuan bisnis) → **PRD** (perilaku produk) → **Arsitektur** (cara membangun). Kriteria penilaian lomba yang dipakai di sini masih asumsi; sesuaikan setelah guidebook resmi dibaca.

---

## 1. Ringkasan

Hygeia adalah platform web yang menghubungkan pasien, dokter/klinik, laboratorium, dan apotek dalam satu alur perawatan. Setiap data sensitif yang berpindah antar pihak harus:

1. **diizinkan pasien** (atau memenuhi hubungan perawatan yang sudah disetujui pasien),
2. **dibatasi** pada data yang dibutuhkan saja, dan
3. **tercatat** di audit log yang integritasnya bisa diverifikasi.

Hygeia **bukan** sistem manajemen rumah sakit, bukan AI diagnosis, dan bukan pengganti platform nasional. Nilai utamanya adalah lapisan **consent dan akuntabilitas** yang terlihat dan bisa dikontrol pasien.

## 2. Masalah Bisnis

Dalam satu episode perawatan, informasi dipegang beberapa pihak yang sistemnya terpisah. Akibatnya:

- Pasien tidak tahu siapa yang punya akses ke datanya, kapan, dan untuk apa.
- Dokter menunggu hasil lab tanpa alur yang jelas untuk menerimanya.
- Lab tidak punya jalur terstruktur untuk mengembalikan hasil.
- Apotek perlu isi resep, tetapi tidak boleh melihat riwayat pasien yang tidak terkait.
- Tidak ada riwayat akses yang dapat dipercaya ketika terjadi sengketa atau kebocoran.
- Situasi darurat (pasien tidak sadar) butuh akses cepat, tetapi akses cepat tanpa pencatatan rawan disalahgunakan.

## 3. Posisi terhadap Solusi yang Sudah Ada

| Pembanding | Fokus | Posisi Hygeia |
|---|---|---|
| SATUSEHAT (Kemenkes) | Integrasi data kesehatan nasional antar fasilitas | Hygeia tidak menggantikannya. Hygeia mendemonstrasikan lapisan consent dan audit yang berpusat pada pasien dan dapat berdiri di samping atau di depan pertukaran data. |
| Standar FHIR | Format pertukaran data | Di luar scope MVP. Model data Hygeia dirancang agar bisa dipetakan (hasil lab ke `Observation`, resep ke `MedicationRequest`). |
| Aplikasi booking / telemedisin | Akses ke layanan | Hygeia tidak mengurus pemesanan atau konsultasi. |
| Rekam medis klinik (EMR) | Pencatatan klinis internal satu fasilitas | Hygeia mengurus pertukaran antar fasilitas, bukan pencatatan internal. |

Landasan regulasi yang relevan (verifikasi pasal dan nomor sebelum dikutip di proposal): UU No. 27 Tahun 2022 tentang Pelindungan Data Pribadi, dan Permenkes No. 24 Tahun 2022 tentang Rekam Medis.

## 4. Tujuan Bisnis

| ID | Tujuan | Hasil yang Diharapkan |
|---|---|---|
| BO-01 | Menyatukan perjalanan pasien | Pasien melihat permintaan lab, hasil, resep, dan status dalam satu timeline. |
| BO-02 | Pertukaran data terkendali | Setiap akses data sensitif diizinkan oleh peran, hubungan perawatan, dan/atau consent pasien. |
| BO-03 | Akuntabilitas | Semua akses dan perubahan penting tercatat dalam audit log yang integritasnya dapat diverifikasi. |
| BO-04 | Interoperabilitas | Antarmuka pasien, dokter, lab, dan apotek memakai REST API terdokumentasi yang sama. |
| BO-05 | Akses darurat yang bertanggung jawab | Dokter dapat membuka data penting dalam keadaan darurat, dan pasien selalu diberi tahu. |
| BO-06 | MVP siap lomba | Satu skenario end-to-end dapat didemokan dengan andal dalam sesi juri singkat. |

## 5. Pemangku Kepentingan

| Pihak | Kebutuhan |
|---|---|
| Pasien | Melihat perjalanan, mengelola consent, melihat riwayat akses dalam bahasa yang mudah dipahami. |
| Dokter | Membuat permintaan lab, melihat hasil yang berhak dilihat, menerbitkan resep. |
| Staf lab | Menerima permintaan, memperbarui status, mengirim hasil terstruktur. |
| Staf apotek | Menerima resep yang ditujukan ke apoteknya dan memperbarui status penyiapan. |
| Admin platform | Mengelola organisasi dan akun tanpa otomatis bisa membaca data klinis. |
| Juri lomba | Menilai inovasi, relevansi, kualitas teknis, UI/UX, dokumentasi, dan kematangan implementasi. |

## 6. Scope

### 6.1 Termasuk dalam MVP

- Login dan dashboard berbasis peran (pasien, dokter, lab, apotek, admin minimal).
- Model organisasi: klinik, laboratorium, apotek.
- Care episode (hubungan pasien–dokter) dan timeline perjalanan pasien.
- Permintaan lab oleh dokter, pengiriman hasil terstruktur oleh lab.
- Resep oleh dokter yang ditujukan ke apotek pilihan pasien.
- Consent: dibuat, disetujui, ditolak, kedaluwarsa, dicabut. Consent lintas organisasi.
- Akses darurat (*break-glass*) dengan alasan wajib dan notifikasi ke pasien.
- Audit log berantai hash dengan fitur verifikasi integritas.
- Ringkasan akses untuk pasien dalam kalimat biasa.
- Notifikasi dalam aplikasi untuk perubahan status penting.
- Deployment demo yang bisa dibuka di browser.

### 6.2 Di luar MVP

- AI diagnosis, saran terapi, atau pengganti keputusan klinis.
- Integrasi nyata dengan SATUSEHAT, BPJS, asuransi, atau sistem rumah sakit.
- Pemakaian rekam medis pasien sungguhan.
- Pembayaran, klaim, ambulans, rawat inap, atau ERP rumah sakit.
- Microservices, message broker, Kubernetes.
- Upload dokumen PDF hasil lab (hasil dibuat terstruktur dulu).
- Berbagi sementara via QR (CarePass). Ditunda sampai alur inti stabil, lihat PRD FR-17.
- Pembaruan realtime (SSE/WebSocket), ditunda.

## 7. Kebutuhan Bisnis

| ID | Kebutuhan | Pernyataan |
|---|---|---|
| BR-01 | Identitas dan peran | Sistem membedakan kemampuan pasien, dokter, lab, apotek, dan admin. |
| BR-02 | Keanggotaan organisasi | Staf terhubung ke satu organisasi dan bekerja dalam konteks organisasi itu. |
| BR-03 | Perjalanan perawatan | Pasien dapat melihat urutan kronologis permintaan, hasil, resep, dan perubahan status. |
| BR-04 | Pertukaran lab | Dokter mengirim permintaan lab dan lab mengembalikan hasil lewat platform. |
| BR-05 | Pertukaran resep | Dokter menerbitkan resep ke apotek pilihan pasien dan apotek memperbarui status. |
| BR-06 | Consent | Pasien dapat menyetujui, menolak, mencabut, dan membatasi waktu akses ke data terlindungi. |
| BR-07 | Akses seperlunya | Setiap peran hanya menerima data yang dibutuhkan alurnya. |
| BR-08 | Auditabilitas | Sistem mencatat siapa mengakses atau mengubah apa dan kapan, dengan integritas yang dapat diverifikasi. |
| BR-09 | Akses darurat | Akses darurat tersedia untuk dokter dengan alasan wajib, batas waktu, cakupan terbatas, dan notifikasi ke pasien. |
| BR-10 | API interoperabel | Operasi bisnis inti tersedia lewat REST API versi `v1` yang terdokumentasi. |
| BR-11 | Transparansi untuk pasien | Riwayat akses ditampilkan dalam bahasa yang dapat dipahami orang awam. |
| BR-12 | Keandalan demo | MVP mendukung demo end-to-end yang deterministik dengan akun dan data sintetis. |

## 8. Aturan Bisnis

| ID | Aturan |
|---|---|
| BRULE-01 | Pasien hanya dapat mengakses profil dan perjalanan miliknya sendiri. |
| BRULE-02 | Tindakan staf dibatasi oleh peran dan keanggotaan organisasi. |
| BRULE-03 | Dokter hanya dapat melihat data pasien jika ada care episode aktif yang disetujui pasien, atau consent aktif untuk resource itu, atau akses darurat yang sah. Keberadaan pasien di sistem saja tidak cukup. |
| BRULE-04 | Dokter yang membuat permintaan lab dapat melihat hasilnya selama care episode-nya aktif, tanpa consent kedua. |
| BRULE-05 | Dokter di luar care episode memerlukan consent eksplisit dari pasien untuk resource yang diminta. |
| BRULE-06 | Lab hanya melihat data permintaan yang ditujukan kepadanya. Apotek hanya melihat resep yang ditujukan kepadanya. |
| BRULE-07 | Consent memiliki penerima, scope (jenis resource dan aksi), status, tujuan, dan waktu kedaluwarsa opsional. |
| BRULE-08 | Consent yang kedaluwarsa atau dicabut ditolak oleh lapisan otorisasi. |
| BRULE-09 | Akses darurat hanya untuk peran dokter, memerlukan kode dan teks alasan, berlaku singkat, hanya untuk resource yang ditetapkan, selalu memberi notifikasi ke pasien, dan ditandai di audit. |
| BRULE-10 | Event audit tidak dapat diubah atau dihapus oleh pengguna biasa, dan perubahan pada rantai audit harus terdeteksi saat verifikasi. |
| BRULE-11 | Admin tidak dapat membaca data klinis secara default. |
| BRULE-12 | Dataset demo hanya berisi data sintetis. |

## 9. Metrik Keberhasilan

| Metrik | Target MVP |
|---|---|
| Demo end-to-end | Skenario PRD bagian 8 selesai tanpa mengubah database secara manual. |
| Ketepatan otorisasi | Setiap kombinasi peran/resource yang tidak berhak menghasilkan `403` pada test otomatis, tanpa membocorkan isi data. |
| Cakupan audit | 100% pembacaan hasil lab, pembacaan resep, perubahan consent, dan akses darurat menghasilkan event audit. |
| Integritas audit | Verifikasi rantai lulus pada data normal dan gagal terdeteksi pada tes manipulasi. |
| Kemudahan pakai | Penguji dapat menyelesaikan alur utama perannya dengan sedikit bantuan. |
| Deployment | Demo publik dapat dibuka di browser dan repositori punya panduan setup. |

## 10. Risiko dan Mitigasi

| Risiko | Dampak | Mitigasi |
|---|---|---|
| Scope melebar menjadi sistem rumah sakit | Tinggi | Batasi pada satu care episode yang menghubungkan empat peran. |
| Topik data kesehatan memancing pertanyaan sulit | Tinggi | Pakai data sintetis, nyatakan Hygeia sebagai prototipe koordinasi, siapkan jawaban untuk SATUSEHAT dan UU PDP. |
| Alur consent membingungkan | Sedang | Consent per resource dengan layar sederhana: peminta, data, tujuan, durasi, setuju/tolak. |
| Terlalu banyak teknologi baru menurunkan kualitas | Sedang | Modular monolith. Tambahkan Redis, SSE, atau object storage hanya jika dibutuhkan. |
| Klaim keamanan dinilai berlebihan | Sedang | Jelaskan batas: hash chain membuat manipulasi **terdeteksi**, bukan mustahil. Jangan klaim "tidak bisa diretas". |
| Demo bergantung pada integrasi eksternal | Tinggi | Simulasikan organisasi di dalam Hygeia dan sediakan API yang siap diintegrasikan. |
| Akses darurat disalahgunakan | Sedang | Alasan wajib, batas waktu, cakupan terbatas, notifikasi pasien, penandaan di audit, dan tinjauan admin. |
| Pergantian stack memperlambat tim | Sedang | Tentukan stack sebelum coding. Lihat Arsitektur bagian 14. |

## 11. Asumsi

- Guidebook resmi belum dibaca saat dokumen ini dibuat. Aturan teknis (deploy wajib, video, batas halaman, kriteria dan bobot) perlu dicocokkan.
- Panitia sudah menyatakan karya lama boleh didaftarkan selama belum pernah menang. Jika Hygeia dikembangkan dari karya sebelumnya, pastikan itu sesuai jawaban panitia.
- Tim dapat mengerjakan semua peran (frontend, backend, database, deployment).
