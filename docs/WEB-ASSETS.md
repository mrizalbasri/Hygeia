# Aset visual web Hygeia

Semua ilustrasi memakai palet biru, cyan, navy, dan putih dari design system. Gambar merupakan ilustrasi buatan AI; tidak menggambarkan pasien, tenaga kesehatan, kemasan obat, atau fasilitas nyata.

| Aset WebP di frontend/static | Penempatan |
| --- | --- |
| patient-care.webp | Pilihan Pasien, panel pendaftaran |
| care-consultation.webp | Pilihan Dokter, Tentang, panel login |
| laboratory-supplies.webp | Pilihan Laboratorium |
| pharmacy-supplies.webp | Pilihan Apotek |
| connected-care.webp | Pengantar empat peran |

Versi PNG disimpan sebagai sumber. Halaman memakai WebP dengan dimensi intrinsik 1536 × 1024, teks alternatif untuk gambar informatif, serta lazy loading untuk gambar di bawah area awal. Panel sambutan login dan pendaftaran mengikuti aturan mobile yang menyembunyikan ilustrasi besar agar form mudah dicapai. Logo dan mockup ponsel yang sudah ada tetap dipakai.

Prompt dan asal masing-masing gambar ada pada dokumen `ASSET-*.md`. Untuk mengganti ilustrasi, simpan aset baru di `frontend/static`, lalu perbarui referensi gambar dan teks alternatif yang sesuai. Empat peran mengambil gambar dari metadata `audiences` pada `frontend/src/routes/+page.svelte`.
