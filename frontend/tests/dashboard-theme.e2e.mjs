import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { mkdir } from 'node:fs/promises';
import { chromium } from 'playwright';
import { waitForServer } from './server-ready.js';

const webPort = Number(process.env.HYGEIA_VISUAL_PORT || '5177');
const base = `http://127.0.0.1:${webPort}`;

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
await new Promise((resolve) => api.listen(0, '127.0.0.1', resolve));
const vite = spawn(
	process.execPath,
	[
		'node_modules/vite/bin/vite.js',
		'dev',
		'--host',
		'127.0.0.1',
		'--port',
		String(webPort),
		'--strictPort'
	],
	{
		env: { ...process.env, API_BASE_URL: `http://127.0.0.1:${api.address().port}` },
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
	await waitForServer(base, {
		// eslint-disable-next-line no-control-regex
		output: () => output.replace(/\x1b\[[0-9;]*m/g, ''),
		isAlive: () => vite.exitCode === null
	});
	browser = await chromium.launch({
		...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {}),
		headless: true
	});
	const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
	await context.addCookies([{ name: 'hygeia_session', value: 'visual-fixture', url: base }]);
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
		const populated = mode === 'populated' || mode === 'pending';
		const bodies = {
			'/api/v1/episodes': {
				episodes: populated
					? [
							{
								id: 'episode-fixture',
								doctor_name: 'Dokter Demo Visual',
								organization_name: 'Klinik Demo',
								status: mode === 'pending' ? 'pending' : 'active'
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
								status: mode === 'pending' ? 'pending_patient_approval' : 'in_progress'
							}
						]
					: []
			},
			'/api/v1/prescriptions': {
				prescriptions: populated
					? [
							{
								id: 'prescription-fixture',
								status: mode === 'pending' ? 'pending_patient_approval' : 'ready',
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
	if (process.env.HYGEIA_DASHBOARD_ONLY !== '1') {
		const landingResponse = await page.goto(`${base}/`);
		assert.equal(landingResponse.status(), 200);
		assert.equal(new URL(page.url()).pathname, '/');
		await page.locator('.hero-three-backdrop').waitFor();
		assert.equal(await page.locator('.hero-three-roles a').count(), 4);
		assert.equal(
			new URL(await page.locator('.hero-three-actions a').first().getAttribute('href'), base)
				.pathname,
			'/login'
		);
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
			assert.ok(await page.locator('.audience-detail li').count());
		}
		await page.locator('.hero-three-backdrop').evaluate((element) => element.decode());
		for (const [index, title] of [
			[0, 'Anda menentukan siapa yang menerima.'],
			[1, 'Bagikan sesuai kebutuhan perawatan.'],
			[2, 'Setiap pihak melanjutkan bagiannya.']
		]) {
			const stage = page.locator('.permission-path button').nth(index);
			await stage.focus();
			await stage.press('Enter');
			await page.getByRole('heading', { name: title, exact: true }).waitFor();
			assert.equal(await stage.getAttribute('aria-pressed'), 'true');
		}
		assert.equal(await page.locator('.closing-primary').getAttribute('href'), '/register');
		assert.equal(await page.locator('.closing-secondary').getAttribute('href'), '/login');
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
		assert.equal((await page.request.get(`${base}/hero-clinic-v2.png`)).status(), 200);
		await page
			.getByRole('navigation', { name: 'Navigasi beranda' })
			.getByRole('button', { name: 'Fitur', exact: true })
			.focus();
		await page.getByRole('button', { name: 'Fitur', exact: true }).press('Enter');
		await page
			.locator('.nav-panel a')
			.filter({ has: page.getByText('Perawatan', { exact: true }) })
			.click();
		assert.equal(new URL(page.url()).hash, '#fitur');
		for (const [label, hash] of [
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
					.locator('.landing-header a, .hero-three-actions a')
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
		await page.getByRole('link', { name: /Masuk ke ruang Anda/ }).click();
		await page.waitForURL('**/login');
		assert.equal(new URL(page.url()).pathname, '/login');
		console.log(
			'PASS public landing, hero preview asset, section navigation, login CTA, seven screen widths and touch targets'
		);
	}
	await page.goto(`${base}/dashboard/patient`);
	await page.getByRole('heading', { name: 'Dokter Demo Visual', exact: true }).waitFor();
	await page.locator('summary').click();
	await page.getByRole('heading', { name: 'Obat sintetis demo' }).waitFor();
	await page.getByRole('button', { name: 'Resep', exact: true }).click();
	assert.equal(await page.locator('.patient-timeline li').count(), 1);
	await page.getByRole('button', { name: 'Semua', exact: true }).click();
	assert.equal(await page.locator('.patient-timeline li').count(), 2);
	await page.locator('summary').click();
	assert.deepEqual(await page.locator('.overview-metric strong').allTextContents(), [
		'1',
		'0',
		'1',
		'1'
	]);
	await page.getByRole('heading', { name: 'Tidak ada persetujuan tertunda' }).waitFor();
	assert.equal(await page.locator('.care-table tbody tr').count(), 1);
	await mkdir('.test-results', { recursive: true });
	for (const width of [1440, 1024, 768, 390, 375, 320]) {
		await page.setViewportSize({ width, height: 1000 });
		await page.evaluate(() => window.scrollTo(0, 0));
		await page.screenshot({ path: `.test-results/dashboard-patient-${width}.png`, fullPage: true });
		assert.ok(
			await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
			`overflow at ${width}: ${JSON.stringify(
				await page.evaluate(() =>
					[...document.querySelectorAll('.dashboard-workspace *')]
						.filter(
							(element) =>
								element.getBoundingClientRect().right > innerWidth &&
								getComputedStyle(element).position !== 'absolute'
						)
						.slice(0, 8)
						.map((element) => ({
							tag: element.tagName,
							class: element.className,
							right: element.getBoundingClientRect().right
						}))
				)
			)}`
		);
	}
	console.log(
		'PASS fixture data, medication details, timeline filters, navigation, six screen widths'
	);
	await page.setViewportSize({ width: 812, height: 375 });
	assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
	await page.screenshot({ path: '.test-results/dashboard-landscape.png', fullPage: true });
	await page.setViewportSize({ width: 375, height: 812 });
	await page.emulateMedia({ reducedMotion: 'reduce' });
	assert.ok(
		await page
			.locator('.dashboard-workspace button:visible, .panel-link:visible')
			.evaluateAll((elements) =>
				elements.every((element) => element.getBoundingClientRect().height >= 44)
			)
	);
	assert.equal(
		await page
			.getByRole('button', { name: 'Perbarui data' })
			.evaluate((element) => getComputedStyle(element).transitionDuration),
		'0s'
	);
	await page.evaluate(() => {
		const text = [...document.querySelectorAll('.dashboard-workspace *')].filter((element) =>
			[...element.childNodes].some(
				(child) => child.nodeType === Node.TEXT_NODE && child.textContent.trim()
			)
		);
		const sizes = text.map((element) => parseFloat(getComputedStyle(element).fontSize));
		text.forEach((element, index) => {
			element.style.fontSize = `${sizes[index] * 2}px`;
		});
	});
	assert.ok(
		await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
		'200% text fits small phone'
	);
	await page.screenshot({ path: '.test-results/dashboard-text-200.png', fullPage: true });
	await page.setViewportSize({ width: 320, height: 1000 });
	await page.reload();
	await page.getByRole('heading', { name: 'Dokter Demo Visual', exact: true }).waitFor();
	assert.equal(await page.getByRole('navigation', { name: 'Menu utama' }).isVisible(), false);
	const menuToggle = page.getByRole('button', { name: 'Menu', exact: true });
	await menuToggle.focus();
	await menuToggle.press('Enter');
	await page
		.getByRole('navigation', { name: 'Menu utama' })
		.getByRole('link', { name: 'Perawatan', exact: true })
		.focus();
	await page.keyboard.press('Escape');
	assert.equal(await menuToggle.getAttribute('aria-expanded'), 'false');
	assert.equal(await menuToggle.evaluate((element) => element === document.activeElement), true);
	await menuToggle.press('Enter');
	assert.equal(
		await page.getByRole('button', { name: 'Tutup menu' }).getAttribute('aria-expanded'),
		'true'
	);
	await page.screenshot({ path: '.test-results/dashboard-menu-320.png', fullPage: true });
	await page.locator('nav').getByRole('link', { name: 'Resep & apotek' }).click();
	await page.waitForURL('**?view=prescriptions');
	assert.equal(await page.getByRole('navigation', { name: 'Menu utama' }).isVisible(), false);
	assert.equal(new URL(page.url()).searchParams.get('view'), 'prescriptions');
	mode = 'pending';
	await page.goto(`${base}/dashboard/patient`);
	await page.locator('.pending-list a').nth(2).waitFor();
	assert.deepEqual(await page.locator('.overview-metric strong').allTextContents(), [
		'0',
		'3',
		'0',
		'0'
	]);
	assert.equal(await page.locator('.pending-list a').count(), 3);
	await page.locator('.overview-metric').nth(1).click();
	assert.equal(new URL(page.url()).hash, '#perlu-tindakan');
	await page.screenshot({ path: '.test-results/dashboard-pending-320.png', fullPage: true });
	await page.locator('.pending-list a').first().click();
	await page.waitForURL('**?view=episodes');
	console.log('PASS pending invitation/lab/prescription counts and action navigation');
	mode = 'empty';
	await page.goto(`${base}/dashboard/patient`);
	await page.getByRole('button', { name: 'Perbarui data' }).click();
	await page.getByRole('heading', { name: 'Belum ada resep', exact: true }).waitFor();
	await page.getByRole('heading', { name: 'Perjalanan Anda dimulai di sini.' }).waitFor();
	assert.deepEqual(await page.locator('.overview-metric strong').allTextContents(), [
		'0',
		'0',
		'0',
		'0'
	]);
	await page.screenshot({ path: '.test-results/dashboard-empty-320.png', fullPage: true });
	console.log('PASS empty data states');
	mode = 'error';
	await page.getByRole('button', { name: 'Perbarui data' }).click();
	await page.getByRole('button', { name: 'Coba lagi' }).first().waitFor();
	assert.equal(await page.getByRole('alert').count(), 4);
	assert.deepEqual(await page.locator('.overview-metric strong').allTextContents(), [
		'—',
		'—',
		'—',
		'—'
	]);
	await page.getByRole('heading', { name: 'Ringkasan belum lengkap' }).waitFor();
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
