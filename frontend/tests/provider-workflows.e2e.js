import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { chromium } from 'playwright';

const labUser = {
	id: 'fixture-lab-staff',
	full_name: 'Staf Lab Sintetis',
	email: 'lab@example.test',
	role: 'lab',
	organization: { id: 'org-lab-1', name: 'Laboratorium Sentra Demo', type: 'laboratory' }
};

const pharmacyUser = {
	id: 'fixture-pharmacy-staff',
	full_name: 'Apoteker Sintetis',
	email: 'pharmacy@example.test',
	role: 'pharmacy',
	organization: { id: 'org-pharmacy-1', name: 'Apotek Sehat Demo', type: 'pharmacy' }
};

const server = createServer((request, response) => {
	response.setHeader('Content-Type', 'application/json');
	const cookie = request.headers.cookie || '';
	const user = cookie.includes('role=pharmacy') ? pharmacyUser : labUser;
	response.end(JSON.stringify({ user }));
});

await new Promise((resolve) => server.listen(18087, '127.0.0.1', resolve));

const vite = spawn(
	process.execPath,
	['node_modules/vite/bin/vite.js', 'dev', '--host', '127.0.0.1', '--port', '5187', '--strictPort'],
	{
		env: { ...process.env, API_BASE_URL: 'http://127.0.0.1:18087' },
		stdio: ['ignore', 'pipe', 'pipe'],
		windowsHide: true
	}
);

let output = '';
vite.stdout.on('data', (data) => (output += data));
vite.stderr.on('data', (data) => (output += data));
// eslint-disable-next-line no-control-regex
const stripAnsi = (str) => str.replace(/\x1b\[[0-9;]*m/g, '');
const base = 'http://127.0.0.1:5187';
let browser;

try {
	for (let attempt = 0; attempt < 100 && !stripAnsi(output).includes(`${base}/`); attempt++)
		await new Promise((resolve) => setTimeout(resolve, 200));
	assert.ok(stripAnsi(output).includes(`${base}/`), output);

	browser = await chromium.launch({
		...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {}),
		headless: true
	});

	// --- 1. Test Laboratory Workflow ---
	{
		const context = await browser.newContext();
		await context.addCookies([
			{ name: 'hygeia_session', value: 'fixture', url: base },
			{ name: 'role', value: 'lab', url: base }
		]);
		const page = await context.newPage();
		const errors = [];
		page.on('pageerror', (err) => errors.push(err.message));

		const labId = 'lab-req-00000000-1111-2222-3333-444444444444';
		let labItem = {
			id: labId,
			patient_ref: 'HYG-DEMO-LAB',
			test_code: 'GLU-01',
			test_name: 'Tes Glukosa Puasa',
			status: 'sent'
		};
		const labMutations = [];

		await page.route('**/api/v1/**', async (route) => {
			const req = route.request();
			const url = new URL(req.url());
			const path = url.pathname.replace('/api/v1', '');
			const send = (json, status = 200) => route.fulfill({ status, json });

			if (path === '/me') return send({ user: labUser });
			if (path === '/auth/csrf') return send({ csrf_token: 'csrf-fixture' });

			if (req.method() === 'GET') {
				if (path === '/lab-requests') {
					const statusParam = url.searchParams.get('status');
					const list = labItem && (!statusParam || labItem.status === statusParam) ? [labItem] : [];
					return send({ lab_requests: list, limit: 20, offset: 0 });
				}
				if (path === '/notifications/unread-count') return send({ unread_count: 2 });
				if (path.startsWith('/notifications'))
					return send({ notifications: [], next_before_id: '' });
			}

			if (req.method() === 'PATCH') {
				assert.equal(req.headers()['x-csrf-token'], 'csrf-fixture');
				const data = req.postDataJSON();
				labMutations.push({ path, data });
				if (path === `/lab-requests/${labId}/status`) {
					labItem.status = data.status;
					return send({ lab_request: { ...labItem } });
				}
			}

			if (req.method() === 'POST' && path === `/lab-requests/${labId}/result`) {
				assert.equal(req.headers()['x-csrf-token'], 'csrf-fixture');
				const data = req.postDataJSON();
				labMutations.push({ path, data });
				labItem = null; // Item leaves active queue upon completion
				return send(
					{
						result: {
							id: 'receipt-001',
							lab_request_id: labId,
							status: 'completed',
							submitted_at: '2026-10-22T00:48:00Z'
						}
					},
					201
				);
			}

			return send({});
		});

		await page.goto(`${base}/dashboard/lab`);
		await page.getByRole('heading', { name: 'Ruang Laboratorium', exact: true }).waitFor();
		await page.getByText('Tes Glukosa Puasa', { exact: true }).waitFor();
		await page.locator('.health-status').getByText('Terkirim', { exact: true }).waitFor();

		// Transition: sent -> accepted
		await page.getByRole('button', { name: 'Terima permintaan', exact: true }).click();
		await page.locator('.health-status').getByText('Diterima', { exact: true }).waitFor();

		// Transition: accepted -> in_progress
		await page.getByRole('button', { name: 'Mulai pemeriksaan', exact: true }).click();
		await page.locator('.health-status').getByText('Sedang diperiksa', { exact: true }).waitFor();

		// Submit result modal
		await page.getByRole('button', { name: 'Kirim hasil pemeriksaan', exact: true }).click();
		await page.getByRole('heading', { name: 'Kirim Hasil Pemeriksaan', exact: true }).waitFor();
		await page.locator('input[name="value"]').fill('95');
		await page.locator('input[name="unit"]').fill('mg/dL');
		await page.locator('input[name="reference_range"]').fill('70–99');
		await page.locator('textarea[name="note"]').fill('Hasil dalam batas normal');
		await page.getByRole('button', { name: 'Serahkan hasil pemeriksaan', exact: true }).click();

		// Receipt verification
		await page.getByRole('heading', { name: 'Tanda Terima Hasil Pemeriksaan' }).waitFor();
		await page.getByText('receipt-001', { exact: true }).waitFor();
		assert.equal(labMutations.length, 3);
		assert.equal(labMutations[0].data.status, 'accepted');
		assert.equal(labMutations[1].data.status, 'in_progress');
		assert.equal(labMutations[2].data.value, '95');
		assert.equal(labMutations[2].data.unit, 'mg/dL');

		// View notifications tab
		await page.getByRole('link', { name: 'Notifikasi', exact: true }).click();
		await page.getByText('2 notifikasi belum dibaca.', { exact: true }).waitFor();
		assert.equal(errors.length, 0, `Browser errors: ${errors.join(', ')}`);
		await context.close();
	}

	// --- 2. Test Pharmacy Workflow ---
	{
		const context = await browser.newContext();
		await context.addCookies([
			{ name: 'hygeia_session', value: 'fixture', url: base },
			{ name: 'role', value: 'pharmacy', url: base }
		]);
		const page = await context.newPage();
		const errors = [];
		page.on('pageerror', (err) => errors.push(err.message));

		const rxId = 'rx-00000000-2222-3333-4444-555555555555';
		let rxItem = {
			id: rxId,
			patient_ref: 'HYG-DEMO-RX',
			status: 'sent',
			items: [
				{
					medication_name: 'Amoxicillin 500mg',
					dosage: '3x sehari 1 tablet',
					instructions: 'Sesudah makan sampai habis',
					quantity: 15
				}
			]
		};
		const rxMutations = [];

		await page.route('**/api/v1/**', async (route) => {
			const req = route.request();
			const url = new URL(req.url());
			const path = url.pathname.replace('/api/v1', '');
			const send = (json, status = 200) => route.fulfill({ status, json });

			if (path === '/me') return send({ user: pharmacyUser });
			if (path === '/auth/csrf') return send({ csrf_token: 'csrf-fixture' });

			if (req.method() === 'GET') {
				if (path === '/prescriptions') {
					const statusParam = url.searchParams.get('status');
					const list = rxItem && (!statusParam || rxItem.status === statusParam) ? [rxItem] : [];
					return send({ prescriptions: list, limit: 20, offset: 0 });
				}
				if (path === '/notifications/unread-count') return send({ unread_count: 0 });
				if (path.startsWith('/notifications'))
					return send({ notifications: [], next_before_id: '' });
			}

			if (req.method() === 'PATCH' && path === `/prescriptions/${rxId}/status`) {
				assert.equal(req.headers()['x-csrf-token'], 'csrf-fixture');
				const data = req.postDataJSON();
				rxMutations.push({ path, data });
				if (data.status === 'completed') {
					rxItem = null; // Leaves active queue
				} else {
					rxItem.status = data.status;
				}
				return send({ prescription: { ...rxItem } });
			}

			return send({});
		});

		await page.goto(`${base}/dashboard/pharmacy`);
		await page.getByRole('heading', { name: 'Ruang Apotek', exact: true }).waitFor();
		await page.getByRole('heading', { name: 'Amoxicillin 500mg', level: 2 }).waitFor();
		await page.locator('.health-status').getByText('Terkirim', { exact: true }).waitFor();

		// Transition: sent -> accepted
		await page.getByRole('button', { name: 'Terima resep', exact: true }).click();
		await page.locator('.health-status').getByText('Diterima', { exact: true }).waitFor();

		// Transition: accepted -> preparing
		await page.getByRole('button', { name: 'Mulai penyiapan obat', exact: true }).click();
		await page.locator('.health-status').getByText('Sedang disiapkan', { exact: true }).waitFor();

		// Transition: preparing -> ready
		await page.getByRole('button', { name: 'Tandai siap diambil', exact: true }).click();
		await page.locator('.health-status').getByText('Siap diambil', { exact: true }).waitFor();

		// Transition: ready -> completed with confirmation prompt
		await page.getByRole('button', { name: 'Selesaikan penyerahan obat', exact: true }).click();
		await page.getByRole('heading', { name: 'Konfirmasi Penyerahan Obat', exact: true }).waitFor();
		await page.getByRole('button', { name: 'Ya, selesaikan penyerahan', exact: true }).click();

		// Completion notice verification
		await page.getByRole('heading', { name: 'Resep Berhasil Diserahkan' }).waitFor();
		assert.equal(rxMutations.length, 4);
		assert.equal(rxMutations[0].data.status, 'accepted');
		assert.equal(rxMutations[1].data.status, 'preparing');
		assert.equal(rxMutations[2].data.status, 'ready');
		assert.equal(rxMutations[3].data.status, 'completed');

		// View notifications tab
		await page.getByRole('link', { name: 'Notifikasi', exact: true }).click();
		await page.getByText('0 notifikasi belum dibaca.', { exact: true }).waitFor();
		assert.equal(errors.length, 0, `Browser errors: ${errors.join(', ')}`);
		await context.close();
	}

	console.log('PASS Laboratory workspace queue lifecycle, result modal, and receipt banner');
	console.log(
		'PASS Pharmacy workspace queue lifecycle, medication details, and completion confirmation'
	);
	console.log('PASS Role wiring, notifications navigation, and clean browser state');
} finally {
	await browser?.close();
	server.close();
	vite.kill();
}
