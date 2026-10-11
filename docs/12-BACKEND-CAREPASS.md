# CarePass dan QR

Memerlukan migrasi 9. Sesi dan CSRF wajib untuk mutasi.

| Metode | Endpoint `/api/v1` | Kontrak |
| --- | --- | --- |
| POST | `/carepasses` | Pasien: `{resource_type,resource_id,ttl_minutes}` → 201 |
| GET | `/carepasses?limit=20&offset=0` | Pasien: `{carepasses:[metadata]}` |
| POST | `/carepasses/redeem` | Dokter klinik: `{token}` → `{carepass:metadata}` |
| PATCH | `/carepasses/{id}` | Pasien pemilik: `{action:"revoke"}` |

Create menerima `lab_result` selesai atau `prescription` milik pasien, satu
resource per pass, TTL 15 atau 60 menit. Respons `{carepass,token,share_url,
qr_png_data_uri}` memuat token acak 256-bit dan QR PNG 384×384 hanya sekali.
Database menyimpan SHA-256 token. Daftar, audit, notifikasi dan error tidak
memuat token, QR, atau isi klinis. Simpan respons create hanya selama dibutuhkan.

`PUBLIC_APP_URL` adalah origin aplikasi, default development
`http://localhost:5173`; production wajib origin HTTPS. QR berisi
`<origin>/carepass#token=<token>`. Fragment tidak dikirim ke access log HTTP.
Frontend nantinya membaca fragment, menghapusnya dari address bar dan mengirim
token dalam JSON ke endpoint redeem; route frontend bukan bagian PR backend.
PNG dibuat dengan [go-qrcode](https://github.com/skip2/go-qrcode).

Penebus pertama mengikat grant ke dokter dan klinik saat redeem. Penebusan ulang
oleh dokter+klinik yang sama idempotent; pengguna lain ditolak. Penebusan bukan
akses anonim. Metadata mengembalikan ID resource; detail memakai endpoint lab
atau resep yang ada. CarePass tidak mengizinkan perubahan atau resource lain.
Berbeda dari akses darurat, pasien boleh membagikan resep historis secara eksplisit.

Expiry dan revoke diperiksa setelah row lock di setiap pembacaan; perpindahan
klinik membatalkan kecocokan grant. Revoke idempotent, tanpa menghidupkan token
kembali. Izin episode/consent yang terpisah tetap berlaku jika masih sah.
Audit HMAC mencatat create/redeem/revoke dan pembacaan klinis dengan ID/expiry
CarePass. Redeem memberi notifikasi pasien; revoke memberi notifikasi dokter.
Mutasi dan audit/notifikasi satu transaksi, termasuk saat kegagalan infrastruktur.

400 input tidak valid; 401 sesi; 403 scope/token/resource tidak dikenal atau
kedaluwarsa; 503 infrastruktur. Pagination limit 1–100, offset 0–10000.
