# Autentikasi Clerk untuk Hygeia

Pengguna memilih Clerk, menunda SMTP mandiri, dan mengizinkan pembuatan aplikasi
Clerk development **Hygeia**. Tidak mengubah aplikasi Clerk proyek lain.

- Gunakan ClerkJS resmi pada SvelteKit dan SDK Go resmi untuk verifikasi JWT.
  UI SignIn/SignUp/UserProfile menangani pemulihan password, verifikasi email,
  daftar sesi, dan pengaturan MFA. Landing tetap publik.
- Backend memverifikasi tanda tangan, waktu, issuer instance Hygeia, serta azp
  origin yang diizinkan. Session Clerk harus aktif dan email utama terverifikasi.
  Role/organisasi/consent tetap ditentukan database Hygeia; metadata Clerk dari
  browser tidak memberi hak staf. Akun baru hanya menjadi pasien.
- Identitas Clerk dihubungkan lewat ID unik. Email yang sudah ada tidak
  dihubungkan otomatis. Operator dapat menautkan akun lama secara eksplisit
  setelah memeriksa identitas, tanpa memindahkan catatan klinis.
- Mirror sesi internal memakai HMAC server dari subject/session ID; token
  internal tidak diberikan ke browser. Middleware mewajibkan JWT pada setiap
  request, lalu meneruskan credential internal ke domain yang sudah ada.
  Sesi aktif Clerk diperiksa ulang saat pembacaan domain/SSE; kegagalan provider
  menutup akses. Pencabutan sesi berlaku pada pembacaan berikutnya.
- Admin/staf harus mengaktifkan dan melewati faktor kedua untuk akses domain.
  Halaman pengaturan akun tetap dapat dibuka untuk setup. Login lokal hanya
  untuk development/test; mode Clerk menolak endpoint password lokal dan cookie
  sesi lokal. Produksi harus memilih Clerk secara eksplisit.
- CSRF tetap aktif untuk mutasi. Tidak memakai webhook untuk otorisasi klinis:
  status sesi/user diperiksa langsung agar tidak bergantung pada pengiriman event.
- Staf diprovisikan secara eksplisit; akun demo lama dipertahankan. Kredensial
  hanya di env yang diabaikan Git; frontend hanya memiliki publishable key.
- Admin dapat menetapkan role staf pada onboarding Clerk yang masih kosong.
  Backend mengunci akun/profil, menolak bila episode atau audit pasien ada,
  menghapus hanya profil pasien kosong, mencatat aktivitas admin, dan mencabut
  sesi lama. Bootstrap admin pertama memakai perintah operator eksplisit dengan
  pemeriksaan riwayat yang sama; tidak dijalankan otomatis untuk akun manusia.

Alternatif adaptor komunitas SvelteKit menambah dependensi framework. ClerkJS
resmi dipilih agar bisa mengikuti versi SvelteKit saat ini, sedangkan backend
tetap memakai SDK Go resmi dan pola domain yang sudah diuji.

Validasi: JWT salah/expired/issuer/azp, sesi revoked/banned, email belum
terverifikasi, larangan auto-link/email dan eskalasi staf, MFA staf, logout,
penolakan local-cookie bypass, reauth SSE, serta regresi semua flow lokal.
Uji browser Clerk development menggunakan akun sintetis terpisah bila tersedia;
hasil lokal/fixture tidak diklaim sebagai kesiapan production instance.
