# Hygeia Frontend

Halaman `/` menampilkan landing page Hygeia dengan aset brand, penjelasan peran,
alur perawatan, izin data, dan FAQ. Halaman `/login` memakai backend Go melalui
proxy `/api`, lalu pengguna diarahkan ke `/dashboard/{role}` berdasarkan identitas
dari backend. Dashboard pasien membaca perawatan, permintaan lab, resep, dan
timeline. Dashboard staf masih berupa ruang awal dengan identitas akun.

Form persetujuan, pemilihan apotek, dan detail nilai hasil lab belum tersedia
pada web. Penjelasan alur di landing membedakan kemampuan backend dari antarmuka
demo yang sudah dapat dicoba. Semua data demo bersifat sintetis.

Jalankan backend dan database terlebih dahulu (lihat `../backend/README.md`),
lalu `pnpm install` dan `pnpm dev` dari folder ini. Store pnpm berada di
`.pnpm-store/` lokal yang diabaikan Git, untuk menghindari cache di drive C.

Untuk tes browser kelima peran pada akun demo yang sudah di-seed:

```powershell
# Jalankan backend di :8080 dan pnpm dev di :5173 sebelum tes.
node --env-file=../backend/.env tests/auth.e2e.mjs
```

Tes memakai Microsoft Edge terpasang (`PLAYWRIGHT_CHANNEL=chrome` untuk Chrome)
dan menulis screenshot lokal ke `.test-results/`. Password dibaca dari
`DEMO_PASSWORD`, tidak ditulis di source. `TEST_WEB_URL` dapat dipakai jika alamat
frontend berbeda dari `http://localhost:5173`.

Untuk pengujian UI tanpa backend atau database:

```powershell
node tests/dashboard-theme.e2e.mjs
```

Tes menjalankan API fixture di `127.0.0.1:18081` dan Vite di `127.0.0.1:5177`,
lalu menutup keduanya. Cakupan: pilihan peran dan FAQ dengan keyboard, tautan
landing, aset ponsel, tujuh lebar layar, detail resep, filter timeline, keadaan
kosong/gagal, dan pemulihan. Identitas dan catatan pada fixture adalah sintetis.
Tes ini tidak membuktikan otorisasi backend maupun keberhasilan login akun demo.

Build produksi dan lint file frontend yang diubah telah lolos. `pnpm check`
masih menemukan error pada contoh bawaan `Welcome.svelte.spec.ts`: impor `page`
dari `vitest/browser` tidak tersedia pada versi dependensi yang terpasang.

## Catatan scaffold Svelte

Everything you need to build a Svelte project, powered by [`sv`](https://github.com/sveltejs/cli).

## Creating a project

If you're seeing this, you've probably already done this step. Congrats!

```sh
# create a new project
npx sv create my-app
```

To recreate this project with the same configuration:

```sh
# recreate this project
pnpm dlx sv@1.1.1 create --template minimal --types ts --add prettier eslint tailwindcss="plugins:none" vitest="usages:unit,component" --install pnpm .
```

## Adding features

Add features to your project with `sv add`:

```sh
npx sv add
```

For example, to add Tailwind CSS:

```sh
npx sv add tailwindcss
```

## Developing

Once you've created a project and installed dependencies with `npm install` (or `pnpm install` or `yarn`), start a development server:

```sh
npm run dev

# or start the server and open the app in a new browser tab
npm run dev -- --open
```

## Building

To create a production version of your app:

```sh
npm run build
```

You can preview the production build with `npm run preview`.

> To deploy your app, you may need to install an [adapter](https://svelte.dev/docs/kit/adapters) for your target environment.
