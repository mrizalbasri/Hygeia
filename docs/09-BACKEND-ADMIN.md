# Admin minimal

Semua endpoint `/api/v1/admin` hanya menerima sesi admin aktif; mutasi memakai
CSRF dan JSON ketat. Daftar memakai `limit` 1–100 (default 20), `offset` 0–10000.

| Metode | Path setelah `/api/v1` | Input / hasil |
|---|---|---|
| GET | `/admin/users` | `{users:[],limit,offset}`; ID, email, nama, role, organisasi, referensi pasien, disabled_at |
| POST | `/admin/users` | `email,full_name,password,role,organization_id?` → `201 {user}` |
| PATCH | `/admin/users/{id}` | `full_name?`, `disabled?` → `{user}` |
| GET | `/admin/organizations` | `{organizations:[],limit,offset}`; ID,nama,type |
| POST | `/admin/organizations` | `name,type` → `201 {organization}` |
| PATCH | `/admin/organizations/{id}` | `name` → `{organization}` |
| GET | `/admin/audit-chains` | `{chains:[],limit,offset}`; chain_id,patient_ref,events |
| GET | `/admin/activity` | `{activity:[],limit,offset}`; actor_id,action,resource_id,occurred_at |

Organisasi bertipe clinic/laboratory/pharmacy. Akun doctor/lab/pharmacy wajib
organisasi yang cocok; patient/admin tidak memiliki organisasi. Nama 1–200
karakter; password 12–1024 byte di-hash Argon2id dan tidak dikembalikan. Pasien
mendapat referensi acak. Email lowercase dan unik. Role dan penugasan organisasi
tidak dapat diubah melalui PATCH; transfer staf/password reset tidak tersedia.
Admin tidak dapat menonaktifkan dirinya. Disable menghapus semua sesi pengguna;
enable tidak menghidupkan sesi lama. Mutasi admin diserialisasi dan identitas
admin diperiksa ulang setelah lock, sehingga admin yang dinonaktifkan ketika
menunggu tidak dapat meneruskan mutasi.

Admin dapat memakai `GET /audit/verify?chain=UUID` yang sudah ada. Daftar kepala
rantai bukan hasil verifikasi; `events` hanya jumlah terakhir. Admin tetap
ditolak pada detail hasil lab dan resep. Metadata identitas tidak memberi akses
klinis. Tidak ada password/hash, hasil lab, atau obat dalam respons admin.

Migrasi 6 menambah `admin_activity`, ledger administratif append-only yang
ditulis atomik bersama mutasi. Ledger ini tidak memakai HMAC pasien karena
administrasi organisasi/akun tidak terikat satu pasien. Trigger memblokir
UPDATE/DELETE/TRUNCATE biasa; pemilik database tetap dapat menonaktifkan trigger.
Kegagalan ledger membatalkan akun/organisasi/sesi yang diubah. Error mengikuti
kontrak umum 400/401/403/409/415/503 tanpa detail SQL atau kredensial.

Jalankan `go run ./cmd/migrate up` sebelum API ini dipakai. Tes khusus:
`go test ./internal/care -run Admin -v` dengan `TEST_DATABASE_URL`.
