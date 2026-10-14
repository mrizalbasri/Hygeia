# Notifikasi dalam aplikasi

Migrasi 7 menambah notifications. Semua akun terautentikasi dapat mengakses
hanya notifikasi sendiri; tidak ada selector user/patient.

| Metode | Endpoint `/api/v1` | Input / hasil |
|---|---|---|
| GET | `/notifications` | `limit` 1–100 default 20, `before_id` cursor eksklusif positif, `unread=true/false`; `{notifications:[],limit,next_before_id}` |
| GET | `/notifications/unread-count` | `{unread_count}` |
| PATCH | `/notifications/{id}` | CSRF + `{"read":true}` → `{notification}`; idempotent |

Notifikasi berisi id, action, resource_type, resource_id, message, created_at,
read_at. Tidak ada nilai lab, obat/dosis, tujuan consent, alasan darurat, token,
atau context audit. Render message sebagai teks. ID lain/tidak ada menghasilkan
403. Query tak dikenal/duplikat atau cursor salah menghasilkan 400.

Milestone sukses episode/lab/resep/consent memilih pasien, dokter episode aktif
yang masih anggota kliniknya, serta staf penyedia tujuan selama assignment
aktif; keputusan consent diteruskan ke dokter peminta. Aktor sendiri dan akun
nonaktif dikecualikan. Lab/apotek tidak dinotifikasi sebelum pasien menyetujui.
Daftar memakai ID menurun; cursor stabil saat notifikasi baru masuk.

Audit writer menambahkan notifikasi dalam transaksi bisnis yang sama; kegagalan
notifikasi membatalkan bisnis dan audit. Unique(user_id,source_chain,source_seq)
mencegah duplikasi. Read/list, outcome deny, dan action tak dikenal tidak membuat
notifikasi. Notifikasi tidak backfill dari audit lama dan tidak otomatis dibaca
ketika daftar dibuka. Ledger HMAC pasien tetap sumber verifikasi integritas.

Dependensi PR: branch admin (migrasi 6) sebelum branch ini (migrasi 7).
Tes: `go test ./internal/care -run Notification -v` dengan TEST_DATABASE_URL.
