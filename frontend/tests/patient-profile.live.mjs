// Opt-in only. Requires a disposable API/database on 127.0.0.1:8080.
// Creates a synthetic patient and leaves it available for inspecting the audit.
import assert from 'node:assert/strict';
import { randomBytes } from 'node:crypto';
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { chromium } from 'playwright';
import { waitForServer } from './server-ready.js';

if (process.env.HYGEIA_LIVE_TEST !== '1') {
	throw new Error('Set HYGEIA_LIVE_TEST=1 only with a disposable API/database on port 8080.');
}
const probe = createServer();
await new Promise((resolve) => probe.listen(0, '127.0.0.1', resolve));
const port = probe.address().port;
await new Promise((resolve) => probe.close(resolve));
const base = `http://127.0.0.1:${port}`;
const vite = spawn(
	process.execPath,
	[
		'node_modules/vite/bin/vite.js',
		'dev',
		'--host',
		'127.0.0.1',
		'--port',
		String(port),
		'--strictPort'
	],
	{
		env: { ...process.env, API_BASE_URL: 'http://127.0.0.1:8080' },
		stdio: ['ignore', 'pipe', 'pipe'],
		windowsHide: true
	}
);
let output = '',
	browser;
vite.stdout.on('data', (data) => (output += data));
vite.stderr.on('data', (data) => (output += data));
try {
	await waitForServer(base, { output: () => output, isAlive: () => vite.exitCode === null });
	browser = await chromium.launch({
		headless: true,
		...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {})
	});
	const context = await browser.newContext();
	const page = await context.newPage();
	const errors = [];
	page.on('pageerror', (error) => errors.push(error.message));
	await page.goto(`${base}/register`);
	await page.getByLabel('Nama lengkap', { exact: true }).fill('Pasien Live Sintetis');
	await page
		.getByLabel('Email', { exact: true })
		.fill(`profile-${randomBytes(6).toString('hex')}@example.test`);
	await page.getByLabel('Tahun lahir', { exact: true }).fill('1998');
	const password = randomBytes(24).toString('base64url');
	await page.getByLabel('Password', { exact: true }).fill(password);
	await page.getByLabel('Konfirmasi password', { exact: true }).fill(password);
	await page.getByRole('button', { name: 'Buat akun pasien', exact: true }).click();
	await page.waitForURL('**/dashboard/patient');
	await page.getByRole('button', { name: /^Menu akun / }).click();
	await page
		.getByRole('navigation', { name: 'Menu akun' })
		.getByRole('link', { name: 'Profil saya', exact: true })
		.click();
	const name = page.getByLabel('Nama lengkap', { exact: true });
	await name.waitFor();
	assert.equal(await name.inputValue(), 'Pasien Live Sintetis');
	await page.getByText(/Tahun lahir tercatat: 1998/).waitFor();
	await name.fill('Pasien Live Diperbarui');
	await page.getByLabel('Tanggal lahir', { exact: false }).fill('1998-06-15');
	await page.getByLabel('Nomor telepon', { exact: false }).fill('081234567890');
	await page.getByLabel('Alamat', { exact: false }).fill('Jalan Live Sintetis 1');
	await page.getByRole('button', { name: 'Simpan profil', exact: true }).click();
	await page.getByRole('status').getByText('Profil Anda berhasil disimpan.').waitFor();
	await page.reload();
	assert.equal(await name.inputValue(), 'Pasien Live Diperbarui');
	assert.equal(await page.getByLabel('Tanggal lahir', { exact: false }).inputValue(), '1998-06-15');
	assert.equal(
		await page.getByLabel('Alamat', { exact: false }).inputValue(),
		'Jalan Live Sintetis 1'
	);
	await page.getByRole('button', { name: /^Menu akun / }).click();
	await page.getByRole('button', { name: 'Keluar', exact: true }).click();
	await page.waitForURL('**/login');
	await page.goto(`${base}/dashboard/patient?view=profile`);
	await page.waitForURL('**/login');
	assert.deepEqual(errors, []);
	await context.close();
	console.log(
		'PASS live profile registration/edit/save/reload/logout via real Go API, PostgreSQL and same-origin Vite proxy'
	);
} finally {
	await browser?.close();
	vite.kill();
}
