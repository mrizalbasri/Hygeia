# Kontribusi ke Hygeia

Hygeia adalah prototipe pertukaran data kesehatan. Gunakan data sintetis dalam
contoh, fixture, screenshot, laporan masalah, dan pengujian.

## Siapkan development

Ikuti [panduan mulai](README.md#instalasi-lokal), lalu baca
[frontend](frontend/README.md) atau [backend](backend/README.md) sesuai bagian
yang diubah. Periksa implementasi dan tes modul sebelum mengubah kontrak API.

Jangan memasukkan `.env`, password, cookie sesi, token CarePass, kunci audit,
atau data pasien nyata ke Git maupun deskripsi issue/PR.

## Membuat perubahan

1. Buat branch sesuai tujuan: `feat/`, `fix/`, `docs/`, `refactor/`, `test/`, atau `chore/`.
2. Batasi perubahan pada satu tujuan yang bisa ditinjau.
3. Perbarui kontrak API atau panduan penggunaan jika perilaku berubah.
4. Jalankan pemeriksaan yang relevan dan tulis hasil aktual pada PR.

Gunakan Conventional Commits dengan judul berbahasa Inggris, misalnya:

```text
feat(carepass): add redemption history
fix(lab): reject duplicate result submissions
docs(readme): clarify local development setup
```

Judul PR mengikuti format yang sama. Deskripsi PR menjelaskan tujuan,
perubahan utama, tes yang dijalankan, tes yang belum dijalankan, dan perubahan
kontrak atau langkah migrasi bila ada.

## Pemeriksaan backend

Dari root repository, set `TEST_DATABASE_URL` ke database development/test,
lalu jalankan:

```powershell
./scripts/Test-Backend.ps1
```

Untuk memakai koneksi pada `backend/.env` lokal dengan `APP_ENV=development`
atau `test`:

```powershell
./scripts/Test-Backend.ps1 -UseLocalDatabase
```

Skrip menjalankan tes dan `go vet` dengan integrasi database aktif. Jangan
menganggap integrasi sudah diuji jika tes dilewati karena konfigurasi database
tidak tersedia. Untuk memeriksa build, dari `backend`:

```powershell
go build ./cmd/api ./cmd/migrate ./cmd/seed
```

Perubahan skema mengikuti pola migrasi SQL yang sudah ada. Tinjau dampaknya
terhadap data lama, transaksi, dan verifikasi audit.

## Pemeriksaan frontend

Dari `frontend`:

```powershell
pnpm install --frozen-lockfile
pnpm exec playwright install chromium
pnpm check
pnpm lint
pnpm build
pnpm test
pnpm test:e2e
```

Gunakan `$env:PLAYWRIGHT_CHANNEL='msedge'` bila menjalankan tes dengan Edge.
Panduan frontend menjelaskan tes fixture dan tes live opt-in yang membutuhkan
API serta database demo aktif.

## Menjaga kontrak klinis

- Backend menentukan otorisasi, kepemilikan resource, dan transisi status.
- Mutasi memakai proteksi CSRF dan mengikuti pola transaksi serta audit modul.
- Hasil lab awal dan koreksinya mempertahankan pola append-only.
- Respons provider mengikuti batas data peran dan penugasan yang disetujui.
- Perubahan izin perlu memeriksa jalur akses ditolak, izin kedaluwarsa, dan
  konflik status yang relevan.

Lihat [workflow CI](.github/workflows/verify.yml) untuk pemeriksaan otomatis.
