import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { mkdir } from 'node:fs/promises';
import { chromium } from 'playwright';
import { waitForServer } from './server-ready.js';

const id = '11111111-1111-4111-8111-111111111111';
const resultId = '22222222-2222-4222-8222-222222222222';
const users = Object.fromEntries(
	['patient', 'doctor', 'lab', 'pharmacy'].map((role) => [
		role,
		{
			id: role,
			role,
			full_name: `Synthetic ${role}`,
			email: `${role}@example.test`,
			organization:
				role === 'patient'
					? null
					: {
							id,
							name: 'Synthetic provider',
							type: { doctor: 'clinic', lab: 'laboratory', pharmacy: 'pharmacy' }[role]
						}
		}
	])
);
const server = createServer((req, res) => {
	const role =
		/hygeia_session=fixture-(doctor|lab|pharmacy)/.exec(req.headers.cookie || '')?.[1] || 'patient';
	res.setHeader('Content-Type', 'application/json');
	res.end(JSON.stringify({ user: users[role] }));
});
await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
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
			API_BASE_URL: `http://127.0.0.1:${server.address().port}`
		},
		stdio: ['ignore', 'pipe', 'pipe'],
		windowsHide: true
	}
);
let output = '',
	browser;
vite.stdout.on('data', (chunk) => (output += chunk));
vite.stderr.on('data', (chunk) => (output += chunk));
const errors = [],
	mutations = [];
let labStatus = 'sent',
	prescriptionStatus = 'sent',
	corrected = false,
	resultDenied = false,
	bootstrapFailure = false;
try {
	await waitForServer(base, { output: () => output, isAlive: () => vite.exitCode === null });
	browser = await chromium.launch({ headless: true });
	await mkdir('.test-results/workflow-recovery', { recursive: true });
	async function pageFor(role) {
		const context = await browser.newContext();
		await context.addCookies([
			{ name: 'hygeia_session', value: `fixture-${role}`, url: base },
			{ name: 'role', value: role, url: base }
		]);
		await context.addInitScript(() => {
			window.__streams = [];
			window.EventSource = class extends EventTarget {
				constructor(url) {
					super();
					this.url = url;
					this.closed = false;
					window.__streams.push(this);
				}
				close() {
					this.closed = true;
				}
			};
		});
		const page = await context.newPage();
		page.on('pageerror', (error) => errors.push(error.message));
		page.on('dialog', (dialog) => dialog.accept());
		await page.route('**/api/v1/**', async (route) => {
			const req = route.request(),
				path = new URL(req.url()).pathname.replace('/api/v1', '');
			const send = (json, status = 200) => route.fulfill({ json, status });
			if (path === '/me') return send({ user: users[role] });
			if (path === '/auth/csrf') return send({ csrf_token: 'fixture-csrf' });
			if (req.method() !== 'GET') {
				assert.equal(req.headers()['x-csrf-token'], 'fixture-csrf');
				const body = req.postDataJSON();
				mutations.push({ path, body });
				if (path.endsWith('/cancel')) {
					if (path.startsWith('/lab-requests')) labStatus = 'cancelled';
					else prescriptionStatus = 'cancelled';
				}
				if (path === '/lab-requests') labStatus = 'pending_patient_approval';
				if (path === '/prescriptions') prescriptionStatus = 'pending_patient_approval';
				return send({ status: 'cancelled' });
			}
			if (path === '/notifications')
				return bootstrapFailure
					? send({ error: { message: 'Temporary outage' } }, 503)
					: send({ notifications: [] });
			if (path === '/lab-requests')
				return send({
					lab_requests: [
						{
							id,
							episode_id: id,
							patient_ref: 'SYNTHETIC-001',
							test_code: 'DEMO',
							test_name: 'Synthetic lab',
							status: labStatus,
							result_id: labStatus === 'completed' ? resultId : null,
							cancellation_note: 'Provider unavailable'
						}
					]
				});
			if (path === '/prescriptions')
				return send({
					prescriptions: [
						{
							id,
							episode_id: id,
							patient_ref: 'SYNTHETIC-001',
							status: prescriptionStatus,
							items: [
								{
									medication_name: 'Synthetic medicine',
									dosage: 'demo',
									instructions: 'test only',
									quantity: 1
								}
							]
						}
					]
				});
			if (path === '/lab-receipts')
				return send({
					receipts: [
						{
							id: resultId,
							lab_request_id: id,
							patient_ref: 'SYNTHETIC-001',
							test_name: 'Synthetic lab',
							submitted_at: '2026-10-29T00:00:00Z'
						}
					]
				});
			if (path === `/lab-results/${resultId}`)
				return resultDenied
					? send({ error: { message: 'Access was revoked' } }, 403)
					: send({
							result: {
								id: resultId,
								test_name: 'Synthetic lab',
								value: 'OLD',
								unit: 'demo',
								corrections: corrected
									? [
											{
												id,
												value: 'NEWEST',
												unit: 'demo',
												reason: 'Instrument corrected',
												corrected_at: '2026-10-29T01:00:00Z'
											}
										]
									: []
							}
						});
			if (path === '/laboratories')
				return send({ laboratories: [{ id, name: 'Replacement laboratory' }] });
			return send({});
		});
		return page;
	}
	async function emit(page, type = 'notification') {
		await page.waitForFunction(() => window.__streams.some((stream) => !stream.closed));
		await page.evaluate((type) => {
			const stream = window.__streams.find((stream) => !stream.closed);
			if (type === 'open') stream.onopen?.();
			else stream.dispatchEvent(new MessageEvent(type, { data: '{"id":"1"}' }));
		}, type);
	}
	const patient = await pageFor('patient');
	await patient.goto(`${base}/dashboard/patient?view=labs`);
	await patient
		.getByText('Tidak dapat melayani', { exact: true })
		.count()
		.then((count) => assert.equal(count, 0));
	await patient.getByText('Batalkan layanan', { exact: true }).click();
	await patient
		.getByLabel('Alasan pembatalan', { exact: true })
		.fill('Draft must survive live updates');
	await emit(patient);
	await patient.waitForTimeout(500);
	assert.equal(
		await patient.getByLabel('Alasan pembatalan', { exact: true }).inputValue(),
		'Draft must survive live updates'
	);
	await patient.setViewportSize({ width: 390, height: 844 });
	assert.ok(await patient.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
	await patient.screenshot({
		path: '.test-results/workflow-recovery/cancellation-mobile.png',
		fullPage: true
	});
	await patient.setViewportSize({ width: 1440, height: 1000 });
	await patient.getByRole('button', { name: 'Konfirmasi pembatalan layanan', exact: true }).click();
	await patient.locator('.health-status').getByText('Dibatalkan', { exact: true }).waitFor();
	assert.equal(mutations.at(-1).path, `/lab-requests/${id}/cancel`);
	console.log('PASS cancellation with reason/CSRF, realtime refresh preserves draft');

	labStatus = 'completed';
	await emit(patient);
	await patient.getByRole('button', { name: 'Buka hasil pemeriksaan', exact: true }).click();
	await patient.getByRole('dialog').getByText('OLD demo', { exact: true }).waitFor();
	assert.equal(
		await patient.evaluate(() => window.__streams.filter((stream) => !stream.closed).length),
		1
	);
	corrected = true;
	await emit(patient);
	await patient
		.getByRole('dialog')
		.locator('dd')
		.getByText('NEWEST demo', { exact: true })
		.waitFor();
	await patient.getByText('Nilai terbaru setelah koreksi', { exact: true }).waitFor();
	await patient.screenshot({ path: '.test-results/workflow-recovery/corrected-result.png' });
	assert.equal(
		await patient.getByRole('dialog').getByText('OLD demo ·', { exact: true }).isVisible(),
		false
	);
	await patient.getByText('Riwayat hasil dan koreksi', { exact: true }).click();
	assert.ok((await patient.getByRole('dialog').textContent()).includes('OLD demo'));
	resultDenied = true;
	await emit(patient);
	await patient
		.getByRole('dialog')
		.getByRole('alert')
		.getByText('Access was revoked', { exact: true })
		.waitFor();
	assert.equal(await patient.getByText('NEWEST demo', { exact: true }).count(), 0);
	await patient.getByRole('button', { name: 'Tutup', exact: true }).click();
	console.log(
		'PASS one shared stream, corrected result refreshes open detail and separates original'
	);

	const lab = await pageFor('lab');
	await lab.goto(`${base}/dashboard/lab?view=receipts`);
	await lab.getByText(resultId, { exact: true }).waitFor();
	await lab.reload();
	await lab.getByText(resultId, { exact: true }).waitFor();
	await lab.setViewportSize({ width: 390, height: 844 });
	assert.ok(await lab.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
	await lab.screenshot({
		path: '.test-results/workflow-recovery/receipts-mobile.png',
		fullPage: true
	});
	await lab.getByRole('link', { name: 'Koreksi hasil ini', exact: true }).click();
	assert.equal(
		await lab.getByLabel('ID hasil pemeriksaan', { exact: true }).inputValue(),
		resultId
	);
	console.log('PASS lab receipts survive reload and open correction with the right result ID');

	const doctor = await pageFor('doctor');
	labStatus = 'cancelled';
	await doctor.goto(`${base}/dashboard/doctor?view=labs`);
	await doctor.getByText('Ajukan pemeriksaan pengganti', { exact: true }).click();
	await doctor.getByLabel('Laboratorium pengganti', { exact: true }).selectOption(id);
	await doctor.getByRole('button', { name: 'Kirim pemeriksaan pengganti', exact: true }).click();
	await doctor
		.locator('.health-status')
		.getByText('Menunggu persetujuan pasien', { exact: true })
		.waitFor();
	assert.deepEqual(mutations.at(-1), {
		path: '/lab-requests',
		body: { episode_id: id, lab_org_id: id, test_code: 'DEMO', test_name: 'Synthetic lab' }
	});
	prescriptionStatus = 'cancelled';
	await doctor.goto(`${base}/dashboard/doctor?view=prescriptions`);
	await doctor.getByText('Terbitkan resep pengganti', { exact: true }).click();
	await doctor
		.getByRole('button', { name: 'Terbitkan resep baru untuk dipilih apoteknya', exact: true })
		.click();
	await doctor
		.locator('.health-status')
		.getByText('Menunggu persetujuan pasien', { exact: true })
		.waitFor();
	assert.equal(mutations.at(-1).body.items[0].medication_name, 'Synthetic medicine');
	console.log('PASS replacement lab/prescription uses a new request requiring patient approval');

	// Reconnect must refresh even if no new event is replayed.
	labStatus = 'sent';
	await emit(patient, 'open');
	await patient.locator('.health-status').getByText('Terkirim', { exact: true }).waitFor();
	// Stop the other pages before fake-clock tests; assert timer bootstrap retry separately.
	const fallback = await pageFor('patient');
	bootstrapFailure = true;
	await fallback.clock.install();
	const failedBootstrap = fallback.waitForResponse(
		(response) => response.url().includes('/notifications?limit=1') && response.status() === 503
	);
	await fallback.goto(`${base}/dashboard/patient?view=labs`);
	await fallback.locator('.health-status').getByText('Terkirim', { exact: true }).waitFor();
	await failedBootstrap;
	bootstrapFailure = false;
	await fallback.clock.runFor(5100);
	await fallback.waitForFunction(() => window.__streams.some((stream) => !stream.closed));
	labStatus = 'accepted';
	await fallback.clock.runFor(30100);
	await fallback.locator('.health-status').getByText('Diterima', { exact: true }).waitFor();
	await fallback.goto(`${base}/login`);
	assert.equal(
		await fallback.evaluate(() => window.__streams.filter((stream) => !stream.closed).length),
		0
	);
	assert.deepEqual(errors, []);
	console.log(
		'PASS reconnect refresh, failed bootstrap retries, fallback polling and stream cleanup'
	);
} catch (error) {
	for (const context of browser?.contexts() || []) {
		for (const page of context.pages())
			console.error(page.url(), (await page.locator('body').innerText()).slice(-4000));
	}
	console.error('Browser errors:', errors);
	throw error;
} finally {
	await browser?.close();
	vite.kill();
	await new Promise((resolve) => server.close(resolve));
}
