import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { chromium } from 'playwright';
import { waitForServer } from './server-ready.js';

const user = {
	id: 'fixture-patient',
	full_name: 'Pasien Sintetis',
	email: 'fixture@example.test',
	role: 'patient',
	organization: null
};
const base = 'http://127.0.0.1:5197';
const id = '11111111-1111-4111-8111-111111111111';
const server = createServer((request, response) => {
	if (request.url.startsWith('/api/v1/notifications/stream')) {
		response.writeHead(200, { 'Content-Type': 'text/event-stream', 'Cache-Control': 'no-cache' });
		response.write(': connected\n\n');
		return;
	}
	response.setHeader('Content-Type', 'application/json');
	response.end(JSON.stringify({ user }));
});
await new Promise((resolve) => server.listen(18107, '127.0.0.1', resolve));
const vite = spawn(
	process.execPath,
	['node_modules/vite/bin/vite.js', 'dev', '--host', '127.0.0.1', '--port', '5197', '--strictPort'],
	{
		env: { ...process.env, AUTH_PROVIDER: 'local', API_BASE_URL: 'http://127.0.0.1:18107' },
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
		...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {}),
		headless: true
	});
	const context = await browser.newContext();
	await context.addCookies([{ name: 'hygeia_session', value: 'fixture', url: base }]);
	const page = await context.newPage();
	const errors = [];
	page.on('pageerror', (error) => errors.push(error.message));
	await page.route('**/api/v1/**', async (route) => {
		const path = new URL(route.request().url()).pathname.replace('/api/v1', '');
		if (path === '/notifications/stream') return route.continue();
		let json = {};
		if (path === '/me') json = { user };
		else if (path === '/auth/csrf') json = { csrf_token: 'csrf-fixture' };
		else if (path === '/lab-requests')
			json = {
				lab_requests: [{ id, result_id: id, test_name: 'Tes sintetis', status: 'completed' }]
			};
		else if (path.startsWith('/lab-results/'))
			json = {
				result: {
					id,
					test_name: 'Hasil sintetis',
					value: 'Nilai koreksi fixture',
					unit: 'demo',
					flag: 'normal'
				}
			};
		else if (path === '/notifications') json = { notifications: [] };
		else if (path === '/notifications/unread-count') json = { unread_count: 0 };
		return route.fulfill({ json });
	});
	const view = async (name) => {
		await page.goto(`${base}/dashboard/patient?view=${name}`);
		await page.locator('.wf-heading').waitFor();
	};
	const saved = () =>
		page.getByText('Perubahan berhasil disimpan. Daftar diperbarui.', { exact: true }).waitFor();
	// Exercise the proposal's CarePass flow in CI, without requiring a live provider.
	// Backend integration tests separately enforce token ownership and authorization.
	const secret = 'synthetic-carepass-token',
		shareURL = `${base}/carepass#token=${secret}`;
	let pass = null,
		redeemDenied = false;
	await page.route('**/api/v1/carepasses**', async (route) => {
		const request = route.request(),
			path = new URL(request.url()).pathname;
		if (request.method() === 'GET')
			return route.fulfill({ json: { carepasses: pass ? [pass] : [] } });
		assert.equal(request.headers()['x-csrf-token'], 'csrf-fixture');
		const body = request.postDataJSON();
		if (path.endsWith('/redeem')) {
			assert.deepEqual(body, { token: secret });
			if (redeemDenied || pass.revoked_at)
				return route.fulfill({ status: 403, json: { error: { message: 'CarePass ditolak' } } });
			return route.fulfill({ json: { carepass: pass } });
		}
		if (request.method() === 'PATCH') {
			assert.equal(path, `/api/v1/carepasses/${id}`);
			assert.deepEqual(body, { action: 'revoke' });
			pass.revoked_at = new Date().toISOString();
			return route.fulfill({ json: { carepass: pass } });
		}
		assert.deepEqual(body, {
			resource_type: 'lab_result',
			resource_id: id,
			ttl_minutes: 15
		});
		pass = {
			id,
			resource_type: 'lab_result',
			resource_id: id,
			expires_at: new Date(Date.now() + 15 * 60000).toISOString()
		};
		return route.fulfill({
			status: 201,
			json: {
				carepass: pass,
				token: secret,
				share_url: shareURL,
				qr_png_data_uri:
					'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jD1sAAAAASUVORK5CYII='
			}
		});
	});
	await view('sharing');
	await page.getByLabel('Hasil pemeriksaan', { exact: true }).selectOption(id);
	await page.getByRole('button', { name: 'Buat CarePass', exact: true }).click();
	await page.getByRole('heading', { name: 'CarePass siap dibagikan' }).waitFor();
	assert.equal(await page.getByLabel('Tautan CarePass', { exact: true }).inputValue(), shareURL);
	assert.equal(await page.getByAltText('QR CarePass untuk satu data yang dipilih').count(), 1);
	await page.getByRole('button', { name: 'Sembunyikan QR & tautan', exact: true }).click();
	assert.equal(await page.getByLabel('Tautan CarePass', { exact: true }).count(), 0);
	await view('sharing');
	assert.equal(await page.getByLabel('Tautan CarePass', { exact: true }).count(), 0);
	// A patient opening the URL must not receive a redemption form.
	await page.goto(shareURL);
	await page.getByText(/CarePass hanya dapat ditebus oleh dokter/).waitFor();
	assert.equal(await page.getByRole('button', { name: 'Tebus CarePass', exact: true }).count(), 0);
	assert.equal(new URL(page.url()).hash, '');
	user.role = 'doctor';
	await page.goto('about:blank');
	await page.goto(shareURL);
	const tokenInput = page.getByLabel('Token atau tautan CarePass', { exact: true });
	await tokenInput.waitFor();
	assert.equal(await tokenInput.inputValue(), secret);
	assert.equal(new URL(page.url()).hash, '');
	// Opening a QR link alone does not redeem it or expose clinical detail.
	assert.equal(await page.getByText('CarePass berhasil ditebus', { exact: true }).count(), 0);
	await page.getByRole('button', { name: 'Tebus CarePass', exact: true }).click();
	await page.getByRole('heading', { name: 'CarePass berhasil ditebus' }).waitFor();
	assert.equal(await tokenInput.inputValue(), '');
	assert.equal(
		await page.evaluate(
			(value) =>
				[...Object.values(localStorage), ...Object.values(sessionStorage)].some((entry) =>
					entry.includes(value)
				),
			secret
		),
		false
	);
	await page.getByRole('button', { name: 'Buka data yang dibagikan', exact: true }).click();
	await page.getByRole('dialog').getByText('Nilai koreksi fixture demo', { exact: true }).waitFor();
	await page.clock.install();
	await page.clock.fastForward(16 * 60000);
	await page
		.getByRole('dialog')
		.getByText('Masa akses berakhir. Tutup panel dan periksa izin terbaru.', { exact: true })
		.waitFor();
	assert.equal(
		await page.getByRole('dialog').getByText('Nilai koreksi fixture demo', { exact: true }).count(),
		0
	);
	await page.getByRole('button', { name: 'Tutup', exact: true }).click();
	// A rejected retry must clear the previous successful redemption.
	redeemDenied = true;
	await tokenInput.fill(shareURL);
	await page.getByRole('button', { name: 'Tebus CarePass', exact: true }).click();
	await page.getByRole('alert').getByText('CarePass ditolak', { exact: true }).waitFor();
	assert.equal(await tokenInput.inputValue(), '');
	assert.equal(
		await page.getByRole('button', { name: 'Buka data yang dibagikan', exact: true }).count(),
		0
	);
	await page.clock.setSystemTime(new Date());
	user.role = 'patient';
	await view('sharing');
	page.once('dialog', (dialog) => dialog.accept());
	await page.getByRole('button', { name: 'Cabut CarePass', exact: true }).click();
	await saved();
	await page.locator('.health-status').getByText('Dicabut', { exact: true }).waitFor();
	assert.equal(await page.getByRole('button', { name: 'Cabut CarePass', exact: true }).count(), 0);
	console.log(
		'PASS CarePass creation, QR/link display, fragment cleanup, explicit redemption, detail, rejection and revoke UI'
	);
	assert.deepEqual(errors, []);
} catch (error) {
	console.error(output.slice(-3000));
	throw error;
} finally {
	await browser?.close();
	vite.kill();
	server.closeAllConnections();
	await new Promise((resolve) => server.close(resolve));
}
