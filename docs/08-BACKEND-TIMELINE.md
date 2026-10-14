# Timeline perjalanan pasien

`GET /api/v1/patient/timeline` menyajikan milestone perjalanan perawatan dari
kejadian bisnis yang berhasil. Pasien melihat undangan dan keputusan episode,
proses lab, hasil yang sudah dikirim, resep dan pemenuhannya, serta keputusan
consent. Aktivitas membaca data dan akses yang ditolak tetap tersedia terpisah
melalui `/patient/access-summary`.

## Akses dan query

Gunakan cookie sesi yang sama dengan endpoint lain. Hanya pasien pemilik yang
diizinkan; identitas pasien diambil dari sesi, tanpa input patient ID/ref/rantai.
Dokter, lab, apotek, dan admin menerima `403`; sesi tidak sah menerima `401`.
Pasien lain hanya menerima perjalanan miliknya sendiri.

| Parameter | Nilai | Default |
|---|---|---|
| `kind` | `episode`, `laboratory`, `prescription`, `consent`; kosong berarti semua | semua |
| `limit` | Bilangan bulat 1–100 | 20 |
| `before_seq` | Bilangan bulat positif sampai 9223372036854775807; eksklusif | tanpa batas atas |

Parameter tak dikenal (termasuk `patient_id`, `patient_ref`, `chain`, `offset`),
query duplikat, URL encoding rusak, filter tidak sah, limit di luar batas,
serta cursor nol/negatif/overflow menghasilkan `400 invalid_input`.

## Respons

```json
{
  "events": [
    {
      "seq": 42,
      "occurred_at": "2026-10-11T09:30:00Z",
      "kind": "prescription",
      "action": "prescription.ready",
      "resource_type": "prescription",
      "resource_id": "00000000-0000-0000-0000-000000000042",
      "actor_name": "Staf apotek demo",
      "message": "Staf apotek demo menandai resep Anda siap diambil."
    }
  ],
  "limit": 20,
  "next_before_seq": null
}
```

`events` selalu array, termasuk bila kosong. `seq` unik dalam rantai pasien,
menjadi ID stabil untuk item UI. Nama aktor berasal dari snapshot audit saat
kejadian, sehingga perubahan profil tidak menulis ulang riwayat. Tampilkan nama
dan pesan sebagai teks biasa, bukan HTML. Respons memakai `Cache-Control:
no-store` dan `X-Request-ID`.

Isi hasil lab, nama/dosis/aturan obat, tujuan consent, ID pengguna pasien/dokter,
serta context/hash audit tidak disertakan. `resource_type` dan `resource_id`
dapat dipakai untuk membuka detail melalui endpoint terkait, yang tetap
memeriksa izin sendiri. Pesan mendeskripsikan kejadian saat itu, bukan status
resource saat ini. Akhir episode tidak menghapus riwayat pasien.

## Pagination

Urutan terbaru dahulu berdasarkan `seq DESC`. Seq dapat memiliki celah karena
rantai juga menyimpan aktivitas akses yang tidak ditampilkan sebagai milestone.
Nomor urut mencerminkan urutan rantai, tanpa bergantung pada perubahan jam server.

```text
GET /api/v1/patient/timeline?kind=prescription&limit=2
  events: seq 42, 35; next_before_seq: 35

GET /api/v1/patient/timeline?kind=prescription&limit=2&before_seq=35
  events: seq 31, 29; next_before_seq: null
```

Gunakan `next_before_seq` persis dari respons sebelumnya, dengan filter yang
sama. Nilainya hanya diisi jika masih ada milestone pada halaman berikut.
Event baru yang ditambahkan setelah halaman pertama tidak menggeser halaman
lanjutan atau membuat duplikasi. Muat ulang tanpa cursor untuk melihat kejadian
terbaru. Jika UI menangani integer melalui JavaScript, pastikan tidak membulatkan
nilai seq di luar rentang safe integer.

## Kejadian yang disertakan

Hanya pasangan action/resource type yang terdaftar, dengan outcome `allow`:

| Jenis | Action | Resource type |
|---|---|---|
| `episode` | `episode.invite`, `episode.accept`, `episode.reject`, `episode.end` | `care_episode` |
| `laboratory` | `lab_request.create`, `lab_request.approve`, `lab_request.reject`, `lab_request.cancel`, `lab_request.accepted`, `lab_request.in_progress` | `lab_request` |
| `laboratory` | `lab_result.submit` | `lab_result` |
| `prescription` | `prescription.create`, `prescription.approve`, `prescription.reject`, `prescription.cancel`, `prescription.accepted`, `prescription.preparing`, `prescription.ready`, `prescription.completed` | `prescription` |
| `consent` | `consent.request`, `consent.approve`, `consent.deny`, `consent.revoke` | `consent` |

`episode.reject` / `lab_request.reject` / `prescription.reject` /
`consent.deny` dengan outcome `allow` adalah keputusan pasien yang sah dan
ditampilkan. Percobaan tindakan yang ditolak otorisasi atau status, dengan
outcome `deny`, dikecualikan. Read/list, aksi tak dikenal, dan pasangan resource
type yang tidak cocok juga dikecualikan.

Consent kedaluwarsa otomatis belum menghasilkan event bisnis tersimpan;
timeline tidak menciptakan milestone kedaluwarsa. Status izin terkini tetap
dapat dibaca pada endpoint consent.

## Konsistensi dan kegagalan

Timeline membaca `audit_events` yang sudah ditulis atomik bersama aksi bisnis.
Aksi yang rollback karena audit gagal tidak muncul. Riwayat sebelum fitur ini
langsung tersedia tanpa backfill. Tidak ada tabel atau migrasi baru; versi
schema tetap 5. Pembacaan hanya memilih kolom metadata yang diperlukan dan
tidak menulis event baru, konsisten dengan ringkasan akses.

Timeline tidak otomatis memverifikasi seluruh rantai; gunakan `/audit/verify`
untuk verifikasi integritas. Respons gagal query/transaksi adalah `503
unavailable`, tanpa data event atau detail SQL. Timeline bukan pengganti audit,
notifikasi, detail klinis, atau akses staf lintas pasien.

## Pengujian

Dari folder `backend`, dengan PostgreSQL khusus pengujian:

```powershell
$env:TEST_DATABASE_URL = 'postgres://USER:PASSWORD@HOST:PORT/TEST_DB?sslmode=disable'
go test ./internal/care -run Timeline -v -count=1
go test ./cmd/... ./db/... ./internal/... -count=1
go vet ./cmd/... ./db/... ./internal/...
```

Fixture memakai schema acak dan pengguna sintetis. Tes mencakup seluruh jenis
milestone, nama historis, kebijakan akses, isolasi pasien, validasi query,
pagination ketika event baru ditambahkan, pengecualian read/deny/aksi tak dikenal,
rollback audit, refresh tanpa side effect, dan respons aman saat query gagal.
