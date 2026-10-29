# Integrasi Clerk — 29 Oktober 2026

Pengguna memilih Clerk dan mengizinkan pembuatan aplikasi development baru
**Hygeia**. Aplikasi tersebut telah dibuat dan ditautkan ke repository; aplikasi
Clerk proyek lain tidak diubah. Tidak ada SMTP mandiri yang ditambahkan.
Tidak ada commit, push, deployment publik, atau pemberian admin pada akun manusia.

## Yang sudah tersedia

- Login/registrasi memakai ClerkJS resmi pada SvelteKit 3. UserProfile tersedia
  di `/account/security` untuk pengaturan email/password, MFA/recovery code,
  serta sesi. Dashboard menautkan halaman ini.
- Instance development mewajibkan verifikasi email pada signup, password minimal
  15 karakter dengan pemeriksaan HIBP, mendukung aplikasi autentikator dan backup
  code. Sesi maksimum 8 jam dan inactivity timeout 30 menit. Konfigurasi yang
  diterapkan disimpan tanpa secret di `clerk-development-config.json`.
- Backend Go memverifikasi JWT dengan SDK resmi, issuer instance, waktu berlaku,
  dan authorized party exact-origin. Status sesi dan user diperiksa langsung ke
  Clerk, termasuk pada poll SSE. Kegagalan provider menutup akses.
- Role/organisasi/izin klinis tetap di database Hygeia. Pengguna baru hanya
  menjadi pasien; metadata role dari browser tidak berpengaruh. Staff/admin
  membutuhkan MFA aktif dan bukti faktor kedua pada token.
- Identitas Clerk tidak otomatis mengambil alih email akun lokal yang sama.
  Operator memiliki perintah penautan eksplisit dengan kecocokan email utama
  terverifikasi. ID lokal dan data klinis lama dipertahankan.
- Admin dapat memprovisikan staf dengan ID Clerk atau mengalihkan onboarding
  pasien Clerk yang masih kosong melalui **Tetapkan peran staf**. Konversi dengan
  episode/audit pasien ditolak. Konversi kosong mencabut sesi lama dan mencatat
  aktivitas admin. Bootstrap admin pertama memakai flag operator eksplisit dan
  tidak pernah dijalankan otomatis.
- Backend lokal dan frontend env telah beralih ke `AUTH_PROVIDER=clerk`.
  Migrasi lokal versi 12, `dirty=false`. API Docker telah dibangun ulang dan sehat.
  `AUTH_PROVIDER=local` tetap hanya untuk development/test demo. Produksi menolak
  provider lokal, Clerk test keys, dan authorized parties non-HTTPS.

## Pemeriksaan

| Pemeriksaan | Hasil |
| --- | --- |
| `./scripts/Test-Backend.ps1 -UseLocalDatabase` | Lulus dengan integrasi PostgreSQL dan `go vet` |
| Regresi autentikasi Clerk | JWT signed salah/expired/issuer lain/azp lain, cookie Clerk dan penolakan cookie lokal, role metadata, MFA staf, provider gagal, sesi dicabut, akun lokal dinonaktifkan, collision email |
| Regresi sesi panjang | Revalidasi context setelah pencabutan menolak pembacaan berikutnya, seperti poll SSE |
| Regresi peran staf | Pasien tidak dapat mempromosikan dirinya, akun kosong dapat diprovisikan admin, akun klinis dipertahankan, role staf existing tidak diganti diam-diam, sesi lama dicabut |
| `pnpm check` | Lulus, 0 error / 0 warning |
| `pnpm lint` | Lulus |
| `pnpm test` | 2 file / 2 tes lulus |
| `pnpm test:e2e` | Ketujuh suite regresi lokal lulus; fixture memilih `AUTH_PROVIDER=local` secara eksplisit |
| `pnpm build` | Lulus dengan batas/warning SDK yang dicatat di bawah |
| `HYGEIA_CLERK_LIVE=1 node tests/clerk-ui.live.js` | UI Clerk development nyata login/signup dimuat pada desktop 1440 dan mobile 390, tanpa overflow/error runtime; tidak mengirim email atau membuat akun orang |
| API lokal setelah cutover | Config hanya provider/public key; cookie/password lokal ditolak; akses klinis anonim ditolak |
| Pemeriksaan output client | Secret key Clerk backend tidak ditemukan di output client yang dihasilkan |

Screenshot login desktop/mobile dan signup mobile tersedia di
`frontend/.test-results/clerk/` dan telah diperiksa secara visual. Secret hanya
ada di env lokal yang diabaikan Git. `clerk doctor` memverifikasi aplikasi
tertaut/reachable; warning env frontend berasal dari nama env khusus SvelteKit
dan pemisahan secret ke backend. Tidak menyalin secret ke frontend agar doctor
menampilkan status tanpa warning. CLI juga melaporkan warning konfigurasi MCP
editor yang tidak terkait dengan integrasi ini; file editor tidak diubah.

## Batas validasi dan langkah berikutnya

Belum menguji login manusia lengkap, pengiriman email reset/verifikasi nyata,
enrollment MFA manusia, atau bootstrap admin manusia. Pengujian signed JWT dan
sesi memakai fixture PostgreSQL/provider terkontrol; UI live hanya membuktikan
SDK development dimuat. Tidak ada klaim instance production sudah siap.

Clerk UI dimuat lewat dynamic import, tetapi menghasilkan chunk sekitar
1,5 MB sebelum gzip dan warning `use client` saat build. Landing anonim tidak
memuat SDK melalui root layout. Adapter-auto belum memilih hosting produksi;
peringatan Vitest `transformIndexHtml` tetap ada. Perlu pengukuran beban dan
kuota/latensi Clerk API pada staging karena pemeriksaan sesi memakai jaringan.

Yang dibutuhkan dari pengguna: memilih akun Clerk untuk administrator pertama,
menyelesaikan signup/verifikasi dan MFA, lalu persetujuan penautan/provisioning
akun tersebut. Petunjuk operator ada pada bagian Autentikasi Clerk di
`backend/README.md`. Akun demo/data lama tetap ada, tetapi password demo lokal
tidak dipakai pada mode Clerk.

Sesudah itu: staging/domain/HTTPS dan Clerk production instance, uji alur
seluruh peran terhadap Clerk nyata, backup/restore DB beserta kunci audit,
monitoring, uji beban dan pilot bersama calon pengguna masih perlu diselesaikan.

Referensi implementasi:
[Clerk JavaScript](https://clerk.com/docs/js-frontend/getting-started/quickstart),
[Clerk Go](https://clerk.com/docs/go/getting-started/quickstart), dan
[verifikasi sesi](https://clerk.com/docs/guides/sessions/verifying).
