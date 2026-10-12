# Review akhir backend — 12 Oktober 2026

Baseline: `09d67627b31c42c71d64a10a2aeb69eea8690a6b` (main setelah PR resep).
HEAD kode yang diuji dan diperiksa ulang: `6512a6a0361d178349734318de20b86c2ec1a6d5`.
Diff: `git diff 09d67627b31c42c71d64a10a2aeb69eea8690a6b...HEAD`.
Review mencakup timeline, admin, notifikasi, akses darurat, CarePass/QR dan SSE.
Sumber: BRD/PRD, arsitektur, dua rencana implementasi, dan kontrak API 08–13.
Tidak ada konfigurasi issue tracker khusus; spesifikasi repo digunakan langsung.

## Standards

- P1 diperbaiki: detail hasil lab dengan basis `break_glass` menghapus catatan
  bebas sesuai arsitektur §7.3. Hasil terstruktur tetap tersedia. Tes sentinel
  membuktikan catatan tidak muncul. Perbaikan berada pada branch/PR akses darurat
  dan diteruskan ke seluruh branch turunannya.
- P2 diperbaiki: heartbeat SSE tetap di dalam loop tunggu poll; tidak mengulang
  batch yang sudah dikirim. Tes memeriksa heartbeat tidak membuka fetch baru
  dan kegagalan write menghentikan stream.
- Pemeriksaan ulang reviewer: tidak ada temuan residual. Duplikasi lookup grant
  dinilai belum memerlukan refactor; tidak ada perubahan kosmetik tambahan.

## Spec

- P2 heartbeat di atas diperbaiki dan diperiksa ulang.
- P2 cakupan tes dilengkapi: expiry setelah menunggu row lock PostgreSQL dan
  perpindahan klinik memutus kecocokan grant, masing-masing untuk akses darurat
  dan CarePass. Pengujian concurrent redemption sudah memastikan satu penebus.
- Tidak ditemukan scope creep substantif atau temuan spesifikasi residual.

Standards: 2 temuan ditangani, 0 tersisa. Spec: 2 temuan ditangani, 0 tersisa.
Temuan heartbeat sama dilaporkan pada kedua sumbu, bukan dua bug terpisah.

## Bukti verifikasi

- `go test ./cmd/... ./db/... ./internal/... -count=1` lulus dengan
  `TEST_DATABASE_URL` pada PostgreSQL khusus tes; fixture memakai schema acak.
  Ini mencakup hasil gabungan timeline dengan seluruh modul baru.
- `go vet ./cmd/... ./db/... ./internal/...` lulus.
- `go build -o bin/hygeia-api-complete.exe ./cmd/api` lulus.
- CLI `migrate up` dua kali dan `migrate version`: versi 9, dirty false.
- SSE diuji melalui server HTTP nyata, termasuk replay, live event melewati
  WriteTimeout JSON, batas koneksi, disconnect cleanup, isolasi dan logout.
- Producer notifikasi diuji dengan transaksi yang ditahan sebelum commit;
  advisory lock mencegah cursor melewatkan alokasi ID yang belum committed.
- Regresi audit/notification rollback, scope peran, CSRF dan kepemilikan lulus.

Tes tidak memakai database aplikasi atau mengubah file frontend lokal.
Binary/cache/cluster tes berada di direktori bin yang diabaikan Git.

## Penyerahan

### Perbaikan setelah review lokal lanjutan

Review lanjutan terhadap `f18b2f4` mereproduksi dua kasus yang belum tercakup:
expiry selama menunggu lock kepala audit dan shutdown dengan stream SSE aktif.
Keduanya diperbaiki pada branch pemilik dan diteruskan ke branch integrasi:

- `98c2ea2` (PR #7): pembacaan grant mengunci kepala audit setelah resource/grant,
  lalu memeriksa expiry sebelum response klinis dan audit allow. Expiry selama
  menunggu menghasilkan 403 dan audit deny. Urutan lock tetap konsisten.
- `a02ca78` (PR #8): jalur CarePass menerapkan pengecekan yang sama. Tes lab dan
  resep masing-masing mencakup grant valid serta expired, tanpa payload bocor,
  dan verifikasi integritas rantai audit.
- `c873854` (PR #9): signal context aplikasi membatalkan hanya SSE; context
  request JSON biasa tidak dibatalkan. Tes HTTP membuktikan stream berhenti,
  request biasa tetap drain, dan Shutdown selesai sebelum deadline.

Seluruh verifikasi tetap lokal dengan PostgreSQL khusus tes; tidak menambah CI.
HEAD integrasi kode yang diuji ulang: `3872626`. Regresi seluruh paket, vet,
build API dan tes reproduksi lokal overlay semuanya lulus setelah perbaikan.
Tes reproduksi lokal kini dijadikan regresi source agar kasus tidak berulang.
Tidak ada migrasi tambahan atau perubahan file frontend.

| Urutan | PR | Cakupan | Base |
| --- | --- | --- | --- |
| 1 | [#5](https://github.com/mrizalbasri/Hygeia/pull/5) | Admin | main |
| 2 | [#6](https://github.com/mrizalbasri/Hygeia/pull/6) | Notifikasi | backend-admin |
| 3 | [#7](https://github.com/mrizalbasri/Hygeia/pull/7) | Akses darurat | backend-notifications |
| 4 | [#8](https://github.com/mrizalbasri/Hygeia/pull/8) | CarePass/QR | backend-break-glass |
| 5 | [#9](https://github.com/mrizalbasri/Hygeia/pull/9) | SSE dan regresi keamanan | backend-carepass |
| 6 | [#4](https://github.com/mrizalbasri/Hygeia/pull/4) | Timeline dan hasil integrasi | backend-notification-stream |

Semua branch memakai prefix `codex/`. PR belum di-merge. Gunakan merge commit
dan pertahankan branch base selama PR turunannya masih terbuka; jika memakai
squash/rebase, rebase dan retarget PR turunan terlebih dahulu agar diff tidak
mengulang commit tahap sebelumnya. PR #4 adalah branch gabungan akhir.

SSE membatasi koneksi per proses dan menserialisasi alokasi notifikasi sampai
commit, sesuai volume MVP. Production memerlukan `PUBLIC_APP_URL` HTTPS dan
proxy SSE tanpa buffering. Integrasi UI login/dashboard/scan QR, deployment,
serta uji beban production tetap di luar perubahan backend ini.
