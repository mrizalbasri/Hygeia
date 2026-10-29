import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { mkdir } from 'node:fs/promises';
import { chromium } from 'playwright';
import { waitForServer } from './server-ready.js';

const user = {
	id: 'synthetic-profile-patient',
	role: 'patient',
	full_name: 'Pasien Sintetis',
	email: 'profile@example.test',
	patient_ref: 'HYG-PROFILE-DEMO',
	organization: null
};
let profile = {
	full_name: user.full_name,
	email: user.email,
	patient_ref: user.patient_ref,
	birth_date: null,
	birth_year: 1998,
	sex: 'unknown',
	phone: '',
	address: '',
	emergency_contact_name: '',
	emergency_contact_phone: ''
};
const fixture = createServer((_request, response) => {
	response.setHeader('Content-Type', 'application/json');
	response.end(JSON.stringify({ user }));
});
await new Promise((resolve) => fixture.listen(0, '127.0.0.1', resolve));
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
		env: { ...process.env, API_BASE_URL: `http://127.0.0.1:${fixture.address().port}` },
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
	const context = await browser.newContext({ viewport: { width: 1440, height: 1080 } });
	await context.addCookies([{ name: 'hygeia_session', value: 'fixture-profile', url: base }]);
	const page = await context.newPage();
	const errors = [],
		mutations = [];
	page.on('pageerror', (error) => errors.push(error.message));
	let getStatus = 503,
		saveStatus = 200,
		releaseSave,
		saveArrived;
	await page.route('**/api/v1/**', async (route) => {
		const req = route.request();
		const path = new URL(req.url()).pathname;
		const send = (json, status = 200) => route.fulfill({ status, json });
		if (path.endsWith('/auth/csrf')) return send({ csrf_token: 'profile-csrf' });
		if (path.endsWith('/me')) return send({ user });
		if (path.endsWith('/me/profile')) {
			if (req.method() === 'GET') {
				if (getStatus !== 200)
					return send({ error: { message: 'Profil belum dapat dimuat. Coba lagi.' } }, getStatus);
				return send({ profile });
			}
			assert.equal(req.method(), 'PATCH');
			assert.equal(req.headers()['x-csrf-token'], 'profile-csrf');
			mutations.push(req.postDataJSON());
			if (saveStatus !== 200)
				return send({ error: { message: 'Profil belum dapat disimpan. Coba lagi.' } }, saveStatus);
			saveArrived?.();
			await new Promise((resolve) => (releaseSave = resolve));
			profile = { ...profile, ...req.postDataJSON(), birth_year: 2000 };
			user.full_name = profile.full_name;
			return send({ profile });
		}
		throw new Error(`Unexpected API request: ${path}`);
	});

	await page.goto(`${base}/dashboard/patient?view=profile`);
	await page.getByRole('heading', { name: 'Profil saya', level: 1 }).waitFor();
	await page.getByRole('alert').getByText('Profil belum dapat dimuat. Coba lagi.').waitFor();
	assert.equal(await page.locator('form').count(), 0);
	getStatus = 200;
	await page.getByRole('button', { name: 'Coba lagi', exact: true }).click();
	const name = page.getByLabel('Nama lengkap', { exact: true });
	await name.waitFor();
	assert.equal(await name.inputValue(), user.full_name);
	assert.equal(await page.getByLabel('Nomor pasien', { exact: true }).getAttribute('readonly'), '');
	assert.equal(await page.getByLabel('Email', { exact: true }).getAttribute('readonly'), '');
	assert.equal(await page.getByLabel('Tanggal lahir', { exact: false }).inputValue(), '');
	await page.getByText(/Tahun lahir tercatat: 1998/).waitFor();
	const save = page.getByRole('button', { name: 'Simpan profil', exact: true });
	assert.equal(await save.isDisabled(), true);

	await name.fill('Perubahan dibatalkan');
	await page.getByRole('button', { name: 'Batal perubahan', exact: true }).click();
	assert.equal(await name.inputValue(), user.full_name);
	await name.fill('Pasien Profil Sintetis');
	await page.getByLabel('Tanggal lahir', { exact: false }).fill('2000-02-29');
	await page.getByLabel('Jenis kelamin', { exact: true }).selectOption('female');
	await page.getByLabel('Nomor telepon', { exact: false }).fill('tidak valid');
	await save.click();
	await page
		.getByRole('alert')
		.getByText(/Nomor telepon harus/)
		.waitFor();
	assert.equal(mutations.length, 0);
	await page.getByLabel('Nomor telepon', { exact: false }).fill('+62 812-3456-7890');
	await page.getByLabel('Alamat', { exact: false }).fill('Jalan Sintetis 1');
	await page.getByLabel('Nama kontak darurat', { exact: true }).fill('Kontak Sintetis');
	await save.click();
	assert.equal(mutations.length, 0, 'incomplete emergency contact must not submit');
	await page.getByLabel('Telepon kontak darurat', { exact: true }).fill('08123456780');

	saveStatus = 503;
	await save.click();
	await page.getByRole('alert').getByText('Profil belum dapat disimpan. Coba lagi.').waitFor();
	assert.equal(await name.inputValue(), 'Pasien Profil Sintetis');
	assert.equal(await page.getByLabel('Alamat', { exact: false }).inputValue(), 'Jalan Sintetis 1');
	saveStatus = 200;
	const arrival = new Promise((resolve) => (saveArrived = resolve));
	await save.click();
	const pending = page.getByRole('button', { name: 'Menyimpan profil…', exact: true });
	await pending.waitFor();
	await arrival;
	assert.equal(await pending.isDisabled(), true);
	await page
		.locator('form')
		.evaluate((form) =>
			form.dispatchEvent(new SubmitEvent('submit', { bubbles: true, cancelable: true }))
		);
	assert.equal(mutations.length, 2, 'pending save must not submit twice');
	assert.deepEqual(mutations.at(-1), {
		full_name: 'Pasien Profil Sintetis',
		birth_date: '2000-02-29',
		sex: 'female',
		phone: '+62 812-3456-7890',
		address: 'Jalan Sintetis 1',
		emergency_contact_name: 'Kontak Sintetis',
		emergency_contact_phone: '08123456780'
	});
	assert.equal(typeof releaseSave, 'function');
	releaseSave();
	await page.getByRole('status').getByText('Profil Anda berhasil disimpan.').waitFor();
	assert.equal(await page.locator('.sidebar-account strong').textContent(), user.full_name);
	assert.equal(await save.isDisabled(), true);
	await page.reload();
	assert.equal(await name.inputValue(), user.full_name);
	assert.equal(await page.getByLabel('Tanggal lahir', { exact: false }).inputValue(), '2000-02-29');

	await mkdir('.test-results', { recursive: true });
	await page.screenshot({ path: '.test-results/profile-desktop.png', fullPage: true });
	for (const width of [768, 390, 320]) {
		await page.setViewportSize({ width, height: 900 });
		assert.ok(
			await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
			`profile overflow at ${width}`
		);
	}
	await page.setViewportSize({ width: 390, height: 900 });
	await page.screenshot({ path: '.test-results/profile-mobile.png', fullPage: true });
	assert.deepEqual(errors, []);
	await context.close();
	console.log(
		'PASS profile load/retry, read-only identity, edit/reset/save/reload, CSRF, validation, duplicate guard, shell name and responsive layout'
	);
} finally {
	await browser?.close();
	vite.kill();
	await new Promise((resolve) => fixture.close(resolve));
}
