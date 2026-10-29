# Profil pasien mandiri

Menu **Profil saya** pada dashboard pasien membaca dan menyimpan profil melalui
API Go. Tidak ada kartu atau QR baru. Tidak menambahkan NIK atau dokumen identitas.

## Endpoint

`GET /api/v1/me/profile` dan `PATCH /api/v1/me/profile` mengembalikan
`{"profile": {...}}`. Keduanya hanya menerima sesi aktif role `patient`.
Tidak menerima ID atau query pemilihan pasien; identitas selalu berasal dari sesi.
PATCH memerlukan cookie CSRF dan header `X-CSRF-Token`.

GET mengembalikan `patient_ref`, `email`, `full_name`, `birth_date`, `birth_year`,
`sex`, `phone`, `address`, `emergency_contact_name`, `emergency_contact_phone`.
Tanggal/tahun yang belum diketahui adalah `null`; kontak kosong berupa `""`.

PATCH menerima **snapshot seluruh field editable**, bukan partial merge:

```json
{
  "full_name": "Pasien Sintetis",
  "birth_date": "2000-02-29",
  "sex": "female",
  "phone": "081234567890",
  "address": "Jalan Sintetis 1",
  "emergency_contact_name": "Kontak Sintetis",
  "emergency_contact_phone": "081234567891"
}
```

- Nama wajib 1–200 karakter setelah trim.
- Jenis kelamin: `female`, `male`, `other`, `unknown`.
- Tanggal opsional: `""` atau ISO `YYYY-MM-DD`, dari 1900-01-01 sampai hari ini (UTC).
- Telepon opsional: 7–15 digit, maksimal 32 karakter; `+` hanya di awal,
  spasi, tanda kurung, titik, dan tanda hubung diperbolehkan.
- Alamat opsional maksimal 1000 karakter.
- Nama kontak darurat maksimal 200 karakter; nama/telepon harus terisi bersama
  atau sama-sama kosong.
- Email, nomor pasien, role, ID, dan `birth_year` tidak editable; unknown fields
  ditolak. Input yang tidak dikirim bernilai kosong, sehingga client harus
  mengirim seluruh snapshot editable untuk menghindari penghapusan field opsional.

Error: 400 input/query tidak valid, 401 sesi tidak ada/kedaluwarsa/akun nonaktif,
403 role selain pasien atau CSRF tidak valid, 415 content type selain JSON,
503 kegagalan database/audit. Respons menggunakan `Cache-Control: no-store`.

## Migrasi dan audit

Jalankan `go run ./cmd/migrate up` sebelum mengaktifkan fitur; versi menjadi 11.
Migrasi mempertahankan `birth_year` dan tidak mengarang tanggal untuk data lama.
Saat tanggal lengkap diisi, tahun diselaraskan. Saat tanggal dikosongkan, tahun
terakhir tetap dipertahankan untuk kompatibilitas data/demografi lama.
Rollback migrasi menghapus kolom profil baru beserta nilainya, tetapi tidak
menghapus akun, nomor pasien, atau tahun lahir.

Nama pada `users` dan demografi pada `patients` disimpan atomik bersama audit
`patient.profile_update` dengan basis `patient_self`. Audit tidak menyimpan salinan
telepon, alamat, kontak darurat, atau tanggal lahir. Kegagalan audit menggagalkan
seluruh perubahan. Tidak menambah proyeksi data untuk dokter, lab, apotek, atau admin.

## Pengujian

- Backend: tes validasi di `internal/patientprofile`; tes PostgreSQL/HTTP,
  otorisasi, CSRF, isolasi, rollback audit, dan migrasi di `internal/care`.
  Gunakan `TEST_DATABASE_URL` dan `./scripts/Test-Backend.ps1`.
- Frontend: `pnpm check`, `pnpm lint`, `pnpm build`, `pnpm test`, `pnpm test:e2e`.
  Suite `tests/patient-profile.e2e.mjs` memakai API fixture, bukan database nyata.
  Screenshot sintetis disimpan di `.test-results/`.
- Browser/API nyata: setelah migrasi pada database sementara dan API di port 8080,
  set `HYGEIA_LIVE_TEST=1` lalu jalankan `node tests/patient-profile.live.mjs`.
  Tes membuat akun sintetis baru; jangan arahkan API ke database produksi.
