# Akses darurat backend

Memerlukan migrasi 8 dan sesi dokter klinik. Semua mutasi wajib CSRF.

| Method | Endpoint `/api/v1` | Kontrak |
| --- | --- | --- |
| POST | `/break-glass` | `{patient_ref,reason_code,reason_text}` → 201 `{session}` |
| GET | `/break-glass?limit=20&offset=0` | Riwayat milik pasien atau dokter+klinik sesi |
| GET | `/break-glass/{id}/resources` | Pagination yang sama; `{resources:[{id,resource_type}]}` |
| GET | `/admin/break-glass` | Metadata/alasan untuk peninjauan; pagination yang sama |
| PATCH | `/admin/break-glass/{id}` | `{decision,note}` → `{session}`; satu tinjauan |

Kode alasan: `unconscious`, `unreachable_patient`, `life_threatening`.
Teks alasan dan catatan tinjauan 10–1000 karakter. Keputusan:
`appropriate` atau `needs_followup`. Tinjauan tidak memperpanjang grant.
Sesi berlaku tepat 60 menit; start duplikat yang masih hidup menghasilkan 409.
Grant terikat dokter dan organisasi saat start. Pasien melihat riwayatnya;
admin melihat metadata, alasan dan catatan, tanpa isi hasil lab/resep.

Directory hanya menampilkan ID hasil lab selesai dan resep berstatus
sent/accepted/preparing/ready. Gunakan endpoint detail lab/resep yang sudah ada.
Grant diperiksa di setiap detail, termasuk waktu setelah menunggu row lock.
Tidak memberi hak perubahan, daftar pasien global, atau akses resep selesai.

Start, tinjauan, directory dan pembacaan klinis diaudit pada rantai HMAC pasien.
Context berisi ID serta expiry grant; teks alasan tidak masuk audit/notifikasi.
Start memberi notifikasi pasien dan admin; tinjauan memberi notifikasi pasien
dan dokter pemula. Kegagalan audit/notifikasi membatalkan transaksi.

Status gagal: 400 input, 401 sesi, 403 akses/resource tidak dikenal, 409 status,
503 infrastruktur. Daftar metadata milik sendiri tidak menambah audit.
