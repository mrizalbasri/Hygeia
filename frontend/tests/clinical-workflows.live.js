import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { spawn } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { chromium, request } from 'playwright';

// Explicit opt-in: creates synthetic accounts and clinical records on the local demo API.
if (process.env.HYGEIA_LIVE_TEST !== '1')
	throw new Error('Set HYGEIA_LIVE_TEST=1 for the local demo integration test.');
const env = await readFile('../backend/.env', 'utf8');
const password = env
	.match(/^DEMO_PASSWORD=(.*)$/m)?.[1]
	?.trim()
	.replace(/^['"]|['"]$/g, '');
assert.ok(password, 'Local demo password is required');
const base = 'http://127.0.0.1:5185';
const backend = 'http://127.0.0.1:8080';
const suffix = randomUUID().slice(0, 8);
const temporaryPassword = `Synthetic-${randomUUID()}`;
const clients = [];
let browser;
const vite = spawn(
	process.execPath,
	['node_modules/vite/bin/vite.js', 'dev', '--host', '127.0.0.1', '--port', '5185', '--strictPort'],
	{
		env: { ...process.env, API_BASE_URL: backend },
		stdio: ['ignore', 'pipe', 'pipe'],
		windowsHide: true
	}
);
let output = '';
vite.stdout.on('data', (chunk) => (output += chunk));
vite.stderr.on('data', (chunk) => (output += chunk));
async function call(client, path, method = 'GET', data, expected) {
	const headers = { Origin: backend };
	if (method !== 'GET') {
		const csrf = await client.get(`${backend}/api/v1/auth/csrf`);
		headers['X-CSRF-Token'] = (await csrf.json()).csrf_token;
	}
	const response = await client.fetch(`${backend}/api/v1${path}`, { method, data, headers });
	assert.equal(
		response.status(),
		expected ||
			(method === 'POST' &&
			!path.includes('approval') &&
			!path.includes('redeem') &&
			!path.includes('auth/')
				? 201
				: 200),
		`${method} ${path}: ${response.status()} ${await response.text()}`
	);
	return response.json();
}
async function login(email, secret) {
	const client = await request.newContext();
	clients.push(client);
	await call(client, '/auth/login', 'POST', { email, password: secret }, 200);
	return client;
}
async function ui(client, role) {
	const context = await browser.newContext({
		storageState: await client.storageState(),
		viewport: { width: 1440, height: 1000 }
	});
	const page = await context.newPage();
	page.on('pageerror', (error) => errors.push(error.message));
	page.on('dialog', (dialog) => dialog.accept());
	await page.goto(`${base}/dashboard/${role}`);
	return page;
}
async function view(page, role, name) {
	await page.goto(`${base}/dashboard/${role}?view=${name}`);
	await page.locator('.wf-heading').waitFor();
}
async function clickSaved(page, name) {
	await page.getByRole('button', { name, exact: true }).click();
	try {
		await page
			.getByText('Perubahan berhasil disimpan. Daftar diperbarui.', { exact: true })
			.waitFor({ timeout: 10000 });
	} catch (error) {
		throw new Error(
			`${name}: ${(await page.getByRole('alert').allTextContents()).join(' | ')}; browser: ${errors.join(', ')}`,
			{ cause: error }
		);
	}
}
const errors = [];
try {
	for (let attempt = 0; attempt < 100 && !output.includes('http://127.0.0.1:5185/'); attempt++)
		await new Promise((resolve) => setTimeout(resolve, 200));
	assert.ok(output.includes('http://127.0.0.1:5185/'), output);
	const admin = await login('admin@demo.hygeia.test', password);
	const organizations = {};
	for (const type of ['clinic', 'laboratory', 'pharmacy']) {
		organizations[type] = (
			await call(admin, '/admin/organizations', 'POST', {
				name: `Synthetic UI ${type} ${suffix}`,
				type
			})
		).organization;
	}
	const accounts = {};
	for (const role of ['patient', 'doctor', 'lab', 'pharmacy']) {
		const email = `ui-${role}-${suffix}@example.test`;
		const body = {
			email,
			full_name: `Synthetic UI ${role} ${suffix}`,
			password: temporaryPassword,
			role
		};
		if (role !== 'patient')
			body.organization_id =
				organizations[{ doctor: 'clinic', lab: 'laboratory', pharmacy: 'pharmacy' }[role]].id;
		const { user } = await call(admin, '/admin/users', 'POST', body);
		accounts[role] = { ...user, client: await login(email, temporaryPassword) };
	}
	const users = (await call(admin, '/admin/users?limit=100')).users;
	const patientRef =
		accounts.patient.patient_ref ||
		users.find((user) => user.id === accounts.patient.id)?.patient_ref;
	assert.ok(patientRef, 'Synthetic patient reference');
	browser = await chromium.launch({ channel: 'msedge', headless: true });
	const doctor = await ui(accounts.doctor.client, 'doctor');
	const patient = await ui(accounts.patient.client, 'patient');
	await doctor.getByText('Undang pasien', { exact: true }).click();
	await doctor.getByLabel('Kode pasien', { exact: true }).fill(patientRef);
	await clickSaved(doctor, 'Kirim undangan');
	await view(patient, 'patient', 'episodes');
	await clickSaved(patient, 'Terima undangan');
	const episode = (await call(accounts.doctor.client, '/episodes')).episodes[0];
	assert.equal(episode.status, 'active');
	await view(doctor, 'doctor', 'labs');
	await doctor.getByText('Ajukan pemeriksaan lab', { exact: true }).click();
	await doctor.getByLabel('Perawatan aktif', { exact: true }).selectOption(episode.id);
	await doctor
		.getByLabel('Laboratorium', { exact: true })
		.selectOption(organizations.laboratory.id);
	await doctor.getByLabel('Kode tes', { exact: true }).fill('SYNTHETIC');
	await doctor.getByLabel('Nama pemeriksaan', { exact: true }).fill('Pemeriksaan sintetis UI');
	await clickSaved(doctor, 'Kirim permintaan lab');
	await view(patient, 'patient', 'labs');
	await clickSaved(patient, 'Setujui pengiriman ke lab');
	const labRequest = (await call(accounts.lab.client, '/lab-requests')).lab_requests[0];
	for (const status of ['accepted', 'in_progress'])
		await call(accounts.lab.client, `/lab-requests/${labRequest.id}/status`, 'PATCH', { status });
	const receipt = await call(accounts.lab.client, `/lab-requests/${labRequest.id}/result`, 'POST', {
		value: '10',
		unit: 'demo',
		reference_range: '8–12',
		flag: 'normal',
		note: 'Synthetic UI test only'
	});
	const resultId =
		receipt.result?.id ||
		receipt.id ||
		(await call(accounts.patient.client, '/lab-requests')).lab_requests[0].result_id;
	await view(patient, 'patient', 'labs');
	await patient.getByRole('button', { name: 'Buka hasil pemeriksaan', exact: true }).click();
	await patient.getByRole('dialog').getByText('10 demo', { exact: true }).waitFor();
	await patient.getByRole('button', { name: 'Tutup', exact: true }).click();
	await view(doctor, 'doctor', 'labs');
	await doctor.getByRole('button', { name: 'Buka hasil pemeriksaan', exact: true }).click();
	await doctor.getByRole('dialog').getByText('10 demo', { exact: true }).waitFor();
	await doctor.getByRole('button', { name: 'Tutup', exact: true }).click();
	await view(doctor, 'doctor', 'prescriptions');
	await doctor.getByText('Buat resep', { exact: true }).click();
	await doctor.getByLabel('Perawatan aktif', { exact: true }).selectOption(episode.id);
	await doctor.getByLabel('Nama obat', { exact: true }).fill('Obat sintetis UI');
	await doctor.getByLabel('Dosis', { exact: true }).fill('Dosis sintetis');
	await doctor
		.getByLabel('Aturan pakai', { exact: true })
		.fill('Pengujian antarmuka, bukan instruksi medis');
	await clickSaved(doctor, 'Terbitkan resep');
	await view(patient, 'patient', 'prescriptions');
	await patient.getByRole('button', { name: 'Pilih apotek', exact: true }).click();
	await patient
		.getByLabel('Apotek tujuan', { exact: true })
		.selectOption(organizations.pharmacy.id);
	await clickSaved(patient, 'Setujui & kirim resep');
	const prescription = (await call(accounts.pharmacy.client, '/prescriptions')).prescriptions[0];
	for (const status of ['accepted', 'preparing', 'ready'])
		await call(accounts.pharmacy.client, `/prescriptions/${prescription.id}/status`, 'PATCH', {
			status
		});
	await view(patient, 'patient', 'prescriptions');
	await patient.locator('.health-status').getByText('Siap diambil', { exact: true }).waitFor();
	await patient.getByRole('button', { name: 'Detail obat & aturan pakai', exact: true }).click();
	await patient.getByRole('dialog').getByText('Obat sintetis UI', { exact: true }).waitFor();
	await patient.getByRole('button', { name: 'Tutup', exact: true }).click();
	console.log(
		'PASS real invitation, approval, lab request/result, prescription and pharmacy choice/status'
	);

	// End episode first so subsequent reads must use the explicit patient grant.
	await view(patient, 'patient', 'episodes');
	await clickSaved(patient, 'Akhiri perawatan');
	await view(doctor, 'doctor', 'consents');
	await doctor.getByText('Minta akses hasil lab', { exact: true }).click();
	await doctor.getByLabel('Kode pasien', { exact: true }).fill(patientRef);
	await doctor.getByLabel('ID hasil lab', { exact: true }).fill(resultId);
	await doctor
		.getByLabel('Tujuan akses', { exact: true })
		.fill('Tinjauan sintetis setelah episode selesai');
	await clickSaved(doctor, 'Kirim permintaan akses');
	await view(patient, 'patient', 'consents');
	await clickSaved(patient, 'Izinkan hasil ini');
	await view(doctor, 'doctor', 'consents');
	await doctor.getByRole('button', { name: 'Buka hasil yang diizinkan', exact: true }).click();
	await doctor.getByRole('dialog').getByText('10 demo', { exact: true }).waitFor();
	await doctor.getByRole('button', { name: 'Tutup', exact: true }).click();
	await view(patient, 'patient', 'consents');
	await clickSaved(patient, 'Cabut izin');
	await call(accounts.doctor.client, `/lab-results/${resultId}`, 'GET', undefined, 403);
	await view(patient, 'patient', 'sharing');
	await patient.getByLabel('Hasil pemeriksaan', { exact: true }).selectOption(resultId);
	await patient.getByRole('button', { name: 'Buat CarePass', exact: true }).click();
	const share = patient.getByLabel('Tautan CarePass', { exact: true });
	await share.waitFor();
	const shareURL = await share.inputValue();
	const token = new URLSearchParams(new URL(shareURL).hash.slice(1)).get('token');
	assert.ok(token);
	await doctor.goto(`${base}/carepass#token=${encodeURIComponent(token)}`);
	await doctor.getByRole('button', { name: 'Tebus CarePass', exact: true }).click();
	await doctor.getByText('CarePass berhasil ditebus', { exact: true }).waitFor();
	assert.equal(new URL(doctor.url()).hash, '');
	assert.equal(
		await doctor.evaluate(
			(secret) =>
				[...Object.values(localStorage), ...Object.values(sessionStorage)].some((value) =>
					value.includes(secret)
				),
			token
		),
		false
	);
	await doctor.getByRole('button', { name: /Buka/ }).click();
	await doctor.getByRole('dialog').getByText('10 demo', { exact: true }).waitFor();
	await doctor.getByRole('button', { name: 'Tutup', exact: true }).click();
	await patient.getByRole('button', { name: /Cabut CarePass/ }).click();
	await patient
		.getByText('Perubahan berhasil disimpan. Daftar diperbarui.', { exact: true })
		.waitFor();
	await call(accounts.doctor.client, `/lab-results/${resultId}`, 'GET', undefined, 403);
	console.log(
		'PASS consent grant/revocation, CarePass fragment removal/redemption/revocation, no browser token storage'
	);

	await view(doctor, 'doctor', 'emergency');
	await doctor.getByText('Mulai akses darurat', { exact: true }).click();
	await doctor.getByLabel('Kode pasien', { exact: true }).fill(patientRef);
	await doctor.getByLabel('Alasan', { exact: true }).selectOption('unconscious');
	await doctor
		.getByLabel('Penjelasan alasan', { exact: true })
		.fill('Synthetic emergency workflow test only');
	await clickSaved(doctor, 'Mulai akses darurat 60 menit');
	await doctor.getByRole('button', { name: 'Lihat data dalam cakupan', exact: true }).click();
	await doctor.getByRole('button', { name: 'Buka detail data', exact: true }).first().click();
	await doctor.getByRole('dialog').locator('details').waitFor();
	await doctor.getByRole('button', { name: 'Tutup', exact: true }).click();
	await view(patient, 'patient', 'emergency');
	await patient
		.getByText('Alasan: Synthetic emergency workflow test only', { exact: true })
		.waitFor();
	await view(patient, 'patient', 'audit');
	await patient.getByRole('button', { name: 'Verifikasi integritas audit', exact: true }).click();
	const verification = await call(accounts.patient.client, '/audit/verify');
	await patient
		.getByText(verification.intact ? /Rantai audit utuh/ : /Ketidakcocokan ditemukan/)
		.waitFor();
	console.log('Audit verification response:', JSON.stringify(verification));
	await view(patient, 'patient', 'notifications');
	await patient.getByRole('button', { name: 'Tandai dibaca', exact: true }).first().click();
	await patient
		.getByText('Perubahan berhasil disimpan. Daftar diperbarui.', { exact: true })
		.waitFor();
	await view(patient, 'patient', 'timeline');
	await patient.locator('.wf-row').first().waitFor();
	await mkdir('.test-results', { recursive: true });
	for (const [page, role] of [
		[patient, 'patient'],
		[doctor, 'doctor']
	]) {
		for (const width of [1440, 768, 414, 375, 320]) {
			await page.setViewportSize({ width, height: 1000 });
			assert.ok(
				await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
				`${role} overflow ${width}`
			);
			await page.screenshot({
				path: `.test-results/${role}-workflow-${width}.png`,
				fullPage: true
			});
		}
	}
	assert.deepEqual(errors, []);
	console.log(
		'PASS emergency resource access/accountability, audit verification, notifications, timeline, responsive layouts'
	);
	console.log(`Synthetic local fixture suffix: ${suffix}; retained for audit reproducibility.`);
} finally {
	await browser?.close();
	for (const client of clients) await client.dispose();
	vite.kill();
}
