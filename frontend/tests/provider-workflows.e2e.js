import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { mkdir } from 'node:fs/promises';
import { chromium } from 'playwright';
import { waitForServer } from './server-ready.js';

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

const adminUser = {
	id: 'fixture-admin-staff',
	full_name: 'Administrator Sintetis',
	email: 'admin@example.test',
	role: 'admin',
	organization: null
};

const server = createServer((request, response) => {
	response.setHeader('Content-Type', 'application/json');
	const cookie = request.headers.cookie || '';
	const user = cookie.includes('role=admin')
		? adminUser
		: cookie.includes('role=pharmacy')
			? pharmacyUser
			: labUser;
	response.end(JSON.stringify({ user }));
});

await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
const portProbe = createServer();
await new Promise((resolve) => portProbe.listen(0, '127.0.0.1', resolve));
const port = portProbe.address().port;
await new Promise((resolve) => portProbe.close(resolve));

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
		env: { ...process.env, API_BASE_URL: `http://127.0.0.1:${server.address().port}` },
		stdio: ['ignore', 'pipe', 'pipe'],
		windowsHide: true
	}
);

let output = '';
vite.stdout.on('data', (data) => (output += data));
vite.stderr.on('data', (data) => (output += data));
// eslint-disable-next-line no-control-regex
const stripAnsi = (str) => str.replace(/\x1b\[[0-9;]*m/g, '');
const base = `http://127.0.0.1:${port}`;
let browser;
await mkdir('.test-results', { recursive: true });

try {
	await waitForServer(base, {
		output: () => stripAnsi(output),
		isAlive: () => vite.exitCode === null
	});

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
		let denyLabMutation = false;
		const resultId = '11111111-2222-4333-8444-555555555555';
		let correctionFailure = 403;

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
					if (denyLabMutation) {
						labItem = null;
						return send({ error: { message: 'Penugasan sudah berakhir.' } }, 403);
					}
					labItem.status = data.status;
					return send({ lab_request: { ...labItem } });
				}
			}

			if (req.method() === 'POST' && path === `/lab-requests/${labId}/result`) {
				if (denyLabMutation) {
					labItem = null;
					return send({ error: { message: 'Penugasan sudah berakhir.' } }, 403);
				}
				assert.equal(req.headers()['x-csrf-token'], 'csrf-fixture');
				const data = req.postDataJSON();
				labMutations.push({ path, data });
				labItem = null; // Item leaves active queue upon completion
				return send(
					{
						result: {
							id: resultId,
							lab_request_id: labId,
							status: 'completed',
							submitted_at: '2026-10-22T00:48:00Z'
						}
					},
					201
				);
			}
			if (req.method() === 'POST' && path === `/lab-results/${resultId}/corrections`) {
				assert.equal(req.headers()['x-csrf-token'], 'csrf-fixture');
				const data = req.postDataJSON();
				labMutations.push({ path, data });
				if (correctionFailure)
					return send(
						{ error: { message: 'Koreksi belum diizinkan.', request_id: 'correction-fixture' } },
						correctionFailure
					);
				return send({ correction: { id: 'correction-001', lab_result_id: resultId } }, 201);
			}

			return send({});
		});

		await page.goto(`${base}/dashboard/lab?view=queue`);
		await page.getByRole('heading', { name: 'Ruang Laboratorium', exact: true }).waitFor();
		try {
			await page.getByText('Tes Glukosa Puasa', { exact: true }).waitFor();
		} catch (error) {
			throw new Error(
				`Laboratory queue: ${await page.locator('body').innerText()}; browser errors: ${errors.join('; ')}; server: ${output.slice(-3000)}`,
				{ cause: error }
			);
		}
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
		assert.equal(await page.locator('select[name="flag"]').inputValue(), '');
		assert.equal(
			await page.getByRole('button', { name: 'Tinjau hasil pemeriksaan', exact: true }).isEnabled(),
			false
		);
		await page.locator('select[name="flag"]').selectOption('normal');
		await page.getByRole('button', { name: 'Tinjau hasil pemeriksaan', exact: true }).click();
		await page.getByRole('heading', { name: 'Tinjau sebelum mengirim' }).waitFor();
		assert.equal(labMutations.length, 2, 'review must not submit a result');
		await page.getByRole('button', { name: 'Kembali mengedit' }).click();
		assert.equal(await page.locator('input[name="value"]').inputValue(), '95');
		await page.getByRole('button', { name: 'Tinjau hasil pemeriksaan', exact: true }).click();
		await page.getByRole('button', { name: 'Serahkan hasil pemeriksaan', exact: true }).click();

		// Receipt verification
		await page.getByRole('heading', { name: 'Tanda Terima Hasil Pemeriksaan' }).waitFor();
		await page.getByText(resultId, { exact: true }).waitFor();
		assert.equal(labMutations.length, 3);
		assert.equal(labMutations[0].data.status, 'accepted');
		assert.equal(labMutations[1].data.status, 'in_progress');
		assert.equal(labMutations[2].data.value, '95');
		assert.equal(labMutations[2].data.unit, 'mg/dL');

		await page.getByRole('link', { name: 'Koreksi hasil ini', exact: true }).click();
		await page.getByRole('heading', { name: 'Tambahkan koreksi hasil', exact: true }).waitFor();
		const correctionForm = page
			.locator('form')
			.filter({ has: page.locator('textarea[name="reason"]') });
		assert.equal(await correctionForm.locator('input[name="result_id"]').inputValue(), resultId);
		await correctionForm.locator('textarea[name="reason"]').fill('Koreksi alat sintetis');
		await correctionForm.locator('input[name="value"]').fill('101');
		await correctionForm.locator('input[name="unit"]').fill('mg/dL');
		await correctionForm.locator('input[name="reference_range"]').fill('70–99');
		assert.equal(await correctionForm.locator('select[name="flag"]').inputValue(), '');
		await correctionForm.locator('select[name="flag"]').selectOption('high');
		await correctionForm.locator('textarea[name="note"]').fill('Catatan koreksi sintetis');
		for (const status of [403, 409, 503]) {
			correctionFailure = status;
			const beforeReview = labMutations.length;
			await correctionForm
				.getByRole('button', { name: 'Tinjau koreksi hasil', exact: true })
				.click();
			await page.getByRole('heading', { name: 'Tinjau sebelum mengirim' }).waitFor();
			assert.equal(labMutations.length, beforeReview, 'review must not submit a correction');
			await page.getByRole('button', { name: 'Kirim koreksi hasil', exact: true }).click();
			await page
				.getByRole('alert')
				.getByText(/correction-fixture/)
				.waitFor();
			await page.getByRole('button', { name: 'Kembali mengedit' }).click();
			assert.equal(await correctionForm.locator('input[name="value"]').inputValue(), '101');
			await correctionForm
				.locator('textarea[name="reason"]')
				.fill(`Koreksi alat sintetis ${status}`);
		}
		await page.screenshot({ path: '.test-results/lab-correction-desktop.png', fullPage: true });
		await page.setViewportSize({ width: 390, height: 844 });
		assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
		await page.screenshot({ path: '.test-results/lab-correction-mobile.png', fullPage: true });
		correctionFailure = 0;
		await correctionForm.getByRole('button', { name: 'Tinjau koreksi hasil', exact: true }).click();
		await page.getByRole('button', { name: 'Kirim koreksi hasil', exact: true }).click();
		await page.getByText('Koreksi hasil berhasil dikirim.', { exact: true }).waitFor();
		assert.equal(await correctionForm.locator('input[name="value"]').inputValue(), '');
		assert.deepEqual(labMutations.at(-1).data, {
			reason: 'Koreksi alat sintetis 503',
			value: '101',
			unit: 'mg/dL',
			reference_range: '70–99',
			flag: 'high',
			note: 'Catatan koreksi sintetis'
		});
		await page.setViewportSize({ width: 1440, height: 1000 });
		for (const status of ['sent', 'in_progress']) {
			labItem = {
				id: labId,
				patient_ref: 'HYG-STALE-LAB',
				test_code: 'STALE',
				test_name: 'Pemeriksaan lama',
				status
			};
			denyLabMutation = true;
			await page.goto(`${base}/dashboard/lab?view=queue`);
			await page.getByRole('heading', { name: 'Pemeriksaan lama', exact: true }).waitFor();
			if (status === 'sent') {
				await page.getByRole('button', { name: 'Terima permintaan', exact: true }).click();
			} else {
				await page.getByRole('button', { name: 'Kirim hasil pemeriksaan', exact: true }).click();
				await page.locator('input[name="value"]').fill('123');
				await page.locator('select[name="flag"]').selectOption('unknown');
				await page.getByRole('button', { name: 'Tinjau hasil pemeriksaan', exact: true }).click();
				await page.getByRole('button', { name: 'Serahkan hasil pemeriksaan', exact: true }).click();
			}
			await page
				.getByText('Belum ada permintaan pemeriksaan pada antrean laboratorium Anda.', {
					exact: true
				})
				.waitFor({ timeout: 5000 });
			assert.equal(await page.locator('dialog[open]').count(), 0, '403 closes the result dialog');
			assert.equal(await page.getByText('HYG-STALE-LAB', { exact: true }).count(), 0);
		}

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
		let denyRxMutation = false;

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
				if (denyRxMutation) {
					rxItem = null;
					return send({ error: { message: 'Resep sudah diselesaikan petugas lain.' } }, 403);
				}
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
		rxItem = {
			id: rxId,
			patient_ref: 'HYG-STALE-RX',
			status: 'ready',
			items: [
				{
					medication_name: 'Obat kedaluwarsa akses',
					dosage: 'Dosis sintetis',
					instructions: 'Aturan sintetis',
					quantity: 1
				}
			]
		};
		denyRxMutation = true;
		await page.goto(`${base}/dashboard/pharmacy`);
		await page.getByRole('heading', { name: 'Obat kedaluwarsa akses', level: 2 }).waitFor();
		await page.getByRole('button', { name: 'Selesaikan penyerahan obat', exact: true }).click();
		await page.getByRole('button', { name: 'Ya, selesaikan penyerahan', exact: true }).click();
		await page
			.getByText('Belum ada resep aktif pada antrean apotek Anda.', { exact: true })
			.waitFor({ timeout: 5000 });
		assert.equal(await page.getByText('Dosis sintetis', { exact: false }).count(), 0);

		// View notifications tab
		await page.getByRole('link', { name: 'Notifikasi', exact: true }).click();
		await page.getByText('0 notifikasi belum dibaca.', { exact: true }).waitFor();
		assert.equal(errors.length, 0, `Browser errors: ${errors.join(', ')}`);
		await context.close();
	}

	// --- 3. Test Admin Workflow ---
	{
		const context = await browser.newContext();
		await context.addCookies([
			{ name: 'hygeia_session', value: 'fixture', url: base },
			{ name: 'role', value: 'admin', url: base }
		]);
		const page = await context.newPage();
		const errors = [];
		page.on('pageerror', (err) => errors.push(err.message));

		const orgs = [{ id: 'org-1', name: 'Klinik Pratama', type: 'clinic' }];
		const users = [
			{
				id: 'u-1',
				email: 'maya@example.test',
				full_name: 'dr. Maya',
				role: 'doctor',
				organization_id: 'org-1',
				disabled_at: null
			}
		];
		const chains = [{ chain_id: 'chain-1', patient_ref: 'HYG-DEMO-001', events: 10 }];
		const breakglassSessions = [
			{
				id: 'bg-1',
				patient_ref: 'HYG-DEMO-001',
				doctor_user_id: 'u-1',
				organization_id: 'org-1',
				reason_code: 'life_threatening',
				reason_text: 'Pasien tidak sadar dan membutuhkan tindakan segera.',
				starts_at: '2026-10-22T00:00:00Z',
				expires_at: '2026-10-22T01:00:00Z',
				review_decision: null,
				review_note: null
			}
		];
		const adminMutations = [];

		await page.route('**/api/v1/**', async (route) => {
			const req = route.request();
			const url = new URL(req.url());
			const path = url.pathname.replace('/api/v1', '');
			const send = (json, status = 200) => route.fulfill({ status, json });

			if (path === '/me') return send({ user: adminUser });
			if (path === '/auth/csrf') return send({ csrf_token: 'csrf-fixture' });

			if (req.method() === 'GET') {
				if (path === '/admin/organizations')
					return send({ organizations: orgs, limit: 20, offset: 0 });
				if (path === '/admin/users') return send({ users, limit: 20, offset: 0 });
				if (path === '/admin/audit-chains') return send({ chains, limit: 20, offset: 0 });
				if (path.startsWith('/audit/verify')) {
					return send({ chain: 'chain-1', events: 10, intact: true });
				}
				if (path === '/admin/break-glass')
					return send({ sessions: breakglassSessions, limit: 20, offset: 0 });
				if (path === '/admin/activity') return send({ activity: [], limit: 20, offset: 0 });
				if (path === '/notifications/unread-count') return send({ unread_count: 0 });
				if (path.startsWith('/notifications'))
					return send({ notifications: [], next_before_id: '' });
			}

			if (req.method() === 'POST' && path === '/admin/organizations') {
				assert.equal(req.headers()['x-csrf-token'], 'csrf-fixture');
				const data = req.postDataJSON();
				adminMutations.push({ path, data });
				const newOrg = { id: 'org-new', name: data.name, type: data.type };
				orgs.push(newOrg);
				return send({ organization: newOrg }, 201);
			}
			if (req.method() === 'POST' && path === '/admin/users') {
				assert.equal(req.headers()['x-csrf-token'], 'csrf-fixture');
				adminMutations.push({ path, data: req.postDataJSON() });
				return send({ user: { id: 'created-user' } }, 201);
			}

			if (req.method() === 'PATCH' && path === '/admin/users/u-1') {
				assert.equal(req.headers()['x-csrf-token'], 'csrf-fixture');
				const data = req.postDataJSON();
				adminMutations.push({ path, data });
				Object.assign(users[0], data);
				if (data.disabled) users[0].disabled_at = '2026-10-22T00:00:00Z';
				return send({ user: users[0] });
			}

			if (req.method() === 'PATCH' && path === '/admin/break-glass/bg-1') {
				assert.equal(req.headers()['x-csrf-token'], 'csrf-fixture');
				const data = req.postDataJSON();
				adminMutations.push({ path, data });
				breakglassSessions[0].review_decision = data.decision;
				breakglassSessions[0].review_note = data.note;
				return send({ session: breakglassSessions[0] });
			}

			return send({});
		});

		await page.goto(`${base}/dashboard/admin`);
		await page.getByRole('heading', { name: 'Kelola Platform Hygeia', exact: true }).waitFor();
		await page.getByText('Klinik Pratama', { exact: true }).waitFor();

		// Create new organization
		await page.getByText('Tambah organisasi baru', { exact: true }).click();
		await page.locator('input[name="name"]').fill('Lab Sentra Diagnostika');
		await page.locator('select[name="type"]').selectOption('laboratory');
		await page.getByRole('button', { name: 'Daftarkan organisasi', exact: true }).click();
		await page.getByText('Lab Sentra Diagnostika', { exact: true }).waitFor();

		// User list resolves organization IDs, and rename starts from full_name.
		await page.getByRole('link', { name: 'Akun', exact: true }).click();
		await page.getByRole('heading', { name: 'dr. Maya', exact: true }).waitFor();
		await page.getByText('Buat akun pengguna baru', { exact: true }).click();
		const createForm = page.locator('form').filter({ has: page.locator('input[name="password"]') });
		await createForm.locator('input[name="email"]').fill('new@example.test');
		await createForm.locator('input[name="full_name"]').fill('Pengguna Sintetis');
		await createForm.locator('select[name="role"]').selectOption('patient');
		const exactPassword = '  fixture-password-123  ';
		await createForm.locator('input[name="password"]').fill(exactPassword);
		await createForm.getByRole('button', { name: 'Daftarkan akun', exact: true }).click();
		await page
			.getByText('Akun new@example.test (Pasien) berhasil dibuat.', { exact: true })
			.waitFor();
		assert.equal(
			adminMutations.find((item) => item.path === '/admin/users').data.password,
			exactPassword
		);
		await page
			.locator('article.wf-row')
			.filter({ hasText: 'dr. Maya' })
			.getByText('Klinik Pratama', { exact: true })
			.waitFor();
		await page.getByRole('button', { name: 'Ubah nama', exact: true }).click();
		const renameDialog = page.locator('dialog[open]');
		const renameInput = renameDialog.locator('input[name="full_name"]');
		await renameInput.waitFor();
		assert.equal(await renameInput.inputValue(), 'dr. Maya');
		await renameInput.fill('dr. Maya Putri');
		await renameDialog.getByRole('button', { name: 'Simpan nama baru', exact: true }).click();
		await page.getByRole('heading', { name: 'dr. Maya Putri', exact: true }).waitFor();

		// Disable the account and confirm the status refresh.
		await page.getByRole('button', { name: 'Nonaktifkan akun', exact: true }).click();
		await page.getByText('Dinonaktifkan', { exact: true }).waitFor();

		// Break-glass review uses the fields returned by the admin API.
		await page.getByRole('link', { name: 'Tinjauan akses darurat', exact: true }).click();
		await page.getByText('Dokter ID:', { exact: false }).waitFor();
		await page.locator('article.wf-row').getByText('Klinik Pratama', { exact: true }).waitFor();
		await page.getByRole('button', { name: 'Tinjau akses darurat ini', exact: true }).click();
		const reviewDialog = page.locator('dialog[open]');
		await reviewDialog
			.locator('textarea[name="note"]')
			.fill('Alasan darurat sesuai prosedur dan telah diverifikasi.');
		await reviewDialog
			.getByRole('button', { name: 'Simpan Keputusan Peninjauan', exact: true })
			.click();
		await page.getByText('Sesuai prosedur', { exact: true }).waitFor();

		// View audit chains and verify
		await page.getByRole('link', { name: 'Status audit', exact: true }).click();
		await page.getByText('Pasien: HYG-DEMO-001', { exact: true }).waitFor();
		await page.getByRole('button', { name: 'Verifikasi Integritas HMAC', exact: true }).click();
		await page.getByText('UTUH & TERVERIFIKASI (HMAC VALID)', { exact: false }).waitFor();

		const renameMutation = adminMutations.find(
			(mutation) => mutation.path === '/admin/users/u-1' && mutation.data.full_name
		);
		assert.equal(renameMutation.data.full_name, 'dr. Maya Putri');
		const disableMutation = adminMutations.find(
			(mutation) => mutation.path === '/admin/users/u-1' && mutation.data.disabled === true
		);
		assert.ok(disableMutation, 'disable account mutation was sent');
		const reviewMutation = adminMutations.find(
			(mutation) => mutation.path === '/admin/break-glass/bg-1'
		);
		assert.equal(reviewMutation.data.decision, 'appropriate');
		assert.match(reviewMutation.data.note, /sesuai prosedur/);

		assert.equal(errors.length, 0, `Browser errors: ${errors.join(', ')}`);
		await context.close();
	}

	console.log(
		'PASS Laboratory queue, result receipt, correction form, CSRF, errors/retry and mobile layout'
	);
	console.log(
		'PASS Pharmacy workspace queue lifecycle, medication details, and completion confirmation'
	);
	console.log(
		'PASS Admin workspace organizations, account updates, break-glass review, audit verification, and role wiring'
	);
	console.log('PASS Role wiring, notifications navigation, and clean browser state');
} finally {
	await browser?.close();
	server.close();
	vite.kill();
}
