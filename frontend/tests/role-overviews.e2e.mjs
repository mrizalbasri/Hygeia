import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { mkdir } from 'node:fs/promises';
import { chromium } from 'playwright';
import { waitForServer } from './server-ready.js';

const users = {
	doctor: {
		id: 'doctor-fixture',
		role: 'doctor',
		full_name: 'Dr. Maya Sintetis',
		email: 'doctor@example.test',
		organization: { id: 'clinic-fixture', name: 'Klinik Demo', type: 'clinic' }
	},
	lab: {
		id: 'lab-fixture',
		role: 'lab',
		full_name: 'Petugas Laboratorium Sintetis',
		email: 'lab@example.test',
		organization: { id: 'lab-fixture', name: 'Laboratorium Demo', type: 'laboratory' }
	}
};
const fixture = createServer((req, res) => {
	const role = req.headers.cookie?.includes('hygeia_session=fixture-lab') ? 'lab' : 'doctor';
	res.setHeader('Content-Type', 'application/json');
	res.end(JSON.stringify({ user: users[role] }));
});
await new Promise((resolve) => fixture.listen(0, '127.0.0.1', resolve));
const portProbe = createServer();
await new Promise((resolve) => portProbe.listen(0, '127.0.0.1', resolve));
const port = Number(process.env.HYGEIA_ROLES_PORT || portProbe.address().port);
await new Promise((resolve) => portProbe.close(resolve));
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
let output = '';
vite.stdout.on('data', (chunk) => (output += chunk));
vite.stderr.on('data', (chunk) => (output += chunk));
let browser;
try {
	await waitForServer(base, { output: () => output, isAlive: () => vite.exitCode === null });
	browser = await chromium.launch({
		headless: true,
		...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {})
	});
	await mkdir('.test-results', { recursive: true });
	for (const role of ['doctor', 'lab']) {
		const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
		await context.addCookies([{ name: 'hygeia_session', value: `fixture-${role}`, url: base }]);
		const page = await context.newPage();
		const errors = [],
			calls = [];
		let mode = 'populated';
		page.on('pageerror', (error) => errors.push(error.message));
		await page.route('**/api/v1/**', async (route) => {
			const request = route.request();
			const path = new URL(request.url()).pathname.replace('/api/v1', '');
			assert.equal(request.method(), 'GET', 'overview only reads existing resources');
			// The shared dashboard owns notifications; clinical reads remain role-specific.
			if (path === '/notifications/unread-count')
				return route.fulfill({ json: { unread_count: 0 } });
			if (path === '/notifications') return route.fulfill({ json: { notifications: [] } });
			if (path === '/notifications/stream') return route.abort();
			calls.push(path);
			if (path === '/me') return route.fulfill({ json: { user: users[role] } });
			if (mode === 'error')
				return route.fulfill({
					status: 503,
					json: { error: { message: 'Ringkasan belum dapat dimuat.' } }
				});
			const lists = {
				episodes: [
					{
						id: 'episode-1',
						patient_ref: 'HYG-SYNTHETIC-001',
						organization_name: 'Klinik Demo',
						doctor_name: 'Dr. Maya Sintetis',
						status: 'active'
					},
					{ id: 'episode-2', patient_ref: 'HYG-SYNTHETIC-002', status: 'pending' }
				],
				lab_requests:
					role === 'lab'
						? ['sent', 'accepted', 'in_progress'].map((status, index) => ({
								id: `lab-${index}`,
								test_name: `Pemeriksaan sintetis ${index + 1}`,
								test_code: `TEST-${index + 1}`,
								patient_ref: `HYG-LAB-00${index + 1}`,
								status
							}))
						: [
								{
									id: 'lab-doctor-1',
									test_name: 'Pemeriksaan sintetis dokter',
									patient_ref: 'HYG-SYNTHETIC-001',
									lab_name: 'Laboratorium Demo',
									status: 'completed',
									result_id: 'result-fixture-1'
								}
							],
				prescriptions: [
					{
						id: 'prescription-1',
						patient_ref: 'HYG-SYNTHETIC-001',
						status: 'sent',
						pharmacy_name: 'Apotek Demo',
						items: [
							{
								medication_name: 'Obat sintetis',
								dosage: 'Demo',
								instructions: 'Demo',
								quantity: 1
							}
						]
					}
				]
			};
			const key = {
				'/episodes': 'episodes',
				'/lab-requests': 'lab_requests',
				'/prescriptions': 'prescriptions'
			}[path];
			assert.ok(key, `unexpected overview API: ${path}`);
			await route.fulfill({ json: { [key]: mode === 'empty' ? [] : lists[key] } });
		});
		const title = role === 'doctor' ? 'Ringkasan dokter' : 'Ringkasan laboratorium';
		await page.goto(`${base}/dashboard/${role}`);
		try {
			await page.getByRole('heading', { name: title, exact: true, level: 1 }).waitFor();
		} catch (error) {
			throw new Error(
				`${role}: ${page.url()} API calls: ${calls.join(', ')} Browser errors: ${errors.join('; ')} Vite: ${output.slice(-4000)}`,
				{ cause: error }
			);
		}
		await page
			.getByRole('heading', {
				name: role === 'doctor' ? 'HYG-SYNTHETIC-001' : 'Pemeriksaan sintetis 1',
				exact: true
			})
			.first()
			.waitFor();
		assert.deepEqual(
			await page.locator('.overview-metric strong').allTextContents(),
			role === 'doctor' ? ['1', '1', '1', '1'] : ['1', '1', '1']
		);
		const menu = page.getByRole('navigation', { name: 'Menu utama' });
		if (role === 'doctor') {
			assert.equal(await menu.getByRole('link', { name: 'Koreksi hasil', exact: true }).count(), 0);
			assert.equal(
				await page.getByRole('button', { name: 'Terima permintaan', exact: true }).count(),
				0
			);
			assert.deepEqual([...new Set(calls)].sort(), [
				'/episodes',
				'/lab-requests',
				'/prescriptions'
			]);
			await page.getByRole('link', { name: 'Undangan & perawatan pasien', exact: true }).click();
			await page.getByText('Undang pasien', { exact: true }).waitFor();
		} else {
			assert.equal(
				await menu.getByRole('link', { name: 'Pasien & perawatan', exact: true }).count(),
				0
			);
			assert.equal(await menu.getByRole('link', { name: 'Resep', exact: true }).count(), 0);
			assert.deepEqual([...new Set(calls)], ['/lab-requests']);
			await page.getByRole('link', { name: 'Buka antrean pemeriksaan', exact: true }).click();
			for (const name of ['Terima permintaan', 'Mulai pemeriksaan', 'Kirim hasil pemeriksaan'])
				await page.getByRole('button', { name, exact: true }).waitFor();
		}
		await page.goto(`${base}/dashboard/${role === 'doctor' ? 'lab' : 'doctor'}`);
		await page.waitForURL(`**/dashboard/${role}`);
		await page.getByRole('heading', { name: title, exact: true, level: 1 }).waitFor();
		await page
			.getByRole('heading', {
				name: role === 'doctor' ? 'HYG-SYNTHETIC-001' : 'Pemeriksaan sintetis 1',
				exact: true
			})
			.first()
			.waitFor();
		for (const width of [1440, 1024, 768, 375, 320]) {
			await page.setViewportSize({ width, height: 1000 });
			assert.ok(
				await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
				`${role} overflow at ${width}`
			);
			await page.screenshot({ path: `.test-results/role-${role}-${width}.png`, fullPage: true });
		}
		mode = 'empty';
		await page.getByRole('button', { name: 'Perbarui ringkasan', exact: true }).click();
		await page
			.getByRole('heading', {
				name: role === 'doctor' ? 'Belum ada perawatan pasien' : 'Belum ada penugasan pemeriksaan',
				exact: true
			})
			.waitFor();
		assert.deepEqual(
			await page.locator('.overview-metric strong').allTextContents(),
			role === 'doctor' ? ['0', '0', '0', '0'] : ['0', '0', '0']
		);
		mode = 'error';
		await page.getByRole('button', { name: 'Perbarui ringkasan', exact: true }).click();
		await page.getByRole('button', { name: 'Coba lagi', exact: true }).first().waitFor();
		assert.equal(await page.getByRole('alert').count(), role === 'doctor' ? 3 : 1);
		assert.deepEqual(
			await page.locator('.overview-metric strong').allTextContents(),
			role === 'doctor' ? ['—', '—', '—', '—'] : ['—', '—', '—']
		);
		mode = 'populated';
		await page.getByRole('button', { name: 'Coba lagi', exact: true }).first().click();
		await page
			.getByRole('heading', {
				name: role === 'doctor' ? 'HYG-SYNTHETIC-001' : 'Pemeriksaan sintetis 1',
				exact: true
			})
			.first()
			.waitFor();
		assert.deepEqual(errors, []);
		await context.close();
		console.log(
			`PASS ${role} overview data, own-role menus, workflow links, cross-role URL redirect, empty/error/retry, five widths`
		);
	}
} finally {
	await browser?.close();
	vite.kill();
	await new Promise((resolve) => fixture.close(resolve));
}
