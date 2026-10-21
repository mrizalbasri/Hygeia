import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { chromium } from 'playwright';

const user = {
	id: 'fixture-patient',
	full_name: 'Pasien Sintetis',
	email: 'fixture@example.test',
	role: 'patient',
	organization: null
};
const server = createServer((request, response) => {
	response.setHeader('Content-Type', 'application/json');
	response.end(JSON.stringify({ user }));
});
await new Promise((resolve) => server.listen(18086, '127.0.0.1', resolve));
const vite = spawn(
	process.execPath,
	['node_modules/vite/bin/vite.js', 'dev', '--host', '127.0.0.1', '--port', '5186', '--strictPort'],
	{
		env: { ...process.env, API_BASE_URL: 'http://127.0.0.1:18086' },
		stdio: ['ignore', 'pipe', 'pipe'],
		windowsHide: true
	}
);
let output = '';
vite.stdout.on('data', (data) => (output += data));
vite.stderr.on('data', (data) => (output += data));
// eslint-disable-next-line no-control-regex
const stripAnsi = (str) => str.replace(/\x1b\[[0-9;]*m/g, '');
const base = 'http://127.0.0.1:5186';
let browser;
try {
	for (let attempt = 0; attempt < 100 && !stripAnsi(output).includes(`${base}/`); attempt++)
		await new Promise((resolve) => setTimeout(resolve, 200));
	assert.ok(stripAnsi(output).includes(`${base}/`), output);
	browser = await chromium.launch({
		...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {}),
		headless: true
	});
	const context = await browser.newContext();
	await context.addCookies([{ name: 'hygeia_session', value: 'fixture', url: base }]);
	const page = await context.newPage();
	const errors = [],
		mutations = [],
		cursors = [];
	page.on('pageerror', (error) => errors.push(error.message));
	const id = '11111111-1111-4111-8111-111111111111';
	let conflict = false,
		forbidden = false,
		unavailable = false,
		unauthenticated = false;
	let rows = {
		episodes: [{ id, doctor_name: 'Dokter Sintetis', status: 'pending' }],
		lab_requests: [{ id, test_name: 'Tes sintetis', status: 'pending_patient_approval' }],
		prescriptions: [
			{
				id,
				status: 'pending_patient_approval',
				items: [
					{ medication_name: 'Obat sintetis', dosage: 'Demo', instructions: 'Demo', quantity: 1 }
				]
			}
		],
		consents: [{ id, grantee_name: 'Dokter Sintetis', status: 'pending', resource_id: id }]
	};
	await page.route('**/api/v1/**', async (route) => {
		const req = route.request(),
			url = new URL(req.url()),
			path = url.pathname.replace('/api/v1', '');
		const send = (json, status = 200) => route.fulfill({ status, json });
		if (path === '/me') return send({ user });
		if (path === '/auth/csrf') return send({ csrf_token: 'csrf-fixture' });
		if (req.method() !== 'GET') {
			assert.equal(req.headers()['x-csrf-token'], 'csrf-fixture');
			mutations.push({ path, body: req.postDataJSON() });
			if (conflict) {
				conflict = false;
				rows.lab_requests[0].status = 'cancelled';
				return send({ error: { message: 'Status berubah', request_id: 'conflict-fixture' } }, 409);
			}
			const key = path.startsWith('/lab-requests') ? 'lab_requests' : path.split('/')[1];
			if (rows[key])
				rows[key][0].status =
					key === 'consents' ? 'denied' : key === 'episodes' ? 'rejected' : 'cancelled';
			return send({});
		}
		if (path.startsWith('/lab-results/'))
			return forbidden
				? send({ error: { message: 'Akses ditolak' } }, 403)
				: send({
						result: {
							id,
							test_name: 'Hasil sintetis',
							value: 'Nilai rahasia fixture',
							unit: 'demo'
						}
					});
		if (path === '/patient/timeline') {
			const before = url.searchParams.get('before_seq');
			cursors.push(before);
			return route.fulfill({
				contentType: 'application/json',
				body: before
					? '{"events":[],"next_before_seq":null}'
					: '{"events":[{"seq":9223372036854775806,"message":"Kejadian sintetis"}],"next_before_seq":9223372036854775806}'
			});
		}
		if (path === '/notifications/unread-count') return send({ unread_count: 1 });
		if (path === '/notifications')
			return send({
				notifications: [
					{ id: '9223372036854775806', message: 'Notifikasi sintetis', read_at: null }
				],
				next_before_id: null
			});
		if (path === '/notifications/stream')
			return route.fulfill({
				contentType: 'text/event-stream',
				body: 'event: session-ended\ndata: {}\n\n'
			});
		if (unauthenticated) return send({ error: { message: 'Sesi berakhir' } }, 401);
		if (unavailable) return send({ error: { message: 'Layanan terganggu' } }, 503);
		const key = path === '/lab-requests' ? 'lab_requests' : path.slice(1);
		return send({ [key]: rows[key] || [] });
	});
	const view = async (name) => {
		await page.goto(`${base}/dashboard/patient?view=${name}`);
		await page.locator('.wf-heading').waitFor();
	};
	const saved = () =>
		page.getByText('Perubahan berhasil disimpan. Daftar diperbarui.', { exact: true }).waitFor();
	for (const [name, button, body] of [
		['episodes', 'Tolak undangan', { action: 'reject' }],
		['labs', 'Tolak pemeriksaan', { decision: 'reject' }],
		['prescriptions', 'Tolak resep', { decision: 'reject' }],
		['consents', 'Tolak akses', { action: 'deny' }]
	]) {
		await view(name);
		await page.getByRole('button', { name: button, exact: true }).click();
		await saved();
		assert.deepEqual(mutations.at(-1).body, body);
	}
	rows.lab_requests[0].status = 'pending_patient_approval';
	conflict = true;
	await view('labs');
	await page.getByRole('button', { name: 'Setujui pengiriman ke lab', exact: true }).click();
	await page
		.getByRole('alert')
		.getByText(/conflict-fixture/)
		.waitFor();
	await page.locator('.health-status').getByText('Dibatalkan', { exact: true }).waitFor();
	assert.equal(
		await page.getByRole('button', { name: 'Setujui pengiriman ke lab', exact: true }).count(),
		0
	);
	console.log(
		'PASS patient rejections, CSRF headers, conflict refresh and obsolete action removal'
	);
	rows.lab_requests[0] = { id, test_name: 'Tes sintetis', status: 'completed', result_id: id };
	await view('labs');
	await page.getByRole('button', { name: 'Buka hasil pemeriksaan', exact: true }).click();
	await page.getByRole('dialog').getByText('Nilai rahasia fixture demo', { exact: true }).waitFor();
	await page.getByRole('button', { name: 'Tutup', exact: true }).click();
	forbidden = true;
	await page.getByRole('button', { name: 'Buka hasil pemeriksaan', exact: true }).click();
	await page.getByRole('dialog').getByText('Akses ditolak', { exact: true }).waitFor();
	assert.equal(await page.getByText(/Nilai rahasia fixture/).count(), 0);
	await page.getByRole('button', { name: 'Tutup', exact: true }).click();
	unavailable = true;
	await view('episodes');
	await page.getByRole('alert').waitFor();
	assert.equal(await page.getByRole('article').count(), 0);
	unavailable = false;
	await page.getByRole('button', { name: 'Coba lagi', exact: true }).click();
	await page.getByRole('heading', { name: 'Dokter Sintetis', exact: true }).waitFor();
	await view('timeline');
	await page.getByText('Kejadian sintetis', { exact: true }).waitFor();
	await page.getByRole('button', { name: 'Berikutnya', exact: true }).click();
	await page.getByText('Belum ada kejadian pada halaman ini.', { exact: true }).waitFor();
	assert.equal(cursors.at(-1), '9223372036854775806');
	await page.getByRole('button', { name: 'Sebelumnya', exact: true }).click();
	await page.getByText('Kejadian sintetis', { exact: true }).waitFor();
	console.log(
		'PASS forbidden detail clears old payload, unavailable/retry, lossless cursor pagination'
	);
	await view('notifications');
	await page.waitForURL('**/login');
	unauthenticated = true;
	await view('episodes');
	await page.waitForURL('**/login');
	assert.deepEqual(errors, []);
	console.log('PASS SSE session-ended and API 401 redirect, no browser runtime errors');
} finally {
	await browser?.close();
	vite.kill();
	await new Promise((resolve) => server.close(resolve));
}
