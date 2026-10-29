# Dokumentasi Hygeia

Mulai dari [README proyek](../README.md) untuk gambaran aplikasi dan development.
Indeks ini mengelompokkan dokumen produk, implementasi, dan catatan verifikasi.

## Produk dan desain

| Dokumen                                 | Isi                                           |
| --------------------------------------- | --------------------------------------------- |
| [Business Requirements](01-BRD.md)      | Kebutuhan bisnis dan lingkup masalah.         |
| [Product Requirements](02-PRD.md)       | Kebutuhan produk dan alur pengguna.           |
| [Arsitektur sistem](03-ARCHITECTURE.md) | Rancangan komponen dan hubungan antarlayanan. |
| [Design system](04-DESIGN.md)           | Acuan visual dan antarmuka.                   |

## Kontrak dan implementasi backend

Semua path endpoint API menggunakan prefix `/api/v1`.

| Dokumen                                         | Modul                                              |
| ----------------------------------------------- | -------------------------------------------------- |
| [Kontrak API](05-BACKEND-API.md)                | Request, response, error, dan integrasi frontend.  |
| [Care episode dan audit](06-BACKEND-CARE.md)    | Perawatan, kebijakan akses, dan rantai audit.      |
| [Resep dan apotek](07-BACKEND-PRESCRIPTIONS.md) | Persetujuan pasien dan lifecycle penugasan apotek. |
| [Timeline pasien](08-BACKEND-TIMELINE.md)       | Riwayat kejadian, filter, dan pagination.          |
| [Admin](09-BACKEND-ADMIN.md)                    | Akun, organisasi, dan ledger aktivitas.            |
| [Notifikasi](10-BACKEND-NOTIFICATIONS.md)       | Inbox dan status baca.                             |
| [Akses darurat](11-backend-emergency-api.md)    | Grant darurat dan tinjauan admin.                  |
| [CarePass](12-BACKEND-CAREPASS.md)              | Token, QR, penebusan, dan pencabutan.              |
| [Realtime](13-BACKEND-REALTIME.md)              | SSE dan replay notifikasi.                         |
| [Profil pasien](15-PATIENT-PROFILE.md)          | Data diri dan kontak yang dikelola pasien sendiri. |

## Development dan pengujian

- [Backend](../backend/README.md): database, environment, migrasi, seed, dan Docker.
- [Frontend](../frontend/README.md): SSR, proxy API, dashboard, dan tes browser.
- [Kontribusi](../CONTRIBUTING.md): branch, commit, PR, dan pemeriksaan perubahan.
- [Verifikasi backend](14-BACKEND-READINESS.md): bukti pemeriksaan dan batas kesiapan pada waktu pencatatan.
- [Setup backend, 25 Oktober 2026](BACKEND-SETUP-2026-10-25.md).
- [Inventaris aplikasi, 25 Oktober 2026](APP-INVENTORY-2026-10-25.md).
- [Handoff pasien, dokter, dan penyedia](plans/2026-10-15-patient-doctor-and-provider-handoff.md).

## Aset dan riwayat rancangan

- [Aset web](WEB-ASSETS.md) dan [catatan logo](LOGO-EDIT.md).
- [Rencana implementasi](plans/) dan [dokumen Superpowers](superpowers/).

Dokumen rencana dan catatan bertanggal menyimpan konteks pada waktu penulisan.
Untuk perilaku terkini, periksa kontrak modul, implementasi, migrasi, serta tes
yang terkait. Catatan verifikasi lama bukan bukti bahwa pemeriksaan baru sudah
dijalankan.
