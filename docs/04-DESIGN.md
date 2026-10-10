# Hygeia — Design System

| | |
|---|---|
| Versi | 0.1 |
| Sumber | Logo Hygeia (`assets/logo.png`) |
| Dokumen terkait | `01-BRD.md`, `02-PRD.md`, `03-ARCHITECTURE.md` |
| Platform | Web, desktop-first untuk dashboard staf; tetap nyaman di layar kecil untuk pasien |

Warna di dokumen ini diambil langsung dari piksel logo. Rasio kontras dihitung (WCAG 2.x), bukan diperkirakan.

---

## 1. Dasar Identitas

### 1.1 Arti logo

Logo berupa huruf **H** dari dua batang putih bulat yang dihubungkan oleh pita gradien biru–cyan dengan tanda plus di tengahnya, di atas ubin gelap berujung bulat.

- **Dua batang** = dua pihak (misalnya pasien dan penyedia layanan).
- **Pita** = data yang mengalir dan menghubungkan keduanya.
- **Plus** = kesehatan.

Ini sejalan dengan produknya: data kesehatan terhubung antar pihak. Pakai cerita ini di presentasi dan jadikan pita sebagai motif visual utama di UI (lihat bagian 6.3).

### 1.2 Kepribadian visual

Tenang, bersih, dapat dipercaya. Produk ini menangani data sensitif, jadi tampilannya harus terasa jelas dan terkendali, bukan ramai. Aksen gradien dipakai hemat; sebagian besar layar netral.

### 1.3 Prinsip desain

1. **Jelas sebelum cantik.** Layar consent harus bisa dipahami dalam beberapa detik.
2. **Satu aksen gradien per layar.** Gradien pita adalah tanda tangan, bukan hiasan.
3. **Status tidak hanya lewat warna.** Selalu pasangkan ikon dan teks.
4. **Akses darurat harus terlihat berbeda** dari semua status consent biasa.
5. **Padat untuk staf, lega untuk pasien.** Staf bekerja dengan tabel; pasien membaca kalimat.

## 2. Penggunaan Logo

| Aturan | Keterangan |
|---|---|
| Selalu dengan ubin gelap | Batang H berwarna putih, jadi hilang di latar terang. Di latar terang, tetap pakai ubin gelapnya. |
| Ukuran minimum | 24 px untuk logo penuh. Di bawah 32 px, detail plus dan gradien pita hilang, jadi pakai versi favicon yang disederhanakan (H + pita, tanpa plus). |
| Ruang bebas | Minimal selebar satu batang H di semua sisi ubin. |
| Dilarang | Mengubah warna pita, meregangkan, memutar, menambah bayangan atau outline, menaruh di atas gambar ramai. |
| Teks nama | "Hygeia" ditulis dengan huruf H besar, tanpa singkatan. |

**Catatan berkas:** `assets/logo.png` berukuran 1254 × 1254 dengan latar hitam di luar ubin, jadi bukan transparan. Untuk web, siapkan versi PNG/SVG dengan latar transparan di luar ubin, ukuran favicon (16, 32, 180, 512), dan versi tanpa ubin untuk kebutuhan khusus. Jika ada berkas sumber (Figma, AI, atau SVG), pakai itu.

## 3. Warna

### 3.1 Dari logo (sampel piksel)

| Peran | Hex | Asal |
|---|---|---|
| Ink (latar ubin) | `#0B101C` | Rata-rata area gelap ubin (`#0F1421` ke `#070D1A`) |
| Tepi ubin | `#2B3345` | Tepi atas-kiri ubin |
| Putih batang | `#F7F9FE` | Batang H bagian atas |
| Bayangan batang | `#D9E6FC` | Batang H bagian bawah |
| Biru dalam | `#034BFD` | Ujung kanan pita |
| Biru | `#146AFE` | Ujung kiri pita |
| Azure | `#0190F9` | Pita, dekat batang kiri |
| Sky | `#00ABEF` | Pita, bagian tengah |
| Cyan | `#1AEBFD` | Pita, bagian terang |

### 3.2 Gradien pita

```css
--ribbon: linear-gradient(90deg, #146AFE 0%, #00ABEF 50%, #1AEBFD 100%);
```

Pakai untuk: garis timeline Perjalanan Saya, bar kemajuan, animasi hasil verifikasi integritas, dan aksen halaman login. Jangan dipakai untuk tombol biasa atau latar kartu.

### 3.3 Token semantik

**Terang (default untuk dashboard staf)**

| Token | Nilai | Catatan |
|---|---|---|
| `--bg` | `#F4F7FC` | Latar halaman |
| `--surface` | `#FFFFFF` | Kartu dan tabel |
| `--border` | `#DCE3F0` | Garis tipis |
| `--text` | `#0B101C` | 19.0 : 1 di atas putih |
| `--text-muted` | `#5B6479` | 5.9 : 1 di atas putih, 5.5 : 1 di atas `--bg` |
| `--primary` | `#034BFD` | Teks putih di atasnya 6.1 : 1 |
| `--primary-hover` | `#0A3FD1` | Teks putih 7.9 : 1 |
| `--focus` | `#146AFE` | Cincin fokus 2 px, offset 2 px |

**Gelap (bawaan logo; untuk login dan pilihan pengguna)**

| Token | Nilai | Catatan |
|---|---|---|
| `--bg` | `#0B101C` | Latar halaman |
| `--surface` | `#111827` | Kartu |
| `--surface-raised` | `#161D2E` | Kartu yang menonjol, menu |
| `--border` | `#2B3345` | Dari tepi ubin |
| `--text` | `#F7F9FE` | 18.0 : 1 di atas `--bg` |
| `--text-muted` | `#9AA6BD` | 7.8 : 1 di atas `--bg` |
| `--primary` | `#146AFE` | Teks putih di atasnya 4.6 : 1; untuk tombol besar atau teks tebal |
| `--accent-text` | `#6FA3FF` | Tautan di latar gelap, 7.6 : 1 |
| `--highlight` | `#1AEBFD` | Aksen kecil, 13.0 : 1 |
| `--focus` | `#1AEBFD` | Cincin fokus |

Catatan: `#146AFE` langsung di atas latar gelap hanya 4.1 : 1, jadi **jangan dipakai sebagai warna teks kecil di mode gelap**; pakai `--accent-text`.

### 3.4 Warna status

Semua status memakai **ikon + teks + warna**. Nilai kontras dihitung terhadap latar yang dipasangkan.

| Status | Teks (terang) | Latar (terang) | Teks (gelap) | Ikon |
|---|---|---|---|---|
| Aktif / Berhasil | `#0B6B3A` (6.0 : 1) | `#E8F7EE` | `#4ADE80` (10.9 : 1) | centang |
| Menunggu | `#8A5300` (5.9 : 1) | `#FFF6E5` | `#FBBF24` (11.4 : 1) | jam |
| Ditolak / Error | `#B42318` (6.1 : 1) | `#FEF3F2` | `#F87171` (6.9 : 1) | silang |
| Dicabut | `#5B6479` | `#EEF1F7` | `#9AA6BD` | lingkaran dicoret |
| Kedaluwarsa | `#5B6479` | `#EEF1F7` | `#9AA6BD` | jam pasir |
| **Akses darurat** | `#B42318` | `#FEF3F2` dengan garis tepi kiri 4 px `#B42318` | `#F87171` | segitiga peringatan |

Akses darurat sengaja berbeda bentuk (garis tepi tebal dan segitiga), bukan hanya warna, supaya tidak tertukar dengan status "Ditolak".

### 3.5 Label dasar akses (jalur A–D)

Dipakai di Riwayat Akses (PRD bagian 4).

| Jalur | Label | Gaya |
|---|---|---|
| A | Episode perawatan | Chip netral |
| B | Disetujui Anda | Chip biru muda |
| C | Izin Anda | Chip biru muda, dengan sisa waktu |
| D | Darurat | Gaya akses darurat |

## 4. Tipografi

| Peran | Font | Alasan |
|---|---|---|
| UI dan judul | **Plus Jakarta Sans** | Bersih, bulat, selaras dengan bentuk logo; mendukung Latin dengan baik. |
| Data teknis (hash, ID, kode) | **JetBrains Mono** | Hash audit dan ID perlu mudah dibaca karakter demi karakter. |
| Cadangan | `Inter`, `system-ui`, `sans-serif` | |

Keduanya tersedia di Google Fonts. Muat hanya bobot yang dipakai (400, 500, 600, 700) dan atur `font-display: swap`.

| Gaya | Ukuran / tinggi baris | Bobot | Dipakai untuk |
|---|---|---|---|
| Display | 36 / 44 | 700 | Judul halaman login, kondisi kosong besar |
| H1 | 28 / 36 | 700 | Judul halaman |
| H2 | 22 / 30 | 600 | Judul bagian |
| H3 | 18 / 26 | 600 | Judul kartu |
| Body | 16 / 24 | 400 | Teks utama pasien |
| Body kecil | 14 / 20 | 400 | Tabel staf, keterangan |
| Label | 13 / 18 | 600 | Label form, chip |
| Mono | 13 / 20 | 500 | Hash, ID, kode |

Teks pasien minimal 16 px. Tabel staf boleh 14 px.

## 5. Bentuk, Jarak, dan Elevasi

- **Satuan jarak:** kelipatan 4 px (4, 8, 12, 16, 24, 32, 48, 64).
- **Radius:** mengikuti logo yang bulat. `sm 8` (input kecil, chip), `md 12` (tombol, input), `lg 16` (kartu), `xl 24` (modal, panel besar), `full` (badge pil).
- **Elevasi:** hemat. Kartu memakai garis tepi 1 px dan bayangan sangat tipis; hanya menu dan modal yang benar-benar terangkat.
  - Kartu: `0 1px 2px rgba(11,16,28,0.06)`
  - Menu/modal: `0 8px 24px rgba(11,16,28,0.16)`
- **Grid:** kontainer maksimum 1280 px; sidebar 240 px; konten maksimum 1120 px.
- **Breakpoint:** `sm 640`, `md 768`, `lg 1024`, `xl 1280`.

## 6. Komponen

### 6.1 Tombol

| Varian | Dipakai untuk |
|---|---|
| Primary (`--primary`, teks putih) | Aksi utama satu per layar: "Izinkan", "Kirim hasil" |
| Secondary (garis tepi, latar transparan) | Aksi pendukung: "Lihat detail" |
| Ghost | Aksi ringan: "Batal" |
| Danger (merah) | "Tolak", "Cabut izin", "Akhiri episode" |

Tinggi 40 px (44 px di layar sentuh), radius 12, label 14–15 px bobot 600. Status disabled memakai opasitas 50% **dan** `aria-disabled`. Cincin fokus selalu terlihat.

### 6.2 Kartu Consent (layar terpenting)

Tata letak, dari atas ke bawah:

1. **Siapa** meminta (nama, peran, organisasi) dengan avatar inisial.
2. **Data apa** (misalnya "Hasil Gula Darah, 10 Okt 2026").
3. **Untuk apa** (teks tujuan dari peminta).
4. **Berapa lama**: pemilih durasi (1 jam, 24 jam, 7 hari) dengan ringkasan waktu berakhir ("berlaku sampai 11 Okt, 15:00").
5. Dua tombol: **Izinkan** (primary) dan **Tolak** (secondary/danger), sejajar dan berukuran sama.

Aturan: tidak ada kata teknis, tidak ada pilihan default yang paling longgar (default durasi paling pendek), dan setelah keputusan tampilkan konfirmasi yang menyebut apa yang baru saja terjadi dan cara mencabutnya.

### 6.3 Timeline Perjalanan (motif pita)

Garis vertikal tipis dengan gradien pita menghubungkan kejadian (permintaan lab, hasil, resep, perubahan status). Tiap kejadian berupa titik berikon plus kartu kecil berisi satu kalimat dan waktu. Kejadian baru "mengalir" masuk dengan animasi garis singkat. Tanpa gradien pada kartu itu sendiri.

### 6.4 Entri Riwayat Akses

Satu kalimat biasa per kejadian (template di Arsitektur 8.6), dengan chip dasar akses (bagian 3.5) dan waktu. Kejadian ditolak ditampilkan dengan ikon status "Ditolak". Kejadian darurat memakai gaya akses darurat dan selalu di bagian yang mudah terlihat.

### 6.5 Panel Verifikasi Integritas

- Tombol **Verifikasi integritas**.
- Hasil utuh: status hijau dengan ikon centang, jumlah event, dan animasi pita satu kali di garis atas panel.
- Hasil tidak utuh: status merah dengan ikon silang, teks "Event nomor N tidak cocok", dan tautan ke event itu.
- `last_hash` ditampilkan dalam font mono, dipotong di tengah, dengan tombol salin.
- Teks pendamping jujur: "Verifikasi ini mendeteksi perubahan pada catatan. Ini bukan jaminan bahwa catatan tidak mungkin diubah."

### 6.6 Banner Akses Darurat

Ditampilkan di layar dokter selama sesi darurat aktif: garis tepi kiri merah 4 px, ikon peringatan, teks "Akses darurat aktif sampai 16:45", sisa waktu berjalan, dan tombol "Akhiri sekarang". Di sisi pasien muncul sebagai notifikasi berprioritas tinggi.

Dialog mulai akses darurat wajib memuat pemilih kode alasan, kolom alasan teks, dan ringkasan konsekuensi: "Pasien akan diberi tahu dan akses ini akan ditinjau."

### 6.7 Tabel Staf

Padat dan nyaman dipindai: baris 44 px, kepala tabel `--text-muted` bobot 600, status memakai badge (bagian 3.4), filter status di atas, paginasi. Kolom referensi pasien memakai mono. Tabel punya urutan fokus keyboard dan label ARIA.

### 6.8 Form, Notifikasi, Kondisi Kosong

- **Form:** label di atas input, teks bantu di bawah, pesan galat spesifik dan berdekatan dengan kolomnya.
- **Notifikasi dalam aplikasi:** ikon + judul + satu kalimat + waktu; yang butuh tindakan (permintaan persetujuan) punya tombol langsung.
- **Kondisi kosong:** satu kalimat menjelaskan dan satu tindakan, tanpa ilustrasi berlebihan.

### 6.9 Navigasi

- **Staf (dokter, lab, apotek, admin):** sidebar kiri 240 px, ikon + label, bagian aktif ditandai garis kiri `--primary`.
- **Pasien:** sidebar di desktop, tab bawah di layar kecil. Menu: Perjalanan Saya, Persetujuan, Izin, Riwayat Akses.
- Peran ditunjukkan dengan chip netral di kepala halaman, bukan warna per peran.

## 7. Ikon dan Ilustrasi

- Pakai satu keluarga ikon garis, misalnya **Lucide**, dengan tebal garis 1.75 px dan ujung bulat agar selaras dengan logo.
- Tanda plus di logo adalah tanda merek. Untuk aksi "tambah" biasa, pakai ikon plus dari keluarga ikon dengan warna netral, jangan meniru plus biru logo.
- Ilustrasi minimal. Bila perlu, gunakan bentuk geometris sederhana dengan palet 3.1.

## 8. Gerak

| Elemen | Durasi | Easing |
|---|---|---|
| Hover, fokus, perubahan status | 120–150 ms | ease-out |
| Panel, modal, menu | 180–220 ms | ease-out |
| Animasi pita (timeline, verifikasi) | 400–600 ms, sekali | ease-in-out |

Hormati `prefers-reduced-motion`: matikan animasi pita dan ganti dengan perubahan status langsung.

## 9. Aksesibilitas

- Kontras teks minimal 4.5 : 1 (angka di dokumen ini sudah memenuhinya, kecuali dicatat di 3.3).
- Cincin fokus 2 px dengan offset 2 px pada semua elemen interaktif.
- Target sentuh minimal 40 × 40 px.
- Status tidak pernah hanya warna.
- Semua form memakai label yang terhubung; galat diumumkan lewat `aria-live`.
- Layar consent dan akses darurat diuji dengan keyboard saja.
- Bahasa halaman diatur `lang="id"`.

## 10. Suara dan Teks

Tenang, jelas, tidak berlebihan. Hindari bahasa promosi dan istilah teknis di layar pasien.

| Situasi | Teks |
|---|---|
| Tombol consent | Izinkan · Tolak · Cabut izin |
| Konfirmasi izin | "Dr. Budi bisa melihat hasil Gula Darah sampai 11 Okt, 15:00. Anda bisa mencabutnya kapan saja." |
| Akses ditolak (403) | "Anda tidak memiliki akses ke data ini." (tidak menyebut apakah datanya ada) |
| Permintaan lab menunggu | "Dr. Maya ingin mengirim permintaan Gula Darah ke Lab Sentra. Izinkan?" |
| Notifikasi akses darurat | "Dr. Rina membuka data Anda lewat akses darurat pada 16:45. Alasan: pasien tidak sadar." |
| Verifikasi utuh | "Catatan akses Anda utuh. 128 kejadian diperiksa." |
| Verifikasi tidak utuh | "Ada ketidakcocokan pada kejadian nomor 57." |

## 11. Implementasi

### 11.1 Token CSS

```css
:root {
  --bg: #F4F7FC;  --surface: #FFFFFF;  --border: #DCE3F0;
  --text: #0B101C;  --text-muted: #5B6479;
  --primary: #034BFD;  --primary-hover: #0A3FD1;  --focus: #146AFE;

  --ok-fg: #0B6B3A;    --ok-bg: #E8F7EE;
  --wait-fg: #8A5300;  --wait-bg: #FFF6E5;
  --bad-fg: #B42318;   --bad-bg: #FEF3F2;
  --off-fg: #5B6479;   --off-bg: #EEF1F7;

  --ribbon: linear-gradient(90deg, #146AFE 0%, #00ABEF 50%, #1AEBFD 100%);
  --radius-sm: 8px; --radius-md: 12px; --radius-lg: 16px; --radius-xl: 24px;
}

/* Gelap: mengikuti sistem, kecuali pengguna memaksa terang */
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
    --bg: #0B101C;  --surface: #111827;  --border: #2B3345;
    --text: #F7F9FE;  --text-muted: #9AA6BD;
    --primary: #146AFE;  --primary-hover: #034BFD;  --focus: #1AEBFD;

    --ok-fg: #4ADE80;    --ok-bg: rgba(74,222,128,.12);
    --wait-fg: #FBBF24;  --wait-bg: rgba(251,191,36,.12);
    --bad-fg: #F87171;   --bad-bg: rgba(248,113,113,.12);
    --off-fg: #9AA6BD;   --off-bg: rgba(154,166,189,.12);
  }
}

/* Gelap: dipaksa pengguna (nilai sama dengan blok di atas) */
:root[data-theme="dark"] {
  --bg: #0B101C;  --surface: #111827;  --border: #2B3345;
  --text: #F7F9FE;  --text-muted: #9AA6BD;
  --primary: #146AFE;  --primary-hover: #034BFD;  --focus: #1AEBFD;

  --ok-fg: #4ADE80;    --ok-bg: rgba(74,222,128,.12);
  --wait-fg: #FBBF24;  --wait-bg: rgba(251,191,36,.12);
  --bad-fg: #F87171;   --bad-bg: rgba(248,113,113,.12);
  --off-fg: #9AA6BD;   --off-bg: rgba(154,166,189,.12);
}
```

### 11.2 Tailwind

```js
// tailwind.config.js (potongan)
export default {
  darkMode: ['class', '[data-theme="dark"]'],
  theme: {
    extend: {
      colors: {
        bg: 'var(--bg)', surface: 'var(--surface)', border: 'var(--border)',
        text: 'var(--text)', muted: 'var(--text-muted)',
        primary: { DEFAULT: 'var(--primary)', hover: 'var(--primary-hover)' },
        ok:   { fg: 'var(--ok-fg)',   bg: 'var(--ok-bg)' },
        wait: { fg: 'var(--wait-fg)', bg: 'var(--wait-bg)' },
        bad:  { fg: 'var(--bad-fg)',  bg: 'var(--bad-bg)' },
        off:  { fg: 'var(--off-fg)',  bg: 'var(--off-bg)' },
      },
      borderRadius: { sm: '8px', md: '12px', lg: '16px', xl: '24px' },
      fontFamily: {
        sans: ['"Plus Jakarta Sans"', 'Inter', 'system-ui', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'ui-monospace', 'monospace'],
      },
      backgroundImage: {
        ribbon: 'linear-gradient(90deg, #146AFE 0%, #00ABEF 50%, #1AEBFD 100%)',
      },
    },
  },
};
```

## 12. Daftar Periksa Desain

- [ ] Logo transparan, favicon, dan versi kecil sudah dibuat.
- [ ] Font dimuat dengan bobot seperlunya.
- [ ] Layar login, consent, Perjalanan Saya, Riwayat Akses, panel verifikasi, dan banner akses darurat sudah punya wireframe.
- [ ] Tidak ada layar dengan lebih dari satu elemen gradien.
- [ ] Semua status punya ikon dan teks.
- [ ] Mode terang dan gelap diuji untuk kontras.
- [ ] Alur demo (PRD bagian 8) diuji dengan keyboard dan di layar kecil.
