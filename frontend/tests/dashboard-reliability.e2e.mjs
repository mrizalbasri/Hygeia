import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { chromium } from 'playwright';
import { waitForServer } from './server-ready.js';

const user = {
	id: 'reliability-patient',
	full_name: 'Pasien Sintetis',
	email: 'reliability@example.test',
	role: 'patient',
	organization: null
};
const identity = createServer((request, response) => {
	response.setHeader('Content-Type', 'application/json');
	response.end(JSON.stringify({ user }));
});
await new Promise((resolve) => identity.listen(0, '127.0.0.1', resolve));
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
		env: { ...process.env, API_BASE_URL: `http://127.0.0.1:${identity.address().port}` },
		stdio: ['ignore', 'pipe', 'pipe'],
		windowsHide: true
	}
);
let output = '';
vite.stdout.on('data', (data) => (output += data));
vite.stderr.on('data', (data) => (output += data));
let browser;
try {
	await waitForServer(base, { output: () => output, isAlive: () => vite.exitCode === null });
	browser = await chromium.launch({
		...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {}),
		headless: true
	});
	const context = await browser.newContext();
	await context.addCookies([{ name: 'hygeia_session', value: 'fixture', url: base }]);
	// Deterministic transport exercises the real dashboard/controller/API wiring.
	// Existing clinical E2E separately verifies a native SSE session-ended event.
	await context.addInitScript(() => {
		window.fixtureStreams = [];
		window.EventSource = class extends EventTarget {
			constructor(url) {
				super();
				this.url = url;
				this.closed = false;
				window.fixtureStreams.push(this);
			}
			close() {
				this.closed = true;
				sessionStorage.setItem('notification-stream-closed', 'yes');
			}
			emit(type, data = {}) {
				if (this.closed) return;
				if (type === 'open') this.onopen?.(new Event('open'));
				else if (type === 'error') this.onerror?.(new Event('error'));
				else this.dispatchEvent(new MessageEvent(type, { data: JSON.stringify(data) }));
			}
		};
	});
	const page = await context.newPage();
	const errors = [];
	page.on('pageerror', (error) => errors.push(error.message));
	let unread = 2,
		active = 31,
		unavailable = false,
		summaryFailure = false,
		summaryReads = 0;
	await page.route('**/api/v1/**', (route) => {
		const request = route.request();
		const path = new URL(request.url()).pathname.replace('/api/v1', '');
		const send = (json, status = 200) => route.fulfill({ status, json });
		if (path === '/me') return send({ user });
		if (path === '/auth/csrf') return send({ csrf_token: 'fixture' });
		if (path === '/notifications/unread-count') return send({ unread_count: unread });
		if (path === '/notifications')
			return unavailable
				? send({}, 503)
				: send({
						notifications: [
							{
								id: '9223372036854775806',
								message: 'Izin dokter perlu ditinjau',
								read_at: unread ? null : '2026-10-26T00:00:00Z'
							}
						],
						next_before_id: null
					});
		if (path.startsWith('/notifications/') && request.method() === 'PATCH') {
			unread = 0;
			return send({});
		}
		if (path === '/patient/work-summary') {
			summaryReads++;
			return summaryFailure
				? send({}, 503)
				: send({
						summary: {
							active_episodes: active,
							labs_in_progress: 22,
							prescriptions_ready: 8,
							pending: { episodes: 25, lab_requests: 21, prescriptions: 3, consents: 7 }
						}
					});
		}
		const key = {
			'/episodes': 'episodes',
			'/lab-requests': 'lab_requests',
			'/prescriptions': 'prescriptions',
			'/consents': 'consents',
			'/patient/timeline': 'events'
		}[path];
		return send(key ? { [key]: [] } : {});
	});
	await page.goto(`${base}/dashboard/patient`);
	await page.getByRole('link', { name: 'Buka notifikasi; 2 belum dibaca', exact: true }).waitFor();
	await page.waitForFunction(() => window.fixtureStreams.length === 1);
	assert.deepEqual(await page.locator('.overview-metric strong').allTextContents(), [
		'31',
		'56',
		'22',
		'8'
	]);
	assert.equal(await page.locator('.pending-list a').count(), 4);
	await page.evaluate(() => window.fixtureStreams[0].emit('open'));
	await page.waitForFunction(
		() => document.querySelector('.overview-metric strong')?.textContent === '31'
	);
	// A live event updates the overview while the notification page is not open.
	const readsBefore = summaryReads;
	unread = 3;
	active = 32;
	await page.evaluate(() => window.fixtureStreams[0].emit('notification', { id: 'new' }));
	await page.waitForFunction(
		() => document.querySelector('.overview-metric strong')?.textContent === '32'
	);
	await page.getByRole('link', { name: 'Buka notifikasi; 3 belum dibaca', exact: true }).waitFor();
	assert.ok(summaryReads > readsBefore);
	assert.ok(
		await page.evaluate(() => window.fixtureStreams[0].url.endsWith('after_id=9223372036854775806'))
	);
	await page.getByRole('link', { name: 'Buka notifikasi; 3 belum dibaca', exact: true }).click();
	await page.getByText('3 notifikasi belum dibaca.', { exact: true }).waitFor();
	assert.equal(
		await page.evaluate(() => window.fixtureStreams.length),
		1,
		'menu navigation must not create a second SSE'
	);
	await page.getByRole('button', { name: 'Tandai dibaca', exact: true }).click();
	await page.getByRole('link', { name: 'Buka notifikasi; 0 belum dibaca', exact: true }).waitFor();
	assert.equal(await page.locator('.notification-badge').count(), 0);
	// Reconnect catches up unread state even without a replayed notification.
	await page.evaluate(() => window.fixtureStreams[0].emit('error'));
	await page.getByRole('button', { name: 'Coba lagi notifikasi' }).waitFor();
	unread = 5;
	await page.evaluate(() => window.fixtureStreams[0].emit('open'));
	await page.getByRole('link', { name: 'Buka notifikasi; 5 belum dibaca', exact: true }).waitFor();
	assert.equal(await page.getByRole('button', { name: 'Coba lagi notifikasi' }).count(), 0);
	await page.getByRole('navigation').getByRole('link', { name: 'Ringkasan', exact: true }).click();
	await page.locator('.pending-list a').nth(3).waitFor();
	summaryFailure = true;
	await page.getByRole('button', { name: 'Perbarui data', exact: true }).click();
	await page.getByRole('heading', { name: 'Ringkasan belum lengkap' }).waitFor();
	assert.deepEqual(await page.locator('.overview-metric strong').allTextContents(), [
		'—',
		'—',
		'—',
		'—'
	]);
	summaryFailure = false;
	await page.getByRole('button', { name: 'Coba lagi ringkasan', exact: true }).click();
	await page.locator('.pending-list a').nth(3).waitFor();
	for (const width of [1440, 375, 320]) {
		await page.setViewportSize({ width, height: 900 });
		assert.ok(
			await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
			`overflow ${width}`
		);
	}
	// Explicit retry closes the old transport before starting its replacement.
	await page.evaluate(() => window.fixtureStreams[0].emit('error'));
	await page.getByRole('button', { name: 'Coba lagi notifikasi' }).click();
	await page.waitForFunction(() => window.fixtureStreams.length === 2);
	assert.equal(await page.evaluate(() => window.fixtureStreams[0].closed), true);
	assert.equal(
		await page.evaluate(() => window.fixtureStreams.filter((stream) => !stream.closed).length),
		1
	);
	// Session expiry works from the overview, not only from the inbox.
	await page.evaluate(() => window.fixtureStreams[1].emit('session-ended'));
	await page.waitForURL('**/login');
	assert.equal(
		await page.evaluate(() => sessionStorage.getItem('notification-stream-closed')),
		'yes'
	);
	// Initial API failure has a working, explicit recovery path.
	unavailable = true;
	await page.goto(`${base}/dashboard/patient`);
	await page.getByRole('button', { name: 'Coba lagi notifikasi' }).waitFor();
	assert.equal(await page.evaluate(() => window.fixtureStreams.length), 0);
	unavailable = false;
	await page.getByRole('button', { name: 'Coba lagi notifikasi' }).click();
	await page.waitForFunction(() => window.fixtureStreams.length === 1);
	assert.deepEqual(errors, []);
	console.log(
		'PASS global notification badge, single stream, reconnect/retry, complete summary, session expiry, desktop/mobile'
	);
} finally {
	await browser?.close();
	vite.kill();
	await new Promise((resolve) => identity.close(resolve));
}
