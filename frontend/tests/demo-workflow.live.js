import assert from 'node:assert/strict';
import { readFile, mkdir } from 'node:fs/promises';
import { spawn } from 'node:child_process';
import { chromium, request } from 'playwright';
import { waitForServer } from './server-ready.js';

// Explicit local opt-in. Requires go run ./cmd/seed-workflow; never real records.
// Creates sessions and audit reads, but does not reset or advance demo workflows.
if (process.env.HYGEIA_LIVE_TEST !== '1')
	throw new Error('Set HYGEIA_LIVE_TEST=1 for the local synthetic demo check.');
const settings = await readFile('../backend/.env', 'utf8');
const password = settings
	.match(/^DEMO_PASSWORD=(.*)$/m)?.[1]
	?.trim()
	.replace(/^['"]|['"]$/g, '');
assert.ok(password, 'Local demo password required');
const backend = 'http://127.0.0.1:8080',
	base = 'http://127.0.0.1:5186';
const vite = spawn(
	process.execPath,
	['node_modules/vite/bin/vite.js', 'dev', '--host', '127.0.0.1', '--port', '5186', '--strictPort'],
	{
		env: { ...process.env, API_BASE_URL: backend },
		stdio: ['ignore', 'pipe', 'pipe'],
		windowsHide: true
	}
);
let browser,
	output = '';
vite.stdout.on('data', (chunk) => (output += chunk));
vite.stderr.on('data', (chunk) => (output += chunk));
try {
	await waitForServer(base, { output: () => output, isAlive: () => vite.exitCode === null });
	browser = await chromium.launch({
		headless: true,
		...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {})
	});
	await mkdir('.test-results/pilot-stage1', { recursive: true });
	for (const role of ['patient', 'doctor', 'lab', 'pharmacy', 'admin']) {
		const client = await request.newContext();
		let context;
		try {
			const csrf = await client.get(`${backend}/api/v1/auth/csrf`);
			const login = await client.post(`${backend}/api/v1/auth/login`, {
				data: { email: `pilot.${role}@demo.hygeia.test`, password },
				headers: { Origin: backend, 'X-CSRF-Token': (await csrf.json()).csrf_token }
			});
			assert.equal(login.status(), 200, `${role} login failed`);
			assert.equal((await login.json()).user.role, role);
			context = await browser.newContext({
				storageState: await client.storageState(),
				viewport: { width: 1440, height: 1000 },
				permissions: ['clipboard-read', 'clipboard-write']
			});
			const page = await context.newPage(),
				errors = [];
			page.on('pageerror', (error) => errors.push(error.message));
			const listPath =
				role === 'admin'
					? '/admin/organizations'
					: role === 'pharmacy'
						? '/prescriptions'
						: '/lab-requests';
			const loaded = page.waitForResponse(
				(response) => response.url().includes(`/api/v1${listPath}`) && response.status() === 200
			);
			await page.goto(`${base}/dashboard/${role}`);
			await loaded;
			await page.locator('h1').last().waitFor();
			if (role === 'patient') {
				await page.getByRole('button', { name: 'Salin kode pasien' }).click();
				await page.getByText('Kode pasien berhasil disalin.', { exact: true }).waitFor();
				assert.equal(
					await page.evaluate(() => navigator.clipboard.readText()),
					'HYG-PILOT-SYNTH-001'
				);
			}
			for (const width of [1440, 390]) {
				await page.setViewportSize({ width, height: width === 390 ? 844 : 1000 });
				assert.ok(
					await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
					`${role} overflow at ${width}`
				);
				await page.screenshot({
					path: `.test-results/pilot-stage1/${role}-${width}.png`,
					fullPage: true
				});
			}
			// Verify a real API list for every clinical role, beyond a styled shell.
			if (role !== 'admin') {
				const response = await client.get(`${backend}/api/v1${listPath}?limit=20`);
				assert.equal(response.status(), 200);
				const payload = await response.json();
				assert.ok(
					(payload.lab_requests || payload.prescriptions).length >= 1,
					`${role} synthetic records missing`
				);
			}
			if (role === 'patient') {
				const timeline = await client.get(`${backend}/api/v1/patient/timeline?limit=20`);
				assert.ok((await timeline.json()).events.length >= 1, 'synthetic timeline missing');
				const audit = await client.get(`${backend}/api/v1/audit/verify`);
				assert.equal((await audit.json()).intact, true, 'synthetic audit integrity');
			}
			assert.deepEqual(errors, [], `${role} browser errors`);
			console.log(`PASS live synthetic ${role}: desktop/mobile and API`);
		} finally {
			await context?.close();
			await client.dispose();
		}
	}
} finally {
	await browser?.close();
	vite.kill();
}
