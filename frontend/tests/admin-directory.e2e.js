import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { chromium } from 'playwright';
import { waitForServer } from './server-ready.js';

const user = {
	id: 'admin-fixture',
	role: 'admin',
	full_name: 'Admin Sintetis',
	email: 'admin@example.test',
	organization: null
};
const api = createServer((req, res) => {
	res.setHeader('Content-Type', 'application/json');
	res.end(JSON.stringify({ user }));
});
await new Promise((resolve) => api.listen(0, '127.0.0.1', resolve));
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
		env: {
			...process.env,
			AUTH_PROVIDER: 'local',
			API_BASE_URL: `http://127.0.0.1:${api.address().port}`
		},
		stdio: ['ignore', 'pipe', 'pipe'],
		windowsHide: true
	}
);
let output = '',
	browser;
vite.stdout.on('data', (chunk) => (output += chunk));
vite.stderr.on('data', (chunk) => (output += chunk));
try {
	await waitForServer(base, { output: () => output, isAlive: () => vite.exitCode === null });
	browser = await chromium.launch({
		headless: true,
		...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {})
	});
	const context = await browser.newContext();
	await context.addCookies([{ name: 'hygeia_session', value: 'fixture', url: base }]);
	const page = await context.newPage(),
		errors = [],
		mutations = [],
		offsets = [];
	page.on('pageerror', (error) => errors.push(error.message));
	// Page one deliberately has no clinics. The last clinic is beyond result 100.
	const organizations = Array.from({ length: 105 }, (_, index) => ({
		id: `org-${index}`,
		name: `Organisasi ${index}`,
		type: index < 20 ? 'laboratory' : 'clinic'
	}));
	let fail = false;
	await page.route('**/api/v1/**', async (route) => {
		const request = route.request(),
			url = new URL(request.url()),
			path = url.pathname.replace('/api/v1', '');
		const send = (json, status = 200) => route.fulfill({ status, json });
		if (path === '/me') return send({ user });
		if (path === '/auth/csrf') return send({ csrf_token: 'fixture' });
		if (path === '/admin/organizations') {
			const offset = Number(url.searchParams.get('offset') || 0),
				limit = Number(url.searchParams.get('limit') || 20);
			if (limit === 20) {
				offsets.push(offset);
				if (fail) return send({ error: { message: 'Pilihan organisasi gagal dimuat.' } }, 503);
			}
			return send({ organizations: organizations.slice(offset, offset + limit), offset, limit });
		}
		if (path === '/admin/users' && request.method() === 'POST') {
			mutations.push(request.postDataJSON());
			return send({ user: { id: 'created' } }, 201);
		}
		return send({ users: [], notifications: [], unread_count: 0 });
	});
	await page.goto(`${base}/dashboard/admin?view=users`);
	await page.getByText('Buat akun pengguna baru', { exact: true }).click();
	const picker = page.locator('.wf-picker');
	await picker.getByText(/Tidak ada pilihan yang sesuai/).waitFor();
	for (let offset = 20; offset <= 100; offset += 20) {
		await picker.getByRole('button', { name: 'Pilihan berikutnya', exact: true }).click();
		await picker
			.getByRole('option', { name: `Organisasi ${offset}`, exact: true })
			.waitFor({ state: 'attached' });
	}
	assert.ok(offsets.includes(100), 'must fetch past first 100 organizations');
	await picker.getByRole('combobox').selectOption('org-104');
	assert.equal(await picker.getByRole('button', { name: 'Pilihan berikutnya' }).isDisabled(), true);
	await picker.getByRole('button', { name: 'Pilihan sebelumnya' }).click();
	await picker
		.getByRole('option', { name: 'Organisasi 80', exact: true })
		.waitFor({ state: 'attached' });
	assert.equal(
		await picker.getByRole('combobox').inputValue(),
		'org-104',
		'selection survives page change'
	);
	const form = page.locator('form').filter({ has: page.locator('input[name="password"]') });
	await form.locator('[name="email"]').fill('new@example.test');
	await form.locator('[name="full_name"]').fill('Staf Sintetis');
	await form.locator('[name="password"]').fill('Synthetic-password-2026');
	await form.getByRole('button', { name: 'Daftarkan akun' }).click();
	await page.getByText(/Akun new@example.test/).waitFor();
	assert.equal(mutations[0].organization_id, 'org-104');
	// Changing role must clear a previous clinic; a failed load must allow retry.
	await picker.getByRole('button', { name: 'Pilihan berikutnya' }).click();
	await picker
		.getByRole('option', { name: 'Organisasi 20', exact: true })
		.waitFor({ state: 'attached' });
	await picker.getByRole('combobox').selectOption('org-20');
	fail = true;
	await form.locator('[name="role"]').selectOption('lab');
	await picker
		.getByRole('alert')
		.getByText('Pilihan organisasi gagal dimuat.', { exact: false })
		.waitFor();
	assert.equal(await picker.getByRole('combobox').isDisabled(), true);
	fail = false;
	await picker.getByRole('button', { name: 'Coba lagi' }).click();
	await picker
		.getByRole('option', { name: 'Organisasi 0', exact: true })
		.waitFor({ state: 'attached' });
	assert.equal(await picker.getByRole('combobox').inputValue(), '');
	assert.equal(
		await picker.getByRole('option', { name: 'Organisasi 20', exact: true }).count(),
		0,
		'wrong type hidden'
	);
	await page.setViewportSize({ width: 390, height: 844 });
	assert.ok(
		await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
		'mobile overflow'
	);
	assert.deepEqual(errors, []);
	console.log(
		'PASS Admin organization pagination beyond 100, empty type page, selection retention, role reset and retry'
	);
} finally {
	await browser?.close();
	vite.kill();
	await new Promise((resolve) => api.close(resolve));
}
