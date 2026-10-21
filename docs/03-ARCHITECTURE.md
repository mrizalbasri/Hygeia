# Hygeia — System Architecture

| | |
|---|---|
| Versi | 0.2 (draft lomba) |
| Dokumen terkait | `01-BRD.md`, `02-PRD.md` |
| Stack default | SvelteKit + TypeScript, Go, PostgreSQL, REST, Docker (lihat bagian 14 untuk keputusan stack) |

---

## 1. Tujuan Arsitektur

- Alur multi-peran mudah dibangun dan didemokan.
- **Otorisasi dan consent ditegakkan di satu tempat** di backend, bukan tersebar di handler.
- Audit ditulis bersama aksi yang diauditnya dan integritasnya bisa diverifikasi.
- Batas domain jelas tanpa biaya operasional microservices.
- API bersih untuk menunjukkan interoperabilitas dan memungkinkan integrasi di masa depan.

## 2. Teknologi

| Lapisan | Pilihan | Alasan |
|---|---|---|
| Frontend | SvelteKit + TypeScript | Dashboard per peran, routing, form; TypeScript menjaga kontrak API. |
| UI | Tailwind CSS | Implementasi cepat dan konsisten. |
| Backend | Go (router `chi`, akses DB `sqlc` + `pgx`, migrasi `golang-migrate`) | Servis API kecil, deployment sederhana, query SQL eksplisit yang mudah dijelaskan ke juri. |
| API | REST/JSON, versi `/api/v1` | Mudah didokumentasikan (OpenAPI) dan diuji. |
| Database | PostgreSQL | Relasi, constraint, transaksi, row locking untuk rantai audit. |
| Container | Docker Compose | Lingkungan dev dan demo sama. |
| Reverse proxy | Caddy atau Nginx | Satu domain: `/` ke web, `/api` ke Go, sehingga cookie sesi bekerja tanpa CORS. |
| Opsional | Redis, object storage, SSE | Hanya jika fitur Could dikerjakan. |

## 3. Arsitektur Logis

```
 Browser
    │ HTTPS
    ▼
 ┌────────────────────┐
 │ Reverse proxy      │  /      → web
 │ (Caddy / Nginx)    │  /api/* → api
 └───────┬────────────┘
         │
   ┌─────┴───────────────┐        ┌──────────────────────────────┐
   │ SvelteKit web       │  REST  │ Go API (modular monolith)    │
   │ Pasien / Dokter /   │───────▶│ ┌──────────────────────────┐ │
   │ Lab / Apotek / Admin│        │ │ HTTP layer (chi)         │ │
   └─────────────────────┘        │ │  authn · request-id · log│ │
                                  │ ├──────────────────────────┤ │
                                  │ │ policy  ← satu pintu     │ │
                                  │ │ otorisasi + audit        │ │
                                  │ ├──────────────────────────┤ │
                                  │ │ auth · users · orgs      │ │
                                  │ │ care · laboratory        │ │
                                  │ │ prescription · consent   │ │
                                  │ │ breakglass · audit       │ │
                                  │ │ notification             │ │
                                  │ └───────────┬──────────────┘ │
                                  └─────────────┼────────────────┘
                                                │ SQL / transaksi
                                                ▼
                                         ┌─────────────┐
                                         │ PostgreSQL  │
                                         └─────────────┘
```

Aturan yang dijaga: semua akses data terlindungi **harus** lewat paket `policy`. Handler tidak boleh membaca tabel `lab_results` atau `prescriptions` langsung tanpa memanggilnya.

## 4. Modul Backend

Tahap laboratorium sudah diimplementasikan dengan kolom hasil terstruktur
(`value`, `unit`, `reference_range`, `flag`, `note`), satu hasil immutable per
permintaan. Blueprint `payload jsonb` di bagian model awal di bawah belum
dipakai. Kontrak aktual mengikuti `05-BACKEND-API.md` dan migrasi SQL.
Penugasan lab yang sudah disetujui pasien tetap dapat diselesaikan setelah
episode diakhiri; hanya permintaan yang belum disetujui yang dibatalkan.

Consent sudah tersedia untuk satu `lab_result` dan tindakan `view`. Izin terikat
pada dokter peminta serta kliniknya, dengan durasi 1 jam, 24 jam, atau 7 hari.
Pasien dapat menolak atau mencabutnya. Scope jenis resource secara keseluruhan,
dan resep dalam blueprint consent berikut belum diterapkan. Akses darurat
tersedia terpisah melalui grant dokter+klinik 60 menit (migrasi 8). Pembacaan
hasil mengunci consent hingga audit selesai dan mengecek waktu setelah menunggu
kunci, agar pencabutan dan kedaluwarsa tidak dilangkahi.

Modul resep sudah tersedia dengan tabel `prescriptions` dan `prescription_items`
(migrasi 5). Dokter episode aktif menerbitkan resep, pasien memilih apotek
sekaligus menyetujui pengiriman, kemudian apotek memproses sampai selesai.
Resep yang belum disetujui dibatalkan saat episode berakhir; penugasan yang sudah
disetujui tetap dapat diselesaikan. Apotek hanya menerima ID, referensi pasien,
item obat, dan status selama penugasan aktif. Tidak ada consent lintas dokter
untuk resep pada implementasi ini. Kontrak aktual: `07-BACKEND-PRESCRIPTIONS.md`.

Timeline perjalanan pasien menggunakan proyeksi allowlist kejadian bisnis
sukses dari audit yang sudah ada. Modul `timeline` menyediakan query dan
presentasi milestone; service `care` menyelesaikan identitas pasien dari sesi.
Tidak ada tabel timeline, dual-write, backfill, atau perubahan format hash.
Riwayat akses tetap mencakup pembacaan dan penolakan; timeline mengecualikannya.
Urutan memakai seq rantai dengan cursor eksklusif, bukan offset. Kontrak aktual:
`08-BACKEND-TIMELINE.md`.

Tahap penyelesaian backend (12 Oktober 2026) menambah admin akun/organisasi,
ledger administratif append-only, notifikasi transaksional, akses darurat,
CarePass/QR untuk satu resource dan stream SSE. CarePass hanya menyimpan hash
token; penebus pertama mengikat grant ke dokter+klinik. Policy di endpoint
klinis memeriksa grant dengan row lock dan waktu setelah lock. Detail kontrak
ada pada dokumen 09–13. SSE menggunakan polling PostgreSQL tanpa Redis;
alokasi notifikasi diserialisasi sampai commit untuk cursor yang tidak melewatkan
transaksi lebih lambat. Router aktual adalah `net/http`; diagram `chi` dan
blueprint di bawah menggambarkan rancangan awal, bukan dependensi implementasi.

Keputusan implementasi MVP (11 Oktober 2026): jalur care episode memberi akses
kepada dokter pengundang yang masih berada di klinik episode, bukan seluruh
dokter klinik. Undangan memakai kode pasien persis. Metadata undangan/riwayat
tersedia bagi peserta; akses klinis jalur A tetap memerlukan episode aktif.
Lihat `06-BACKEND-CARE.md` untuk detail implementasi tahap care dan audit.

| Modul | Tanggung jawab |
|---|---|
| `auth` | Login/logout, sesi, hash password, konteks identitas. |
| `users` | Profil pengguna dan peran. |
| `organizations` | Klinik, lab, apotek, keanggotaan staf. |
| `care` | Care episode (undangan, terima, tolak, akhiri). |
| `laboratory` | Permintaan lab, status, hasil terstruktur. |
| `prescription` | Resep, item, alur apotek. |
| `consent` | Permintaan consent lintas organisasi, keputusan pasien, pencabutan, kedaluwarsa. |
| `breakglass` | Sesi akses darurat dan tinjauan admin. |
| `policy` | **Keputusan akses**: menggabungkan peran, organisasi, relasi, consent, akses darurat, lalu memanggil audit. |
| `audit` | Penulisan event berantai hash, verifikasi, pembuatan ringkasan kalimat. |
| `notification` | Notifikasi dalam aplikasi. |

## 5. Struktur Proyek

```
hygeia/
├─ backend/
│  ├─ cmd/api/main.go
│  ├─ internal/
│  │  ├─ auth/  users/  organizations/  care/
│  │  ├─ laboratory/  prescription/
│  │  ├─ consent/  breakglass/
│  │  ├─ policy/        # Authorize(), basis, projection per peran
│  │  ├─ audit/         # chain.go, verify.go, summary.go
│  │  └─ notification/
│  ├─ pkg/ database/  middleware/  response/  validator/
│  ├─ db/ migrations/  queries/   # SQL untuk sqlc
│  ├─ scripts/ seed.sql  demo-tamper.sql
│  ├─ openapi.yaml
│  └─ Dockerfile
├─ frontend/src/
│  ├─ lib/ api/  components/  stores/  types/  utils/
│  ├─ routes/
│  │  ├─ login/
│  │  ├─ patient/   journey/  episodes/  approvals/  consents/  access-history/
│  │  ├─ doctor/    patients/  lab-requests/  prescriptions/  access/
│  │  ├─ lab/       requests/
│  │  ├─ pharmacy/  prescriptions/
│  │  └─ admin/     organizations/  accounts/  audit-status/  break-glass-review/
│  └─ hooks.server.ts
├─ docker-compose.yml
└─ docs/  01-BRD.md  02-PRD.md  03-ARCHITECTURE.md
```

## 6. Model Data

```
users(id, email unique, password_hash, full_name, role, created_at, disabled_at)
   role ∈ patient | doctor | lab | pharmacy | admin

organizations(id, type, name)                     type ∈ clinic | laboratory | pharmacy
organization_memberships(user_id, organization_id, staff_role)   PK(user_id, organization_id)

patients(id, user_id unique, patient_ref unique, sex, birth_year)
   -- lab/apotek menerima patient_ref, sex, usia; bukan nama lengkap

care_episodes(id, patient_id, doctor_user_id, organization_id,
              status, invited_at, accepted_at, ended_at)
   status ∈ invited | active | declined | ended

lab_requests(id, episode_id, lab_org_id, test_code, test_name, clinical_note,
             status, created_by, created_at, patient_approved_at)
   status ∈ pending_patient_approval | sent | accepted | in_progress | completed | cancelled

lab_results(id, lab_request_id unique, payload jsonb, submitted_by, submitted_at)
   payload: { items:[{name, value, unit, ref_range, flag}], note }

prescriptions(id, episode_id, created_by, pharmacy_org_id null,
              status, created_at, patient_approved_at)
   status ∈ awaiting_pharmacy_choice | sent | received | preparing | ready | completed | cancelled
prescription_items(id, prescription_id, drug_name, dose, frequency, duration, instructions)

consents(id, patient_id, grantee_user_id, grantee_org_id null,
         resource_type, resource_id null, allowed_action,
         purpose, status, requested_at, decided_at, expires_at, revoked_at)
   resource_type ∈ lab_result | prescription ; allowed_action = view
   status ∈ pending | active | denied | revoked | expired

break_glass_sessions(id, doctor_user_id, patient_id, reason_code, reason_text,
                     starts_at, expires_at, reviewed_by null, reviewed_at null, review_note null)

audit_chain_heads(chain_id PK, last_seq, last_hash)
audit_events(id, chain_id, seq, actor_user_id, actor_role, action,
             resource_type, resource_id, patient_id, outcome, basis jsonb,
             context jsonb, request_id, created_at, prev_hash, hash)
   UNIQUE(chain_id, seq) ; outcome ∈ allow | deny

notifications(id, user_id, type, payload jsonb, read_at, created_at)
```

Catatan:
- `consents.status = expired` hanya label; **pemeriksaan akses selalu membandingkan `expires_at` dengan waktu sekarang**, tidak bergantung pada job latar belakang.
- Aplikasi memakai role database yang **tidak punya `UPDATE`/`DELETE`** pada `audit_events`, ditambah trigger yang menolak keduanya. Pengubahan hanya mungkin lewat role administrasi database, itulah yang terdeteksi oleh verifikasi hash.
- Pemetaan ke FHIR (di luar MVP): `lab_results` → `Observation`, `prescriptions` → `MedicationRequest`, `care_episodes` → `EpisodeOfCare`, `consents` → `Consent`.

## 7. Model Otorisasi

### 7.1 Pipeline

Setiap permintaan ke resource terlindungi melewati `policy.Authorize(actor, action, resource)`:

```
 1. Autentikasi        → siapa pengguna ini?
 2. Gerbang peran      → peran boleh melakukan aksi ini?
 3. Gerbang organisasi → staf bertindak dalam organisasi yang relevan?
 4. Jalur akses (urutan, yang pertama cocok dicatat sebagai basis):
      A. care episode aktif milik aktor
      B. penugasan yang disetujui pasien (lab atau apotek tujuan)
      C. consent aktif dan belum kedaluwarsa untuk resource ini
      D. sesi akses darurat aktif untuk pasien ini
 5. Proyeksi field     → kembalikan hanya field yang diizinkan peran + basis
 6. Audit              → tulis event (allow atau deny) dengan basis
 Gagal di langkah mana pun → 403 generik, tanpa isi resource.
```

### 7.2 Matriks akses (ringkas)

| Resource | Pasien | Dokter pembuat | Dokter lain | Lab | Apotek | Admin |
|---|---|---|---|---|---|---|
| Permintaan lab | milik sendiri | A | tidak | B (ditugaskan + disetujui) | tidak | tidak |
| Hasil lab | milik sendiri | A | C atau D | menulis (untuk permintaan yang ditugaskan) | tidak | tidak |
| Resep | milik sendiri | A | C atau D | tidak | B (dipilih pasien) | tidak |
| Audit | rantai sendiri | tidak | tidak | tidak | tidak | metadata operasional, tanpa isi klinis |

### 7.3 Proyeksi field

- Lab melihat: `patient_ref`, jenis kelamin, usia, `test_code`, `test_name`, catatan klinis seperlunya. Tidak ada nama lengkap atau riwayat lain.
- Apotek melihat: `patient_ref`, item resep. Tidak ada hasil lab atau episode.
- Akses darurat: hasil lab dan resep aktif saja, tanpa catatan klinis bebas.
- Admin: metadata organisasi, akun, status rantai, dan tinjauan akses darurat (alasan, waktu), tanpa isi hasil atau resep.

### 7.4 State machine consent

```
 pending ──setuju──▶ active ──cabut──▶ revoked
    │                  │
    └─tolak─▶ denied   └─waktu habis─▶ expired
```

### 7.5 State machine permintaan lab

```
 pending_patient_approval ──setuju──▶ sent ──▶ accepted ──▶ in_progress ──▶ completed
          │
          └──tolak / dibatalkan──▶ cancelled
```

Resep serupa: `awaiting_pharmacy_choice → sent → received → preparing → ready → completed`.

## 8. Audit Berantai Hash

### 8.1 Tujuan dan batas klaim

Tujuan: manipulasi atau penghapusan event audit **terdeteksi** saat verifikasi. Ini bukan klaim "tidak bisa diubah": siapa pun yang menguasai database **dan** kunci HMAC dapat menulis ulang seluruh rantai. Mitigasi opsional di bagian 8.5.

### 8.2 Struktur

- Satu rantai per pasien (`chain_id = patient_id`) untuk event yang menyangkut pasien itu. Event sistem (login, admin) masuk rantai `system`.
- Tiap event:
  `hash = HMAC-SHA256(AUDIT_HMAC_KEY, prev_hash ‖ canonical(event))`
- `canonical(event)`: JSON dengan urutan field tetap, kunci objek diurutkan, waktu UTC format RFC 3339 dengan presisi mikrodetik. Event pertama memakai `prev_hash` nol.
- `context` menyimpan *snapshot* nama aktor dan organisasi saat kejadian agar ringkasan kalimat stabil dan ikut terhash.

### 8.3 Penulisan (dalam transaksi yang sama dengan aksinya)

```
BEGIN
  -- aksi bisnis (misalnya membaca hasil)
  SELECT last_seq, last_hash FROM audit_chain_heads
    WHERE chain_id = $1 FOR UPDATE            -- serialisasi per rantai
  seq  = last_seq + 1
  hash = HMAC(key, last_hash ‖ canonical(event))
  INSERT INTO audit_events (...)
  UPDATE audit_chain_heads SET last_seq = seq, last_hash = hash
COMMIT
```

- **Gagal tertutup:** jika penulisan audit gagal, transaksi dibatalkan dan data tidak dikembalikan.
- Event `deny` ditulis di transaksi terpisah agar tetap tercatat meski aksi dibatalkan.
- Kunci `FOR UPDATE` per rantai membatasi konkurensi hanya untuk pasien yang sama, tidak global.

### 8.4 Verifikasi

`GET /api/v1/audit/verify?chain=<id>` menghitung ulang hash dari `seq = 1` dan mengembalikan:

```json
{ "chain": "…", "events": 128, "intact": true }
```

atau `{ "intact": false, "first_mismatch_seq": 57 }`. Pasien dapat memverifikasi rantainya sendiri; admin dapat memverifikasi semua rantai dan melihat ringkasannya di layar Status Audit.

### 8.5 Penguatan opsional

- Tampilkan `last_hash` di layar pasien agar dapat dicatat atau di-screenshot.
- Ekspor hash kepala rantai secara berkala ke tempat di luar database (berkas yang ditandatangani atau repositori terpisah).

### 8.6 Ringkasan kalimat

Tiap event dirender dari template per `action` dan `basis`, misalnya:

- `lab_result.view` + A → "Dr. Maya melihat hasil Gula Darah pada 14:02 karena episode perawatan Anda aktif."
- `lab_result.view` + C → "Dr. Budi melihat hasil Gula Darah pada 15:10 dengan izin Anda, berlaku sampai 11 Oktober 15:00."
- `lab_result.view` + D → "**Akses darurat:** Dr. Rina melihat hasil Gula Darah pada 16:45. Alasan: pasien tidak sadar."
- `lab_result.view` outcome deny → "Dr. Budi mencoba melihat hasil Gula Darah pada 15:05 dan ditolak."

## 9. Akses Darurat (Break-glass)

Alur:
1. Dokter memilih pasien dan menekan "Akses Darurat".
2. Wajib memilih `reason_code` (misalnya `unconscious`, `unreachable_patient`, `life_threatening`) dan menulis teks alasan.
3. Server membuat `break_glass_sessions` dengan `expires_at = now() + BREAK_GLASS_TTL` (default 60 menit).
4. Dalam sesi itu, jalur D di `policy` mengizinkan `view` untuk hasil lab dan resep aktif pasien tersebut saja.
5. Pasien menerima notifikasi segera; event audit bertanda `basis.path = "D"`.
6. Sesi berakhir otomatis; admin dapat meninjau dan memberi catatan (`reviewed_*`).

Batasan yang perlu dinyatakan jujur: pada sistem nyata, akses darurat juga membutuhkan kebijakan organisasi dan sanksi; di prototipe, kontrol yang ada adalah alasan wajib, batas waktu, cakupan sempit, notifikasi, dan jejak audit.

## 10. Contoh Permukaan API

Semua di bawah `/api/v1`. Kesalahan memakai bentuk `{ "error": { "code", "message", "request_id" } }`; `403` selalu generik.

| Metode | Endpoint | Fungsi |
|---|---|---|
| POST | `/auth/login` · `/auth/logout` | Autentikasi. |
| GET | `/me` | Identitas, peran, organisasi. |
| POST | `/episodes` | Dokter mengundang pasien. |
| PATCH | `/episodes/{id}` | Pasien terima/tolak; pihak mana pun mengakhiri. |
| POST | `/lab-requests` | Dokter membuat permintaan lab. |
| GET | `/lab-requests` | Daftar sesuai aktor dan organisasi. |
| POST | `/lab-requests/{id}/approval` | Pasien setuju/tolak pengiriman ke lab. |
| PATCH | `/lab-requests/{id}/status` | Lab memperbarui status. |
| POST | `/lab-requests/{id}/result` | Lab mengirim hasil terstruktur. |
| GET | `/lab-results/{id}` | Hasil terlindungi (jalur A/C/D). |
| POST | `/prescriptions` | Dokter membuat resep. |
| POST | `/prescriptions/{id}/pharmacy` | Pasien memilih apotek. |
| PATCH | `/prescriptions/{id}/status` | Apotek memperbarui status. |
| GET | `/prescriptions/{id}` | Resep terlindungi. |
| POST | `/consents` | Dokter lain meminta akses. |
| PATCH | `/consents/{id}` | Pasien setuju (dengan durasi), tolak, cabut. |
| POST | `/break-glass` | Memulai akses darurat. |
| GET | `/patient/journey` | Timeline pasien. |
| GET | `/patient/access-summary` | Riwayat akses dalam kalimat. |
| GET | `/audit/verify` | Verifikasi integritas rantai. |
| GET | `/admin/break-glass` · PATCH `/admin/break-glass/{id}` | Tinjauan akses darurat. |
| GET | `/notifications` | Notifikasi. |
| GET | `/healthz` | Pemeriksaan kesehatan. |

Opsional (Could): `POST /carepass` dan `GET /share/{token}`, dengan token disimpan sebagai hash.

## 11. Baseline Keamanan

- Password: Argon2id (atau bcrypt dengan cost memadai).
- HTTPS di demo; cookie sesi `HttpOnly`, `Secure`, `SameSite=Lax`; token CSRF untuk semua mutasi.
- Otorisasi sisi server di setiap endpoint terlindungi; peran dari klien tidak pernah dipercaya.
- Validasi input dan query terparameter (`sqlc`/`pgx`).
- Batas laju pada login dan endpoint sensitif.
- Kunci dan rahasia (`SESSION_SECRET`, `AUDIT_HMAC_KEY`) hanya di variabel lingkungan, bukan di repositori.
- Enkripsi payload hasil lab di level aplikasi (AES-256-GCM) bersifat opsional; kunci dari lingkungan.
- Data demo sintetis.

## 12. Deployment

```
 Browser ─HTTPS─▶ Caddy ─┬─▶ web (SvelteKit, adapter-node)
                         └─▶ api (Go) ──▶ PostgreSQL
```

Variabel lingkungan: `DATABASE_URL`, `SESSION_SECRET`, `AUDIT_HMAC_KEY`, `APP_ENV`, `COOKIE_DOMAIN`, `BREAK_GLASS_TTL`, `CONSENT_DEFAULT_TTL`. Satu `docker-compose.yml` menjalankan semuanya; `make seed` memuat data sintetis; `make demo-reset` mengembalikan keadaan awal demo.

## 13. Mengapa Modular Monolith

Beban dan ukuran tim tidak membenarkan microservices. Modular monolith memberi pengembangan lebih cepat, transaksi sederhana (penting untuk audit yang ditulis bersama aksinya), dan lebih sedikit titik gagal saat penjurian. Batas modul tetap memungkinkan ekstraksi di masa depan.

## 14. Keputusan Terbuka

| Keputusan | Default | Catatan |
|---|---|---|
| **Stack** | SvelteKit + Go + PostgreSQL | **Putuskan sebelum Phase 1.** Uji satu hari: login, satu endpoint terlindungi, dan satu migrasi di Go. Jika belum selesai dalam sehari, pindah ke Next.js (TypeScript) + PostgreSQL + Prisma/Drizzle; arsitektur logis di dokumen ini tetap berlaku. |
| Router Go | `chi` | Pakai yang sudah dikuasai tim. |
| Akses DB | `sqlc` + `pgx` | Query eksplisit, mudah dijelaskan. |
| Sesi | Cookie sesi server-side | Token hanya jika ada klien non-browser di demo. |
| Format hasil lab | Terstruktur (JSON) | PDF/object storage hanya jika alur inti sudah stabil. |
| Realtime | Tidak di MVP | SSE untuk notifikasi jika waktu tersisa. |
| `BREAK_GLASS_TTL` | 60 menit | Dapat dikonfigurasi. |
| Pilihan durasi consent | 1 jam, 24 jam, 7 hari | Dapat dikonfigurasi. |
| Bahasa antarmuka | Indonesia | Istilah teknis tetap Inggris jika lazim. |

## 15. Strategi Pengujian

| Jenis | Isi |
|---|---|
| Matriks otorisasi | Tes berbasis tabel: setiap kombinasi peran × resource × jalur menghasilkan allow atau 403; respons deny tidak memuat nilai hasil. |
| Consent | Dengan jam yang dapat disuntik: kedaluwarsa, pencabutan, penolakan memblokir akses pada permintaan berikutnya. |
| Audit | Setiap jalur menulis event; verifikasi lulus pada data normal; gagal dan menunjuk `seq` yang benar setelah satu baris diubah. |
| Konkurensi audit | Banyak pembacaan paralel pada pasien yang sama menghasilkan `seq` berurutan tanpa celah atau duplikat. |
| Akses darurat | Tanpa alasan ditolak; kedaluwarsa otomatis; notifikasi terkirim; tertandai di audit. |
| End-to-end | Skenario demo (PRD bagian 8) otomatis dengan Playwright pada data seed. |
| Keamanan dasar | Akses tanpa sesi, IDOR antar pasien/organisasi, CSRF pada mutasi, laju login. |

## 16. Peta Jalan Implementasi

Urutan sengaja menempatkan lapisan akses dan audit **sebelum** pemolesan UI karena itulah pembeda Hygeia.

| Fase | Hasil |
|---|---|
| 0 — Keputusan | Uji stack satu hari, putuskan, baca guidebook, finalkan nama dan ejaan. |
| 1 — Fondasi | Repositori, Docker, migrasi, pengguna, organisasi, login, routing per peran. |
| 2 — Alur inti | Care episode, permintaan lab, alur lab, hasil terstruktur. |
| 3 — Lapisan akses | Paket `policy`, penugasan disetujui pasien (jalur B), consent lintas organisasi (jalur C). |
| 4 — Audit | Rantai hash, penulisan transaksional, verifikasi, ringkasan kalimat. |
| 5 — Resep | Resep, pilihan apotek oleh pasien, alur apotek. |
| 6 — Akses darurat dan admin | Sesi darurat, notifikasi, tinjauan admin. |
| 7 — Pengalaman | Perjalanan pasien, notifikasi, pemolesan UI, aksesibilitas. |
| 8 — Kesiapan lomba | Data seed, tes otomatis, deployment, skrip demo, skrip manipulasi audit, tangkapan layar dan diagram untuk proposal. |
| Opsional | CarePass, SSE. |

## 17. Artefak Berikutnya

1. ERD terperinci dengan constraint dan indeks.
2. `openapi.yaml` dengan skema request/response.
3. Wireframe empat dashboard utama, layar consent, dan layar riwayat akses.
4. Rencana tes (bagian 15) menjadi daftar kasus.
5. Diagram arsitektur dan alur pengguna untuk proposal.
6. Skrip demo tertulis (versi 3 menit dan 8 menit).
