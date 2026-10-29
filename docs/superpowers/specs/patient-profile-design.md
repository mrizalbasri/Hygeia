# Profil pasien mandiri

## Cakupan yang disetujui

Pasien melihat dan mengedit profilnya sendiri. Nama lengkap, tanggal lahir,
jenis kelamin, telepon, alamat, serta pasangan nama/telepon kontak darurat.
Email dan nomor pasien hanya baca. Tidak ada kartu, QR baru, NIK, KTP, foto,
perubahan password/email, atau akses profil oleh provider/admin dalam fitur ini.

## Data dan kompatibilitas

Migrasi 11 menambah kolom nullable `birth_date` dan kolom kontak opsional pada
`patients`; nama tetap bersumber dari `users.full_name`. Tidak mengarang tanggal
lahir untuk data lama yang hanya memiliki `birth_year`. Mengisi tanggal lengkap
menyelaraskan tahun; mengosongkannya mempertahankan tahun terakhir yang diketahui.
Kontak darurat boleh kosong, tetapi nama dan telepon harus diisi bersama.

## Kontrak dan keamanan

`GET /api/v1/me/profile` membaca profil dari sesi aktif. `PATCH` menerima snapshot
seluruh field editable, bukan ID pasien. Field identitas, role, email, dan nomor
pasien ditolak sebagai unknown field. Tidak ada parameter pemilihan pasien.
Hanya role patient; sesi kedaluwarsa/akun nonaktif ditolak. PATCH memakai CSRF,
validasi server, SQL berparameter, dan transaksi untuk nama, demografi, serta
audit `patient.profile_update`. Audit tidak menyalin alamat/telepon/tanggal lahir.
Error audit membatalkan seluruh perubahan.

## Antarmuka

Menu Profil saya berada di dropdown nama/avatar pada navbar ruang pasien, bersama
Keluar. Sidebar berfokus pada menu perawatan. Dropdown tetap terlihat di mobile,
mendukung keyboard, tutup dengan Escape (fokus kembali ke pemicu), dan klik di luar.
Halaman profil mengikuti token, tipografi, form,
focus state, dan breakpoint dashboard yang ada (mode Operate). Header profil,
identitas hanya baca, lalu kelompok Data diri dan Kontak darurat. Form memiliki
loading, retry, simpan, batal perubahan, pesan error yang mempertahankan isian,
dan konfirmasi simpan. Nama pada shell dashboard disegarkan setelah menyimpan.
Tidak mengubah identitas visual global.

## Bukti yang diperlukan

Tes validasi tanggal/telepon/panjang/Unicode; integrasi DB untuk isolasi pasien,
penolakan role/sesi/CSRF/field terlarang, persistensi, kompatibilitas tahun lama,
atomicity audit, dan migrasi. E2E browser fixture untuk load/edit/save/reload,
error/retry, submit ganda, read-only, pembaruan nama shell, dan layout mobile.
Tes live dengan PostgreSQL sementara untuk menghubungkan kontrak dengan API nyata.
