import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { mkdir } from 'node:fs/promises';
import { chromium } from 'playwright';

// Local fixtures exercise the UI without reading or changing patient records.
const user = {
	id: 'visual-test',
	full_name: 'Pasien Demo Visual',
	email: 'visual@example.test',
	role: 'patient',
	organization: null
};
const api = createServer((request, response) => {
	response.setHeader('Content-Type', 'application/json');
	if (request.url === '/api/v1/me') response.end(JSON.stringify({ user }));
	else {
		response.statusCode = 404;
		response.end('{}');
	}
});
await new Promise((resolve) => api.listen(18081, '127.0.0.1', resolve));
const vite = spawn(
	process.execPath,
	['node_modules/vite/bin/vite.js', 'dev', '--host', '127.0.0.1', '--port', '5177', '--strictPort'],
	{
		env: { ...process.env, API_BASE_URL: 'http://127.0.0.1:18081' },
		stdio: ['ignore', 'pipe', 'pipe'],
		windowsHide: true
	}
);
let output = '';
vite.stdout.on('data', (data) => {
	output += data;
});
vite.stderr.on('data', (data) => {
	output += data;
});
let browser;
try {
	await new Promise((resolve, reject) => {
		const deadline = Date.now() + 30000;
		const poll = setInterval(() => {
			// eslint-disable-next-line no-control-regex
			if (output.replace(/\x1b\[[0-9;]*m/g, '').includes('http://127.0.0.1:5177/')) {
				clearInterval(poll);
				resolve();
			} else if (Date.now() > deadline || vite.exitCode !== null) {
				clearInterval(poll);
				reject(new Error(output || 'Vite did not start'));
			}
		}, 100);
	});
	browser = await chromium.launch({
		...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {}),
		headless: true
	});
	const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
	await context.addCookies([
		{ name: 'hygeia_session', value: 'visual-fixture', url: 'http://127.0.0.1:5177' }
	]);
	const page = await context.newPage();
	const errors = [];
	page.on('pageerror', (error) => errors.push(error.message));
	let mode = 'populated';
	await page.route('**/api/v1/**', async (route) => {
		const path = new URL(route.request().url()).pathname;
		if (mode === 'error') {
			await route.fulfill({ status: 503, json: {} });
			return;
		}
		const populated = mode === 'populated';
		const bodies = {
			'/api/v1/episodes': {
				episodes: populated
					? [
							{
								id: 'episode-fixture',
								doctor_name: 'Dokter Demo Visual',
								organization_name: 'Klinik Demo',
								status: 'active'
							}
						]
					: []
			},
			'/api/v1/lab-requests': {
				lab_requests: populated
					? [
							{
								id: 'lab-fixture',
								test_name: 'Pemeriksaan sintetis',
								lab_name: 'Laboratorium Demo',
								status: 'in_progress'
							}
						]
					: []
			},
			'/api/v1/prescriptions': {
				prescriptions: populated
					? [
							{
								id: 'prescription-fixture',
								status: 'ready',
								pharmacy_name: 'Apotek Demo',
								items: [
									{
										medication_name: 'Obat sintetis demo',
										dosage: 'Dosis demo',
										instructions: 'Aturan pakai sintetis untuk pengujian tampilan.',
										quantity: 2
									}
								]
							}
						]
					: []
			},
			'/api/v1/patient/timeline': {
				events: populated
					? [
							{
								seq: 2,
								kind: 'prescription',
								occurred_at: '2026-10-14T03:00:00Z',
								message: 'Apotek Demo menandai resep sintetis siap diambil.'
							},
							{
								seq: 1,
								kind: 'episode',
								occurred_at: '2026-10-13T03:00:00Z',
								message: 'Anda menerima undangan perawatan dari Dokter Demo Visual.'
							}
						]
					: []
			}
		};
		await route.fulfill({ json: bodies[path] || {} });
	});
	await mkdir('.test-results', { recursive: true });
	const landingResponse = await page.goto('http://127.0.0.1:5177/');
	assert.equal(landingResponse.status(), 200);
	assert.equal(new URL(page.url()).pathname, '/');
	await page.locator('.hero-phone svg > image').waitFor();
	await page.locator('.audience-options button:not([disabled])').first().waitFor();
	for (const [label, title] of [
		['Dokter', 'Lanjutkan perawatan dengan konteks yang jelas.'],
		['Laboratorium', 'Dari permintaan menjadi hasil pemeriksaan.'],
		['Apotek', 'Siapkan resep yang ditujukan kepada Anda.'],
		['Pasien', 'Pahami apa yang sedang berjalan.']
	]) {
		const button = page
			.locator('.audience-options')
			.getByRole('button', { name: label, exact: true });
		await button.focus();
		await button.press('Enter');
		await page
			.locator('.audience-detail')
			.getByRole('heading', { name: title, exact: true })
			.waitFor();
		assert.equal(await button.getAttribute('aria-pressed'), 'true');
	}
	for (const detail of await page.locator('.faq-list details').all()) {
		const summary = detail.locator('summary');
		await summary.focus();
		await summary.press('Enter');
		assert.equal(await detail.evaluate((element) => element.open), true);
		await summary.press('Enter');
		assert.equal(await detail.evaluate((element) => element.open), false);
	}
	assert.equal(await page.locator('.flow-steps li').count(), 5);
	for (const [label, hash] of [
		['Untuk siapa', '#untuk-siapa'],
		['Izin data', '#izin-data'],
		['Pertanyaan', '#pertanyaan']
	]) {
		await page
			.getByRole('navigation', { name: 'Informasi tambahan' })
			.getByRole('link', { name: label, exact: true })
			.click();
		assert.equal(new URL(page.url()).hash, hash);
	}
	console.log(
		'PASS four role views and five FAQ disclosures by keyboard, complete care flow, footer navigation'
	);
	assert.equal(
		(await page.request.get('http://127.0.0.1:5177/hero-phone-front.png')).status(),
		200
	);
	for (const [label, hash] of [
		['Fitur', '#fitur'],
		['Cara kerja', '#cara-kerja'],
		['Tentang', '#tentang']
	]) {
		await page
			.getByRole('navigation', { name: 'Navigasi beranda' })
			.getByRole('link', { name: label, exact: true })
			.click();
		assert.equal(new URL(page.url()).hash, hash);
	}
	for (const width of [1440, 1024, 768, 414, 390, 375, 320]) {
		await page.setViewportSize({ width, height: 1000 });
		assert.ok(
			await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
			`landing overflow at ${width}`
		);
		assert.ok(
			await page
				.locator('.landing a, .landing button')
				.evaluateAll((elements) =>
					elements.every((element) => getComputedStyle(element).whiteSpace === 'nowrap')
				),
			`single-line controls at ${width}`
		);
		assert.ok(
			await page
				.locator('.landing-header a, .hero-actions a')
				.evaluateAll((elements) =>
					elements.every((element) => element.getBoundingClientRect().height >= 44)
				),
			`touch target size at ${width}`
		);
		await page.evaluate(() => window.scrollTo(0, 0));
		await page.screenshot({ path: `.test-results/landing-${width}.png`, fullPage: true });
		if (width === 1440 || width === 375) {
			await page
				.locator('.landing-audiences')
				.screenshot({ path: `.test-results/landing-roles-${width}.png` });
			await page
				.locator('.landing-faq')
				.screenshot({ path: `.test-results/landing-faq-${width}.png` });
		}
	}
	await page.getByRole('link', { name: 'Masuk ke ruang Anda', exact: true }).click();
	await page.waitForURL('**/login');
	assert.equal(new URL(page.url()).pathname, '/login');
	console.log(
		'PASS public landing, supplied phone asset, section navigation, login CTA, seven screen widths and touch targets'
	);
	await page.goto('http://127.0.0.1:5177/dashboard/patient');
	await page.getByRole('heading', { name: 'Dokter Demo Visual', exact: true }).waitFor();
	await page.locator('summary').click();
	await page.getByRole('heading', { name: 'Obat sintetis demo' }).waitFor();
	await page.getByRole('button', { name: 'Resep', exact: true }).click();
	assert.equal(await page.locator('.patient-timeline li').count(), 1);
	await page.getByRole('button', { name: 'Semua', exact: true }).click();
	assert.equal(await page.locator('.patient-timeline li').count(), 2);
	await page.locator('nav').getByRole('link', { name: 'Resep & apotek' }).click();
	await page.waitForURL('**?view=prescriptions');
	assert.equal(new URL(page.url()).searchParams.get('view'), 'prescriptions');
	await mkdir('.test-results', { recursive: true });
	for (const width of [1440, 1024, 768, 390, 320]) {
		await page.setViewportSize({ width, height: 1000 });
		assert.ok(
			await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
			`overflow at ${width}`
		);
		await page.evaluate(() => window.scrollTo(0, 0));
		await page.screenshot({ path: `.test-results/dashboard-patient-${width}.png`, fullPage: true });
	}
	console.log(
		'PASS fixture data, medication details, timeline filters, navigation, five screen widths'
	);
	mode = 'empty';
	await page.goto('http://127.0.0.1:5177/dashboard/patient');
	await page.getByRole('button', { name: 'Perbarui data' }).click();
	await page.getByRole('heading', { name: 'Belum ada resep', exact: true }).waitFor();
	await page.getByRole('heading', { name: 'Perjalanan Anda dimulai di sini.' }).waitFor();
	console.log('PASS empty data states');
	mode = 'error';
	await page.getByRole('button', { name: 'Perbarui data' }).click();
	await page.getByRole('button', { name: 'Coba lagi' }).first().waitFor();
	assert.equal(await page.getByRole('alert').count(), 4);
	mode = 'empty';
	await page.getByRole('button', { name: 'Coba lagi' }).first().click();
	await page.getByRole('heading', { name: 'Belum ada resep', exact: true }).waitFor();
	assert.deepEqual(errors, []);
	console.log('PASS API failures, retry recovery, no browser runtime errors');
} finally {
	await browser?.close();
	vite.kill();
	await new Promise((resolve) => api.close(resolve));
}
