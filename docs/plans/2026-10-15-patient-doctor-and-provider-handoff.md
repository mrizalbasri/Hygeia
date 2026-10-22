# Penyelesaian frontend pasien/dokter dan handoff penyedia

## Tujuan dan baseline

Melanjutkan PR #16 pada branch `codex/patient-doctor-workflows`. Pasien dan dokter dapat menyelesaikan semua alur yang didukung kontrak backend 05–13 melalui web. Lab dan apotek dikerjakan teman pengembang dengan kontrak dan kriteria penerimaan di bawah. Tidak mengubah aturan klinis atau memberi diagnosis/resep otomatis.

## Cakupan pasien

- Login/logout, pembatasan route peran, ringkasan dengan data sesi sendiri.
- Episode: daftar/filter/pagination, detail, menerima/menolak undangan, mengakhiri hubungan dengan konfirmasi konsekuensi.
- Pemeriksaan: daftar/status, approve/reject pengiriman, membuka hasil selesai tanpa interpretasi klinis buatan.
- Resep: daftar/detail item, pilih apotek dari direktori, approve/reject, status penyiapan.
- Consent: dokter/klinik/tujuan/data ditampilkan, pilihan 1/24/168 jam (default terpendek), approve/deny/revoke. Pencabutan tidak menghapus akses episode yang masih sah.
- CarePass: pilih hasil/resep, TTL 15/60 menit, QR/link hanya tampil setelah create, metadata daftar, revoke. Token tidak disimpan di storage dan hilang saat panel ditutup.
- Timeline dengan filter jenis dan cursor; riwayat akses terpisah; verifikasi integritas audit dan riwayat akses darurat.
- Inbox notifikasi, pagination cursor, tandai dibaca, pembaruan SSE, penanganan sesi berakhir.

## Cakupan dokter

- Daftar episode milik dokter, filter/pagination/detail; undang memakai kode pasien persis (tanpa pencarian pasien global), akhiri episode dengan konfirmasi.
- Pilih episode aktif; pilih laboratorium dari direktori; input kode/nama tes; kirim permintaan; baca hasil sesuai izin.
- Buat resep dengan 1–20 item, nama obat/dosis/aturan pakai/jumlah; daftar dan detail resep.
- Minta akses satu hasil dengan kode pasien, ID hasil, dan tujuan; daftar izin efektif; buka hasil hanya lewat endpoint detail berotorisasi.
- Tebus CarePass melalui token atau tautan fragment; hapus fragment segera, tidak menyimpan token; tampilkan resource yang benar melalui endpoint terkait.
- Akses darurat: kode pasien, kode alasan, alasan 10–1000 karakter, konfirmasi akuntabilitas; daftar sesi, expiry, direktori resource berpagination; baca detail sesuai grant, tanpa hak perubahan.
- Inbox notifikasi dan sesi/logout seperti pasien.

## Rancangan teknis

Pertahankan SvelteKit, tema Hygeia, dan shell autentikasi. Komponen workflow pasien/dokter memakai modul API bersama: same-origin credentials, CSRF setiap mutasi, timeout request, pesan error dan request ID, 401 menuju login, hapus payload klinis sebelum membaca resource baru. Data server menentukan izin dan status; jangan menyaring otorisasi hanya di klien. Daftar mendukung status/pagination; cursor besar dipertahankan sebagai string. Form tidak mengirim ID pengguna/peran untuk menentukan kepemilikan. Tidak ada token sesi/CarePass di localStorage, sessionStorage, log, atau query string.

Komponen terpisah untuk daftar/form domain, pembacaan detail klinis, notifikasi, dan CarePass. Form mutasi mencegah submit ganda, memuat ulang status setelah sukses atau 409, dan membedakan kosong dari gagal. Kontrol tersedia di desktop/mobile dan keyboard. Konfirmasi hanya diperlukan pada tindakan pengguna yang mengakhiri atau mencabut akses.

## Tugas teman: laboratorium

Branch: `codex/frontend-laboratory`, mulai dari branch frontend bersama setelah kontrak komponen shared siap. File domain: `frontend/src/lib/workflows/LaboratoryWorkspace.svelte`; ubah wiring role lab saja, jangan menimpa pasien/dokter.

1. Antrean `/lab-requests?status=...&limit=20&offset=...`; status `sent/accepted/in_progress`, tampilkan kode pasien, kode/nama tes, status. Gunakan proyeksi `LabQueueRequest`, bukan data pasien penuh.
2. Detail `GET /lab-requests/{id}`; accepted hanya dari sent, in_progress hanya dari accepted. `PATCH .../status` dengan CSRF. Muat ulang setelah 409, jangan melompati tahap.
3. Form hasil hanya untuk in_progress: value wajib 1–500, unit max100, reference_range max200, flag normal/high/low/abnormal/unknown, note max2000. `POST /lab-requests/{id}/result`; jangan kirim test_name/test_code dari form.
4. Setelah submit, tampilkan receipt id/waktu dan hapus data form. Hasil tidak dapat diedit atau dibaca ulang oleh lab. Header Location bukan izin membaca hasil.
5. Gunakan inbox shared, loading/empty/error, pagination, keyboard/mobile. Jangan menampilkan nama/email pasien atau hasil dari resource lain.

Selesai bila: permintaan belum disetujui pasien tidak tampil; lifecycle sent→accepted→in_progress→completed berhasil; double-submit ditahan; permintaan selesai keluar antrean; 403 tidak meninggalkan payload lama. Tes fixture + alur bersama dokter/pasien pada database demo terpisah, termasuk reject dan episode berakhir setelah assignment disetujui.

## Tugas teman: apotek

Branch: `codex/frontend-pharmacy`; komponen `frontend/src/lib/workflows/PharmacyWorkspace.svelte`, wiring role pharmacy saja.

1. Antrean `/prescriptions` dengan filter `sent/accepted/preparing/ready`, limit/offset. Detail memakai `QueuePrescription`: id, patient_ref, items, status; bukan episode atau profil pasien.
2. Detail obat/dosis/aturan/jumlah; jangan mengubah resep dokter.
3. `PATCH /prescriptions/{id}/status`: sent→accepted→preparing→ready→completed, satu tahap sekali. Konfirmasi penyelesaian penyerahan, pending state dan refresh setelah 409.
4. Resep completed keluar antrean dan akses apotek ditutup; jangan menjanjikan halaman riwayat selesai dari API ini. Kosongkan detail yang tidak lagi boleh diakses.
5. Gunakan inbox shared dan gaya form/pagination yang sama. Tidak boleh mengakses hasil lab, resep sebelum persetujuan, resep apotek lain, atau data pasien tambahan.

Selesai bila: hanya resep yang disetujui dan ditujukan ke apotek sendiri muncul; semua transisi berhasil; ready dapat dilihat pasien; completed keluar antrean; percobaan akses silang dan stale detail ditolak tanpa kebocoran. Tes mencakup episode yang berakhir setelah resep dibagikan.

## Integrasi dan uji penerimaan

- Pertahankan endpoint/key respons dari docs 05–13; perubahan kontrak dibahas terlebih dahulu, tidak ditebak.
- Uji UI: form required/bounds, detail lab dan resep, semua mutasi, status konflik, CSRF, gagal jaringan, sesi habis, pagination, scope peran, keyboard, lebar 320/375/414/768/1440.
- Uji integrasi: dokter undang→pasien terima→dokter minta lab→pasien approve→teman lab kirim hasil→pasien/dokter baca→dokter resep→pasien pilih apotek→teman apotek selesaikan. Uji consent lintas dokter, revoke/expiry, CarePass, audit dan akses darurat secara terpisah.
- Jangan menganggap fixture UI sebagai bukti otorisasi server. Catat hasil pengujian aktual dan keterbatasan lingkungan pada PR.
- Build, type check, lint, serta test relevan harus selesai sebelum push. Commit/push frontend pasien/dokter dan plan; buat PR bertumpuk di #16 tanpa merge otomatis.

## Kontrak komponen shared untuk teman

- Dashboard wiring: `frontend/src/routes/dashboard/[role]/+page.svelte`; pilih role `lab` / `pharmacy` dan render workspace baru. Pertahankan `requireUser` pada server load; role berasal dari sesi.
- `api<T>(path, method?, body?, signal?)` di `api.ts`: path tanpa `/api/v1`; mutasi POST/PATCH mengambil CSRF baru. Gunakan `ApiError.status` untuk 409 dan `errorText` untuk pesan aman/request ID.
- `PagedList`: props `path`, `listKey`, `revision`, `statuses`, `empty`, snippet `children(item)`. Tambah revision setelah mutasi sukses/409. Default pagination limit20/offset; jangan memakai cursor notifikasi untuk antrean provider.
- `Notifications`: props `run`, `busy`, `revision`; `Run` mengembalikan boolean sukses, mencegah submit ganda, dan menangani error parent. Gunakan `ClinicalWorkspace.run` sebagai contoh, jangan impor komponen pasien/dokter untuk menentukan hak provider.
- `workflow.css`: import sekali di workspace; bungkus form/daftar dalam `.wf-content`, gunakan `.wf-form-panel`, `.wf-row`, `.wf-actions`. Disable fieldset sampai onMount dan ketika mutasi berlangsung agar form SSR tidak submit sebelum hidrasi.
- `ClinicalDetail` khusus pembaca klinis pasien/dokter. Lab/apotek memakai endpoint queue dan proyeksi masing-masing, bukan mengambil detail klinis umum untuk mengakali cakupan provider.

## Hasil verifikasi implementasi pasien/dokter

- Svelte check: 0 error/warning; lint dan production build lolos.
- Vitest: 2 tests lolos dengan Microsoft Edge; impor contoh Welcome diperbaiki agar sesuai versi terpasang.
- E2E fixture: tindakan penolakan, CSRF, 409/status terbaru, 403 tanpa detail lama, 503/retry, cursor 9223372036854775806, redirect 401 dan SSE session-ended lolos.
- Integrasi demo lokal: undangan/accept, lab/approval/hasil, resep/pilih apotek/status ready, end episode, consent approve/revoke, CarePass create/redeem/revoke, akses darurat/direktori/detail, audit utuh (57 event), notifikasi/read dan timeline lolos. Lab/apotek diuji melalui API.
- Pasien/dokter pada 320/375/414/768/1440 px tanpa overflow halaman; screenshot diperiksa. Landing diuji tujuh lebar layar.
- Perubahan identitas menambahkan `patient_ref` akun sendiri tanpa membuka direktori pasien. Tes backend memakai PostgreSQL sementara di loopback :15432, schema transaksi di-rollback; seluruh paket source `./internal/... ./cmd/... ./db` lolos. Folder `backend/bin` berisi arsip lokal yang bukan source Git dan dikecualikan dari perintah ini.

## Hasil implementasi penyedia dan admin (PR #20)

- `LaboratoryWorkspace.svelte`: antrean proyeksi `LabQueueRequest`, transisi
  `sent → accepted → in_progress`, form hasil dengan batas 500/100/200/2000
  karakter, pengiriman sekali pakai, receipt, dan pengeluaran dari antrean.
- `PharmacyWorkspace.svelte`: antrean `QueuePrescription`, transisi
  `sent → accepted → preparing → ready → completed`, detail obat tanpa data
  episode/pasien tambahan, dan konfirmasi sebelum penyerahan selesai.
- `AdminWorkspace.svelte`: organisasi, akun, status audit HMAC, tinjauan akses
  darurat, log aktivitas, dan notifikasi. Daftar pengguna menampilkan nama
  organisasi melalui pemetaan `organization_id` ke direktori organisasi karena
  API hanya mengembalikan ID. Tinjauan akses darurat menampilkan ID dokter dan
  nama organisasi; nama dokter belum tersedia pada respons admin, sehingga
  ditampilkan sebagai ID, bukan ditebak.
- Semua mutasi memakai `api()` bersama (CSRF baru, timeout, `ApiError.status`,
  request ID). 403/409 memuat ulang daftar; tidak ada payload lama yang
  dipertahankan sebagai izin.
- E2E `provider-workflows.e2e.js` mencakup lifecycle lab dan apotek, hasil dan
  receipt, konfirmasi penyelesaian, pembuatan organisasi, ubah nama akun,
  nonaktifkan akun, tinjauan akses darurat, dan verifikasi rantai audit.
- Ketiga tes E2E memakai polling HTTP untuk menunggu Vite, karena Vite 8 tidak
  selalu mencetak URL dev server pada stdout non-TTY. `pnpm check`, `pnpm lint`,
  `pnpm build`, Vitest, dan ketiga E2E lolos pada branch PR.
- Batas: respons admin break-glass belum memuat nama dokter; UI menampilkan ID
  dokter dan nama organisasi dari direktori. Penambahan nama dokter pada backend
  dapat dilakukan terpisah tanpa mengubah kontrak klinis.
