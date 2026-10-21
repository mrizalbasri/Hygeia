# Kontrak API Hygeia untuk frontend

Backend Go pada `http://127.0.0.1:8080`. Semua endpoint diawali `/api/v1`.
Setup database, migrasi, seed, dan API mengikuti [panduan backend](../backend/README.md).

Frontend sudah menyediakan `/login`, `/dashboard/{role}`, `/carepass`, helper
SSR identitas, proxy `/api`, dan alur pasien/dokter. Rincian cara menjalankan dan
pengujian ada di [panduan frontend](../frontend/README.md). Antarmuka lab/apotek
mengikuti [rencana handoff](plans/2026-10-15-patient-doctor-and-provider-handoff.md).

Browser memakai URL relatif `/api/v1/...` dengan cookie sesi. SSR meneruskan
cookie hanya ke `API_BASE_URL` yang dikonfigurasi aplikasi. Production memerlukan
reverse proxy satu origin dan HTTPS; CORS belum dibuka. Backend selalu menentukan
identitas, organisasi, izin resource, serta transisi status.

## Autentikasi dan error

| Metode | Endpoint | Request | Respons sukses |
|---|---|---|---|
| GET | `/auth/csrf` | — | `200 {"csrf_token":"..."}` + cookie CSRF |
| POST | `/auth/login` | `{"email":"...","password":"..."}` | `200 {"user": User}` + cookie sesi |
| POST | `/auth/logout` | `{}` | `204`, sesi dicabut dan cookie dihapus |
| GET | `/me` | — | `200 {"user": User}` |

`User`: `id`, `email`, `full_name`, `role`, `organization`, serta `patient_ref`
opsional untuk akun pasien sendiri (dipakai untuk undangan perawatan).
Role: `patient`, `doctor`, `lab`, `pharmacy`, `admin`. Organization berupa
`{id,name,type}` untuk staf atau `null`. Identitas dan organisasi selalu ditentukan
backend dari sesi; jangan mengirim `doctor_id`, `patient_id`, atau `role` untuk
menentukan pemilik tindakan.

Semua POST/PATCH memerlukan `Content-Type: application/json`, cookie CSRF, dan
header `X-CSRF-Token` dari `/auth/csrf`. Token sesi tidak disimpan di localStorage.
Jangan membiarkan form login submit sebagai GET. Respons memakai `no-store`.

Contoh error:

```json
{
  "error": {
    "code": "forbidden",
    "message": "Akses tidak diizinkan.",
    "request_id": "..."
  }
}
```

| HTTP | Code | Penanganan frontend |
|---|---|---|
| 400 | `invalid_input` | Perbaiki input, action, filter, atau pagination |
| 401 | `unauthenticated` | Arahkan ke login |
| 403 | `forbidden` | Tampilkan akses tidak diizinkan; jangan tampilkan data lama |
| 403 | `csrf_invalid` | Ambil token CSRF baru dan minta pengguna mencoba kembali |
| 409 | `invalid_state` | Muat ulang episode/permintaan; status berubah, undangan masih terbuka, atau hasil sudah dikirim |
| 415 | `invalid_content_type` | Kirim JSON |
| 429 | `rate_limited` | Login terlalu sering; ikuti `Retry-After` |
| 503 | `unavailable` | Tampilkan gangguan layanan, simpan `request_id` untuk pelacakan |

Login salah memakai `401 invalid_credentials`. ID episode yang tidak ada dan
ID yang tidak boleh diakses sama-sama mengembalikan `403 forbidden`, tanpa isi
episode. UUID yang formatnya tidak valid mengembalikan `400`.

## Episode

| Metode | Endpoint | Akses |
|---|---|---|
| POST | `/episodes` | Dokter membuat undangan |
| GET | `/episodes` | Dokter: episode sendiri pada klinik terkait; pasien: episode sendiri |
| GET | `/episodes/{id}` | Pasien pemilik atau dokter pengundang pada klinik episode |
| PATCH | `/episodes/{id}` | Pasien accept/reject; pasien atau dokter pengundang end |

**Membuat undangan:**

```http
POST /api/v1/episodes
Content-Type: application/json
X-CSRF-Token: <token>

{"patient_ref":"HYG-DEMO-001"}
```

Kode harus cocok persis dan peka huruf besar/kecil, panjang 1–64 karakter,
diawali huruf/angka dan selanjutnya hanya huruf, angka, `_`, atau `-`.
Tidak ada pencarian seluruh pasien. Respons `201 {"episode": Episode}` dan
header `Location`. Undangan ganda `pending`/`active` ke pasien yang sama oleh
dokter yang sama menghasilkan `409`; kode tidak ditemukan menghasilkan `403`.

**Menjawab undangan atau mengakhiri episode:**

```http
PATCH /api/v1/episodes/<id>
Content-Type: application/json
X-CSRF-Token: <token>

{"action":"accept"}
```

| Status awal | Action | Pelaku | Status hasil |
|---|---|---|---|
| `pending` | `accept` | Pasien pemilik | `active` |
| `pending` | `reject` | Pasien pemilik | `rejected` |
| `active` | `end` | Pasien pemilik atau dokter pengundang | `ended` |

Respons `200 {"episode": Episode}`. Tidak ada pengaktifan ulang atau penghapusan.
Episode terminal tetap dapat dibaca sebagai riwayat; akses klinis jalur A
memerlukan episode aktif. Klinik tidak memberikan akses otomatis kepada dokter
lain. Lab, apotek, dan admin tidak boleh membaca episode.

**Daftar:** `GET /episodes?status=pending&limit=20&offset=0`.
Filter opsional `pending|active|rejected|ended`; tanpa filter semua status.
`limit` default 20, rentang 1–100; `offset` default 0, rentang 0–10000.
Urutan terbaru berdasarkan `created_at DESC,id DESC`.

```json
{"episodes": [], "limit": 20, "offset": 0}
```

Daftar mengembalikan undangan dan riwayat. Gunakan `status=active` untuk hubungan
perawatan aktif. Array kosong berarti tidak ada episode yang terlihat pada
halaman itu; tidak ada total count. Jika jumlah hasil kurang dari limit, halaman
berikutnya kosong. Offset pagination dapat bergeser saat ada undangan baru.

**Bentuk Episode** (UUID dan timestamp berikut hanya contoh):

```json
{
  "id": "00000000-0000-0000-0000-000000000001",
  "patient_id": "00000000-0000-0000-0000-000000000002",
  "patient_ref": "HYG-DEMO-001",
  "doctor_id": "00000000-0000-0000-0000-000000000003",
  "doctor_name": "Dr. Maya",
  "organization_id": "00000000-0000-0000-0000-000000000004",
  "organization_name": "Klinik Sentra",
  "status": "pending",
  "created_at": "2026-10-11T03:00:00Z",
  "updated_at": "2026-10-11T03:00:00Z",
  "accepted_at": null,
  "closed_at": null
}
```

Timestamp RFC 3339 dengan offset; formatkan untuk zona waktu pengguna di browser.
Nama lengkap pasien, email pasien, dan isi klinis tidak dikembalikan oleh API ini.

## Riwayat akses dan verifikasi audit

`GET /patient/access-summary?limit=20&offset=0`, hanya pasien pemilik sesi.
Pagination sama dengan episode; urutan `seq DESC`.

```json
{
  "events": [
    {
      "seq": 1,
      "occurred_at": "2026-10-11T03:00:00Z",
      "actor_name": "Dr. Maya",
      "action": "episode.invite",
      "resource_id": "00000000-0000-0000-0000-000000000001",
      "outcome": "allow",
      "basis": "doctor_invitation",
      "message": "Dr. Maya mengirim undangan perawatan Anda."
    }
  ],
  "limit": 20,
  "offset": 0
}
```

Action saat ini: `episode.invite`, `episode.accept`, `episode.reject`,
`episode.end`, `episode.view`, `episode.list`, serta action laboratorium di bawah.
Outcome `allow|deny`; basis
`doctor_invitation|patient_self|episode_doctor|approved_lab_assignment|episode_ended|no_access|invalid_state`.
Tampilkan `message` sebagai teks biasa, bukan HTML. Riwayat belum diverifikasi
otomatis setiap dibaca; panggil endpoint verifikasi untuk mengetahui integritasnya.

`GET /audit/verify` memverifikasi rantai pasien yang login. Pasien boleh mengirim
`?chain=<patient_id>` hanya untuk dirinya. Admin wajib menyebut chain yang ingin
diperiksa dan hanya menerima hasil integritas, tanpa isi episode/event.
Dokter, lab, dan apotek tidak mendapat akses endpoint ini.

```json
{"chain":"00000000-0000-0000-0000-000000000002","events":12,"intact":true}
```

Jika rusak: `200 {"chain":"...","events":12,"intact":false,"first_mismatch_seq":5}`.
Rantai pasien yang belum memiliki event: `events:0,intact:true`.
`chain` adalah **patients.id**, berbeda dari `user.id` di `/me`.

## Contoh pemanggilan browser

```typescript
async function mutate(path: string, method: 'POST' | 'PATCH', body: unknown) {
  const csrfResponse = await fetch('/api/v1/auth/csrf', { credentials: 'same-origin' });
  if (!csrfResponse.ok) throw new Error('Tidak dapat menyiapkan permintaan.');
  const { csrf_token } = await csrfResponse.json();
  const response = await fetch('/api/v1' + path, {
    method,
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf_token },
    body: JSON.stringify(body)
  });
  const result = response.status === 204 ? null : await response.json();
  if (!response.ok) throw new Error(result.error.message);
  return result;
}

// Pada form login yang perlu dibuat; password berasal dari input pengguna:
await mutate('/auth/login', 'POST', { email, password });
// Sesudah login sebagai dokter:
await mutate('/episodes', 'POST', { patient_ref: 'HYG-DEMO-001' });
// Sesudah login sebagai pasien, gunakan ID yang dikembalikan API:
await mutate('/episodes/<episode-id>', 'PATCH', { action: 'accept' });
```

Contoh helper browser di atas perlu ditambahkan oleh pengembang frontend setelah
proxy tersedia. Untuk SSR, implementasikan helper server sesuai langkah integrasi
di atas; jangan memakai helper browser di server.

## Alur uji integrasi frontend

1. Dokter `maya@demo.hygeia.test` login, kirim undangan `HYG-DEMO-001`.
2. Pasien `aulia@demo.hygeia.test` login, muat daftar `status=pending`, lalu accept.
3. Dokter muat daftar `status=active`, buka detail.
4. Pasien muat riwayat akses dan verifikasi audit.
5. Dokter lain `budi@demo.hygeia.test` mencoba ID tersebut: harus `403`.
6. Pasien atau dokter mengakhiri episode. Status berubah `ended`; riwayat tetap ada.
7. Untuk mengetes reject, buat undangan baru setelah episode sebelumnya terminal.

Password demo mengikuti `DEMO_PASSWORD` saat seed awal; tidak ada password di
dokumen/repository. Untuk mesin rekan pengembang, setup `.env`, migrasi, dan seed
sendiri atau gunakan backend bersama melalui proxy satu origin.

API resep dan apotek tersedia; kontrak lengkap dan contoh request/response ada
di [07-BACKEND-PRESCRIPTIONS.md](07-BACKEND-PRESCRIPTIONS.md).
Timeline perjalanan pasien tersedia melalui `GET /patient/timeline`; kontrak
filter dan pagination cursor ada di [08-BACKEND-TIMELINE.md](08-BACKEND-TIMELINE.md).
Modul tambahan tersedia: [admin](09-BACKEND-ADMIN.md),
[notifikasi](10-BACKEND-NOTIFICATIONS.md), [akses darurat](11-backend-emergency-api.md),
[CarePass](12-BACKEND-CAREPASS.md), dan [SSE](13-BACKEND-REALTIME.md).
Riwayat akses mencakup episode, laboratorium, consent satu hasil lab, dan resep.

## Laboratorium

| Metode | Endpoint | Request | Respons sukses |
|---|---|---|---|
| GET | `/laboratories` | Pagination | `200 {laboratories:[{id,name}],limit,offset}` |
| POST | `/lab-requests` | `episode_id, lab_org_id, test_code, test_name` | `201 {lab_request: LabRequest}` + Location |
| GET | `/lab-requests` | Filter status dan pagination | `200 {lab_requests:[],limit,offset}` |
| GET | `/lab-requests/{id}` | — | `200 {lab_request: LabRequest atau LabQueueRequest}` |
| POST | `/lab-requests/{id}/approval` | `{"decision":"approve"}` atau `reject` | `200 {lab_request: LabRequest}` |
| PATCH | `/lab-requests/{id}/status` | `{"status":"accepted"}` atau `in_progress` | `200 {lab_request: LabQueueRequest}` |
| POST | `/lab-requests/{id}/result` | ResultInput | `201 {result: ResultReceipt}` + Location |
| GET | `/lab-results/{id}` | — | `200 {result: LabResult}` |

Semua mutasi memakai JSON dan CSRF. Pagination sama dengan episode, termasuk
direktori laboratorium: default limit 20, maksimal 100, offset maksimal 10000.
Direktori hanya untuk dokter, diurutkan `name,id`; direktori tidak berisi data
pasien dan pembacaannya tidak menambah audit pasien. Daftar permintaan diurutkan
`created_at DESC,id DESC`, tanpa total count.

### Hak akses dan status

- Jalur episode: dokter membuat/membaca permintaan serta hasil pada episode
  aktif yang dibuatnya, dengan keanggotaan klinik yang masih sesuai. Rekan satu
  klinik tidak mendapat akses otomatis. Hasil juga dapat dibaca melalui consent
  aktif untuk hasil tersebut; consent tidak membuka permintaan lab.
- Pasien membaca permintaan serta hasil miliknya, termasuk setelah episode
  berakhir. Hanya pasien tersebut yang boleh approve/reject saat permintaan
  `pending_patient_approval` dan episode masih aktif.
- Lab tujuan baru melihat permintaan setelah approve. Semua staf aktif di lab
  tujuan boleh memprosesnya. Lab lain, admin, dan apotek tidak boleh mengakses.
- `pending_patient_approval → sent → accepted → in_progress → completed`.
  Reject pasien menghasilkan `cancelled` dengan alasan `patient_rejected`.
  Perpindahan status tidak bisa dilewati atau diulang. Completion hanya lewat
  pengiriman hasil, bukan PATCH status.
- Mengakhiri episode membatalkan permintaan yang belum disetujui, dengan alasan
  `episode_ended`. Yang sudah disetujui tetap bisa diselesaikan lab; dokter
  kehilangan akses, sedangkan pasien tetap boleh membaca hasil.
- Lab melihat antrean `sent|accepted|in_progress` saja. Filter status terminal
  menghasilkan array kosong untuk lab. Detail permintaan terminal ditolak `403`.
  Pasien melihat semua status; dokter melihat permintaan pada episode aktif saja.
- Lab mengirim hasil sekali saat `in_progress` dan menerima receipt tanpa nilai
  hasil. Lab tidak boleh GET hasil atau mengedit hasil yang telah dikirim.
  Pengiriman ulang oleh lab tujuan menghasilkan `409`.
- ID yang tidak ditemukan dan resource yang tak boleh diakses sama-sama `403`.
  Format UUID salah, filter salah, dan input tidak valid menghasilkan `400`.

### Membuat permintaan dan menjawab persetujuan

```json
{
  "episode_id": "<episode-uuid>",
  "lab_org_id": "<laboratory-uuid>",
  "test_code": "DEMO-GLUCOSE",
  "test_name": "Tes sintetis demo"
}
```

Ambil ID lab dari `/laboratories`. Lab harus organisasi bertipe `laboratory`.
`test_code`: 1–64 karakter ASCII, diawali huruf/angka, selanjutnya huruf, angka,
`_`, `.`, atau `-`. `test_name`: 1–200 karakter setelah trim. Identitas dokter,
pasien, dan klinik diturunkan dari sesi/episode; field tambahan ditolak.
Permintaan tes yang sama boleh dibuat beberapa kali, masing-masing mendapat
ID baru dan persetujuan terpisah; tombol submit frontend perlu dinonaktifkan
saat pending. Tidak ada idempotency key pada tahap ini.

Pasien memanggil `/lab-requests/{id}/approval` dengan `{"decision":"approve"}`
atau `{"decision":"reject"}`. Persetujuan tidak bisa dicabut atau diubah setelah
diputuskan pada tahap ini. Tidak ada endpoint pembatalan manual.

### Bentuk respons permintaan

Pasien/dokter menerima `LabRequest`:

```typescript
type LabRequest = {
  id: string;
  episode_id: string;
  patient_ref: string;
  lab_org_id: string;
  lab_name: string;
  test_code: string;
  test_name: string;
  status: 'pending_patient_approval' | 'sent' | 'accepted'
    | 'in_progress' | 'completed' | 'cancelled';
  created_at: string;
  updated_at: string;
  approved_at: string | null;
  completed_at: string | null;
  cancelled_at: string | null;
  cancellation_reason: 'patient_rejected' | 'episode_ended' | null;
  result_id: string | null;
};
```

Tidak ada payload hasil dalam request/list. Gunakan `result_id` untuk GET hasil
secara terpisah. Seluruh respons permintaan untuk staf lab menggunakan tipe terpisah:

```typescript
type LabQueueRequest = {
  id: string;
  patient_ref: string;
  test_code: string;
  test_name: string;
  status: 'sent' | 'accepted' | 'in_progress';
};
```

Tidak ada episode ID, nama/email pasien, profil, hasil lain, atau catatan
klinis bebas pada respons lab. `id` adalah ID permintaan, bukan ID pasien.

### Mengirim dan membaca hasil

Contoh sintetis untuk POST `/lab-requests/{id}/result`:

```json
{
  "value": "NILAI-SINTETIS-DEMO",
  "unit": "",
  "reference_range": "",
  "flag": "unknown",
  "note": "Data sintetis, bukan hasil pemeriksaan nyata."
}
```

`value` wajib berupa teks nonkosong, maksimal 500 karakter; boleh nilai
nonnumerik. `unit` maksimal 100 dan `reference_range` maksimal 200 karakter;
keduanya boleh kosong dan default kosong jika tidak dikirim. `flag` wajib
`normal|high|low|abnormal|unknown`. `note` opsional, default kosong, maksimal
2000 karakter. Karakter NUL ditolak; batas body JSON 16 KiB.
Nama/kode tes mengikuti permintaan dan tidak boleh disertakan untuk menimpanya.

Receipt yang diterima lab tidak memuat nilai hasil:

```typescript
type ResultReceipt = {
  id: string;
  lab_request_id: string;
  status: 'completed';
  submitted_at: string;
};
```

Header Location menunjuk `/api/v1/lab-results/{id}`, tetapi GET endpoint tersebut
hanya diizinkan untuk pasien pemilik, dokter episode aktif, atau dokter dengan
consent aktif untuk hasil tersebut dan klinik yang sesuai. Lab tidak
mendapat hak membaca dari header Location.

```typescript
type LabResult = {
  id: string;
  lab_request_id: string;
  test_code: string;
  test_name: string;
  value: string;
  unit: string;
  reference_range: string;
  flag: 'normal' | 'high' | 'low' | 'abnormal' | 'unknown';
  note: string;
  submitted_at: string;
};
```

Timestamp RFC 3339 dengan offset. Hasil tidak bisa diedit/dihapus lewat API;
koreksi hasil, PDF, dan unggahan berkas belum tersedia. Pada kegagalan audit,
mutasi dibatalkan dan GET hasil mengembalikan `503` tanpa payload klinis.

### Audit dan skenario frontend

Action tambahan: `lab_request.create`, `lab_request.approve`, `lab_request.reject`,
`lab_request.cancel`, `lab_request.view`, `lab_request.list`, `lab_request.status`
(percobaan tidak sah), `lab_request.accepted`, `lab_request.in_progress`,
`lab_result.submit`, `lab_result.view`. Payload audit tidak memuat nilai hasil.
Daftar dicatat sekali per pasien yang muncul di halaman, seperti daftar episode.
Tanpa sesi, gagal CSRF/validasi, direktori lab, ID tidak ada, provider tidak valid,
dan daftar kosong tidak menambah event rantai pasien.

1. Dokter login, pilih episode aktif, GET `/laboratories`, lalu buat tes.
2. Lab login: permintaan tersebut belum muncul.
3. Pasien login, GET `/lab-requests?status=pending_patient_approval`, lalu approve.
4. Lab login, muat antrean, PATCH accepted, lalu PATCH in_progress, lalu POST hasil.
5. Pasien/dokter login, muat detail permintaan untuk mengambil result_id, lalu GET hasil.
6. Dokter lain, lab, apotek, dan admin mencoba GET hasil: harus `403` tanpa nilai.
7. Pasien memeriksa riwayat akses dan verifikasi integritas.
8. Ulangi dengan permintaan baru untuk reject dan pengakhiran episode sebelum/
   sesudah approve. Pengakhiran tidak menghapus hasil yang sudah terkirim.

Frontend pasien/dokter mengintegrasikan endpoint ini, timeline, dan notifikasi
SSE. Gunakan refresh setelah mutasi untuk mengambil status terbaru; event SSE
memberi notifikasi, bukan payload hasil klinis. UI staf lab mengikuti rencana handoff.

## Consent untuk satu hasil laboratorium

Consent memberi dokter peminta izin membaca **satu hasil lab**, termasuk dokter
dari klinik berbeda. Identitas dokter dan kliniknya selalu diambil dari sesi.
Pasien tetap memiliki akses ke hasilnya sendiri. Semua mutasi memakai CSRF dan
format error yang sama dengan endpoint lain.

| Metode | Endpoint | Request | Respons |
|---|---|---|---|
| POST | `/consents` | `patient_ref,resource_type,resource_id,purpose` | `201 {consent:Consent}` + Location |
| GET | `/consents` | Filter `status`, `limit`, `offset` | `200 {consents:Consent[],limit,offset}` |
| GET | `/consents/{id}` | — | `200 {consent:Consent}` |
| PATCH | `/consents/{id}` | `action`, opsional `duration_hours` | `200 {consent:Consent}` |

Daftar pasien hanya memuat izin untuk dirinya. Daftar dokter hanya memuat
permintaannya pada klinik saat ini. Lab, apotek, dan admin tidak dapat mengakses
consent. Pagination default 20, maksimum 100, offset maksimum 10000.

Dokter membuat permintaan:

```json
{
  "patient_ref": "HYG-0001",
  "resource_type": "lab_result",
  "resource_id": "<result-uuid>",
  "purpose": "Pendapat kedua atas hasil pemeriksaan pasien"
}
```

`resource_id` adalah ID hasil, bukan ID permintaan lab. Pasien memberikan ID
hasil kepada dokter melalui proses di luar API ini; backend tidak menyediakan
direktori hasil pasien untuk dokter lain. Kode pasien harus cocok persis dengan
pemilik hasil tersebut. `purpose` wajib, maksimal 1000 karakter setelah trim.
Jenis resource yang tersedia hanya `lab_result`, tindakan hanya `view`.
Satu permintaan `pending` atau izin `active` per dokter, klinik, dan hasil;
duplikat menghasilkan `409`. Setelah ditolak, dicabut, atau kedaluwarsa, dokter
dapat mengajukan permintaan baru dengan ID consent baru.

Contoh respons pasien (UUID dan waktu contoh):

```json
{
  "consent": {
    "id": "<consent-uuid>",
    "patient_ref": "HYG-0001",
    "grantee_user_id": "<doctor-uuid>",
    "grantee_name": "Dokter Demo",
    "grantee_org_id": "<clinic-uuid>",
    "grantee_org_name": "Klinik Demo",
    "resource_type": "lab_result",
    "resource_id": "<result-uuid>",
    "allowed_action": "view",
    "purpose": "Pendapat kedua atas hasil pemeriksaan pasien",
    "status": "pending",
    "requested_at": "2026-10-11T08:00:00Z",
    "decided_at": null,
    "expires_at": null,
    "revoked_at": null,
    "test_code": "DEMO-TEST",
    "test_name": "Pemeriksaan sintetis"
  }
}
```

Respons dokter tidak memuat `test_code` atau `test_name`. Tidak ada respons
consent yang memuat nilai hasil, satuan, rentang rujukan, penanda, atau catatan.
Untuk membaca nilai setelah disetujui, dokter memakai `GET /lab-results/{id}`.

Pasien menyetujui:

```json
{"action":"approve","duration_hours":24}
```

Durasi wajib dipilih: `1`, `24`, atau `168` jam (7 hari), dihitung dari waktu
persetujuan server. Penolakan: `{"action":"deny"}`. Pencabutan:
`{"action":"revoke"}`. Deny dan revoke tidak menerima durasi selain nol.

Status: `pending → active → expired/revoked`, atau `pending → denied`.
Approve/deny hanya untuk pending; revoke hanya untuk active yang belum habis.
Transisi lain menghasilkan `409`; action/durasi tidak valid menghasilkan `400`.
Kedaluwarsa berlaku saat waktu server mencapai `expires_at`, tanpa job latar
belakang. GET dan filter daftar mengembalikan status efektif `expired`.

Izin terikat pada dokter dan kliniknya saat meminta. Pindah klinik memblokir
izin lama; dokter perlu mengajukan izin baru pada klinik baru. Consent tetap
berlaku setelah episode asal berakhir, hingga kedaluwarsa atau dicabut. Izin
hanya membuka hasil yang dipilih, bukan episode, permintaan lab, atau hasil
lain. Pencabutan memblokir pembacaan berikutnya, tetapi tidak menghapus akses
mandiri dokter pengundang selama episode masih aktif (basis A).

Perubahan dan audit memakai transaksi yang sama. Kegagalan audit menghasilkan
`503` dan membatalkan mutasi; hasil tidak dikirim ketika pencatatan pembacaan
gagal. Audit mencatat permintaan, keputusan, daftar/detail, pembacaan hasil dan
penolakan akses pada resource yang ditemukan. Pembacaan berbasis consent
memuat `basis: "consent"`, `consent_id`, serta `consent_expires_at` pada ringkasan
audit pasien, tanpa nilai hasil. Field context tambahan ikut HMAC canonical v1,
sehingga rantai episode/lab yang sudah ada tetap dapat diverifikasi.

### Alur integrasi frontend consent

1. Selesaikan alur tes lab; pasien memiliki ID hasil dari permintaan miliknya.
2. Dokter lain meminta izin dengan kode pasien, ID hasil, dan tujuan.
3. Pasien GET `/consents?status=pending`, meninjau dokter, klinik, tujuan, dan tes.
4. Pasien PATCH approve dengan durasi pilihan, atau deny. Refresh daftar/detail.
5. Dokter GET `/consents?status=active`, lalu GET `/lab-results/{resource_id}`.
6. Pasien revoke; permintaan baca dokter berikutnya harus `403`.
7. Uji batas kedaluwarsa, dokter lain, hasil lain, dan pindah klinik. Bersihkan
   data hasil dari tampilan ketika mendapat `403`, `401`, atau `503`.
8. Pasien periksa riwayat akses dan verifikasi rantai setelah aktivitas tersebut.

UI consent dan notifikasi tersedia untuk pasien/dokter. Persetujuan satu jenis
resource dan koreksi hasil belum tersedia; consent tetap untuk satu hasil lab.
Tes backend: `go test ./internal/care -run Consent -v` dengan
`TEST_DATABASE_URL` lokal; setiap fixture menggunakan schema terpisah.
