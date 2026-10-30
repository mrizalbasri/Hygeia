import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { chromium } from 'playwright';
import { waitForServer } from './server-ready.js';

const user = {
	id: 'synthetic-patient',
	role: 'patient',
	full_name: 'Pasien Sintetis',
	email: 'new@example.test',
	patient_ref: 'HYG-DEMO-NEW',
	organization: null
};
const server = createServer((request, response) => {
	response.setHeader('Content-Type', 'application/json');
	response.end(JSON.stringify({ user }));
});
await new Promise((resolve) => server.listen(18090, '127.0.0.1', resolve));
const vite = spawn(
	process.execPath,
	['node_modules/vite/bin/vite.js', 'dev', '--host', '127.0.0.1', '--port', '5190', '--strictPort'],
	{
		env: { ...process.env, AUTH_PROVIDER: 'local', API_BASE_URL: 'http://127.0.0.1:18090' },
		stdio: ['ignore', 'pipe', 'pipe'],
		windowsHide: true
	}
);
let output = '',
	browser;
vite.stdout.on('data', (data) => (output += data));
vite.stderr.on('data', (data) => (output += data));
const base = 'http://127.0.0.1:5190';
await mkdir('.test-results', { recursive: true });

try {
	await waitForServer(base, { output: () => output, isAlive: () => vite.exitCode === null });
	browser = await chromium.launch({
		...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {}),
		headless: true
	});
	const context = await browser.newContext({ viewport: { width: 1440, height: 1080 } });
	const page = await context.newPage();
	const errors = [],
		requests = [];
	page.on('pageerror', (e) => errors.push(e.message));
	let status = 409,
		release;
	await page.route('**/api/v1/**', async (route) => {
		const req = route.request(),
			path = new URL(req.url()).pathname;
		const send = (json, code = 200) => route.fulfill({ status: code, json });
		if (path.endsWith('/auth/csrf')) return send({ csrf_token: 'csrf-registration' });
		if (path.endsWith('/auth/register')) {
			assert.equal(req.headers()['x-csrf-token'], 'csrf-registration');
			requests.push(req.postDataJSON());
			if (status === 0) return route.abort();
			if (status !== 201)
				return send({ error: { message: `Pendaftaran gagal (${status}). Coba lagi.` } }, status);
			await new Promise((resolve) => {
				release = resolve;
			});
			return route.fulfill({
				status: 201,
				json: { user },
				headers: { 'Set-Cookie': 'hygeia_session=fixture; Path=/; HttpOnly; SameSite=Lax' }
			});
		}
		if (path.endsWith('/me')) return send({ user });
		return send({ episodes: [], lab_requests: [], prescriptions: [], events: [], unread_count: 0 });
	});
	await page.goto(`${base}/login`);
	await page.getByRole('link', { name: 'Daftar sebagai pasien', exact: true }).click();
	await page.getByRole('heading', { name: 'Daftar sebagai pasien', exact: true }).waitFor();
	await page.getByLabel('Nama lengkap', { exact: true }).fill(' Pasien Sintetis ');
	await page.getByLabel('Email', { exact: true }).fill('new@example.test');
	await page.getByLabel('Tahun lahir', { exact: true }).fill('2000');
	await page.getByLabel('Password', { exact: true }).fill('Synthetic-password-2026');
	await page.getByLabel('Konfirmasi password', { exact: true }).fill('Different-password-2026');
	const submit = page.getByRole('button', { name: 'Buat akun pasien', exact: true });
	await submit.click();
	await page
		.getByRole('alert')
		.getByText(/Konfirmasi password belum sama/)
		.waitFor();
	assert.equal(requests.length, 0);
	await page.getByLabel('Konfirmasi password', { exact: true }).fill('Synthetic-password-2026');
	await page.getByLabel('Tahun lahir', { exact: true }).fill(String(new Date().getFullYear() + 1));
	await submit.click();
	assert.equal(requests.length, 0, 'future birth year must not submit');
	await page.getByLabel('Tahun lahir', { exact: true }).fill('2000');
	for (const code of [409, 429, 503]) {
		status = code;
		await submit.click();
		await page
			.getByRole('alert')
			.getByText(`Pendaftaran gagal (${code}). Coba lagi.`, { exact: true })
			.waitFor();
		assert.equal(
			await page.getByLabel('Nama lengkap', { exact: true }).inputValue(),
			' Pasien Sintetis '
		);
	}
	status = 0;
	await submit.click();
	await page
		.getByRole('alert')
		.getByText(/Layanan belum dapat dihubungi/)
		.waitFor();
	await page.screenshot({ path: '.test-results/register-desktop.png', fullPage: true });
	for (const width of [768, 390, 320]) {
		await page.setViewportSize({ width, height: 900 });
		assert.ok(
			await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
			`registration overflow at ${width}`
		);
	}
	await page.screenshot({ path: '.test-results/register-mobile.png', fullPage: true });
	status = 201;
	await submit.click();
	await page.getByRole('button', { name: 'Mendaftarkan akun…', exact: true }).waitFor();
	assert.equal(
		await page.getByRole('button', { name: 'Mendaftarkan akun…', exact: true }).isDisabled(),
		true
	);
	await page
		.locator('form')
		.evaluate((form) =>
			form.dispatchEvent(new SubmitEvent('submit', { bubbles: true, cancelable: true }))
		);
	await page.waitForFunction(
		() => document.querySelector('form')?.getAttribute('aria-busy') === 'true'
	);
	assert.equal(requests.length, 5, 'pending registration must not be submitted twice');
	assert.deepEqual(requests.at(-1), {
		email: 'new@example.test',
		password: 'Synthetic-password-2026',
		full_name: 'Pasien Sintetis',
		birth_year: 2000,
		sex: 'unknown'
	});
	assert.equal(typeof release, 'function');
	release();
	await page.waitForURL('**/dashboard/patient');
	await page.reload();
	assert.equal(new URL(page.url()).pathname, '/dashboard/patient');
	assert.deepEqual(errors, []);
	await context.close();
	console.log(
		'PASS registration validation, CSRF, errors/retry, duplicate guard, session redirect and responsive layout'
	);
} finally {
	await browser?.close();
	vite.kill();
	await new Promise((resolve) => server.close(resolve));
}
