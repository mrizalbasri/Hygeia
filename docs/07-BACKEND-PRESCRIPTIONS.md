# Backend resep dan apotek

Base URL: `/api/v1`. Semua endpoint memakai sesi cookie yang sama dengan API
care/lab. Mutasi memakai JSON, cookie CSRF, dan header `X-CSRF-Token`; respons
terlindungi memakai `Cache-Control: no-store` dan `X-Request-ID`.

## Endpoint

| Metode | Endpoint | Aktor dan fungsi |
|---|---|---|
| GET | `/pharmacies` | Pasien: direktori apotek ID/nama |
| POST | `/prescriptions` | Dokter pembuat episode aktif: terbitkan resep |
| GET | `/prescriptions` | Pasien/dokter/apotek: daftar sesuai izin |
| GET | `/prescriptions/{id}` | Detail sesuai izin |
| POST | `/prescriptions/{id}/approval` | Pasien pemilik: pilih apotek dan setujui, atau tolak |
| PATCH | `/prescriptions/{id}/status` | Staf apotek tujuan: pindahkan satu tahap |

Direktori dan daftar mendukung `limit` (default 20, 1–100) dan `offset`
(default 0, 0–10000). Daftar resep mendukung filter `status` dari seluruh status
di bawah; urutan `created_at DESC,id DESC`. Direktori diurutkan `name,id`.
Offset dapat bergeser saat resep baru dibuat. Hasil daftar dapat lebih pendek
dari limit bila izin/status berubah selama permintaan menunggu row lock.
Direktori tidak memuat data pasien dan pembacaannya tidak menambah audit pasien.

## Membuat resep

`POST /prescriptions`:

```json
{
  "episode_id": "UUID-episode-aktif",
  "items": [
    {
      "medication_name": "Obat sintetis untuk demo",
      "dosage": "Dosis demo",
      "instructions": "Aturan pakai sintetis",
      "quantity": 2
    }
  ]
}
```

`episode_id` harus UUID valid (ganti placeholder contoh). Item wajib 1–20,
berurutan sesuai input. Nama dan dosis masing-masing 1–200 karakter, aturan
pakai 1–1000 karakter, jumlah bilangan bulat 1–10000. Teks dipangkas pada kedua
ujung, tidak boleh kosong atau memuat NUL. Batas body 64 KiB. Field asing,
termasuk dokter/pasien/apotek yang dipaksakan klien, ditolak. Identitas dokter
dan pasien selalu diturunkan dari sesi dan episode.

Respons `201 {"prescription": Prescription}` dengan
`Location: /api/v1/prescriptions/{id}`. Detail item tidak bisa diedit melalui API;
perubahan resep memerlukan penerbitan resep baru. Database memblokir
UPDATE/DELETE/TRUNCATE biasa pada item yang sudah diterbitkan.

## Pilihan pasien dan transisi

Pasien mengambil direktori dengan `GET /pharmacies`:

```json
{"pharmacies":[{"id":"UUID-apotek","name":"Apotek demo"}],"limit":20,"offset":0}
```

`POST /prescriptions/{id}/approval`:

```json
{"decision":"approve","pharmacy_org_id":"UUID-apotek-pilihan-pasien"}
```

Persetujuan berlaku satu kali dan menetapkan penerima resep. Organisasi harus
bertipe `pharmacy`. Penggantian apotek atau persetujuan ulang menghasilkan `409`.
Untuk menolak, kirim `{"decision":"reject"}` tanpa `pharmacy_org_id`.
Keduanya mengembalikan `200 {"prescription": Prescription}`.

| Status awal | Aksi | Status berikut |
|---|---|---|
| `pending_patient_approval` | Pasien approve dan pilih apotek | `sent` |
| `pending_patient_approval` | Pasien reject | `cancelled` (`patient_rejected`) |
| `pending_patient_approval` | Episode diakhiri | `cancelled` (`episode_ended`) |
| `sent` | Apotek accepted | `accepted` |
| `accepted` | Apotek preparing | `preparing` |
| `preparing` | Apotek ready | `ready` |
| `ready` | Apotek completed | `completed` |

`PATCH /prescriptions/{id}/status` menerima `{"status":"accepted"}`,
`preparing`, `ready`, atau `completed`. Respons
`200 {"prescription": QueuePrescription}` termasuk respons transisi terakhir.
Melompati, membalik, atau mengulang tahapan menghasilkan `409` untuk apotek
yang ditugaskan. Pengguna lain menerima `403`.

Episode yang berakhir mencabut akses klinis dokter, membatalkan resep pending,
dan mempertahankan penugasan apotek yang sudah disetujui sampai selesai. Ini
konsisten dengan perilaku penugasan laboratorium. Status selesai mengeluarkan
resep dari antrean dan menutup pembacaan berikutnya oleh apotek. Pasien tetap
dapat membaca riwayatnya. Pembatalan setelah pengiriman, transfer apotek,
penyuntingan obat, dan consent lintas dokter untuk resep tidak tersedia.

## Bentuk respons dan aturan akses

`Prescription` (pasien/dokter) memuat:

```json
{
  "id":"UUID-resep",
  "episode_id":"UUID-episode",
  "patient_ref":"HYGEIA-DEMO",
  "pharmacy_org_id":null,
  "pharmacy_name":null,
  "items":[{"medication_name":"Obat sintetis","dosage":"Demo","instructions":"Demo","quantity":2}],
  "status":"pending_patient_approval",
  "created_at":"2026-10-11T00:00:00Z",
  "updated_at":"2026-10-11T00:00:00Z",
  "approved_at":null,
  "completed_at":null,
  "cancelled_at":null,
  "cancellation_reason":null
}
```

Saat disetujui, `pharmacy_org_id`, `pharmacy_name`, dan `approved_at` terisi.
Saat selesai, `completed_at` terisi. Pembatalan mengisi `cancelled_at` dan
`cancellation_reason`, tanpa penugasan apotek.

`QueuePrescription` adalah proyeksi khusus apotek:

```json
{
  "id":"UUID-resep",
  "patient_ref":"HYGEIA-DEMO",
  "items":[{"medication_name":"Obat sintetis","dosage":"Demo","instructions":"Demo","quantity":2}],
  "status":"sent"
}
```

Tidak ada episode, ID pengguna pasien/dokter, hasil lab, atau data organisasi
klinik dalam proyeksi apotek. `GET /prescriptions` mengembalikan
`{"prescriptions":[],"limit":20,"offset":0}` dengan masing-masing item sesuai
proyeksi aktornya. Detail memakai `{"prescription":...}`.

| Aktor | Detail/daftar | Mutasi |
|---|---|---|
| Pasien pemilik | Semua status resep sendiri | Approve/reject resep pending dalam episode aktif |
| Dokter episode, masih anggota klinik episode | Resep sendiri selama episode aktif | Buat resep dalam episode aktif |
| Staf apotek pilihan pasien | Hanya `sent/accepted/preparing/ready` milik organisasinya | Transisi berurutan sampai selesai |
| Pasien lain, dokter lain termasuk rekan klinik, apotek lain | Detail `403`; daftar tidak memuat resep tersebut | `403` |
| Lab, admin | `403` | `403` |

Resep yang tidak ada memakai respons `403` yang sama dengan akses tidak sah.
Keputusan izin dilakukan di server. Daftar memeriksa ulang akses setelah
menunggu lock, agar berakhirnya episode atau selesainya resep tidak
mengembalikan data klinis dari snapshot lama.

## Audit, kegagalan, dan pengujian

Setiap detail dan mutasi resep yang ada dicatat pada rantai HMAC pasien,
termasuk penolakan akses/status. Pembacaan daftar mencatat satu event per pasien
yang resepnya dikembalikan, dengan satu ID resep representatif. Permintaan
dengan format tidak valid, direktori, daftar kosong, serta ID yang tidak ada
tidak menghasilkan event resep pasien. Audit tidak menyimpan nama obat, dosis,
aturan pakai, atau jumlah obat. Basis akses:
`patient_self`, `episode_doctor`, `approved_pharmacy_assignment`, `no_access`,
`invalid_state`, atau `episode_ended`.

Ringkasan `/patient/access-summary` mengenali event resep dalam bahasa Indonesia;
`/audit/verify` memverifikasi rantai yang sama. Penulisan bisnis dan audit
berada dalam satu transaksi. Kegagalan audit membatalkan pembuatan/item,
approval/reject/status, dan pengakhiran episode beserta pembatalan pending.
Payload klinis tidak dikembalikan ketika transaksi gagal.

Error mengikuti kontrak umum: `400 invalid_input`, `401 unauthenticated`,
`403 forbidden` (atau penolakan middleware CSRF), `409 invalid_state`,
`415 invalid_content_type`, `503 unavailable`. Field error tidak memuat obat.

Jalankan dari `backend`, memakai PostgreSQL khusus pengujian:

```powershell
$env:TEST_DATABASE_URL = 'postgres://USER:PASSWORD@HOST:PORT/TEST_DB?sslmode=disable'
go test ./cmd/... ./db/... ./internal/... -count=1
go vet ./cmd/... ./db/... ./internal/...
```

Tanpa `TEST_DATABASE_URL`, tes integrasi dilewati. Fixture membuat schema acak
dan menghapus hanya schema itu, menerapkan migrasi 1–5, serta memakai pengguna
sintetis. Tes mencakup lifecycle HTTP, CSRF, matriks akses, validasi item,
proyeksi apotek, penolakan pasien, episode berakhir, race approval/end/status,
akses setelah menunggu lock, integritas audit, dan rollback kegagalan audit.

Upgrade database aplikasi memakai `go run ./cmd/migrate up` sebelum API baru
dijalankan. Tidak perlu mengulang seed bila organisasi dan akun demo sudah ada.
