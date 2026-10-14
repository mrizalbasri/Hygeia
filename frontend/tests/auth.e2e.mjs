import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { chromium } from 'playwright';

const password = process.env.DEMO_PASSWORD;
assert.ok(
	['development', 'test'].includes(process.env.APP_ENV),
	'Use a development/test environment'
);
assert.ok(password, 'DEMO_PASSWORD must be set for seeded development accounts');
const baseURL = process.env.TEST_WEB_URL || 'http://localhost:5173';
const artifacts = new URL('../.test-results/', import.meta.url);
await mkdir(artifacts, { recursive: true });
// Use an installed browser; override with PLAYWRIGHT_CHANNEL when needed.
const browser = await chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || 'msedge' });
const pageErrors = [];
try {
	const anonymous = await browser.newContext({ baseURL, viewport: { width: 1440, height: 1000 } });
	const anonPage = await anonymous.newPage();
	anonPage.on('pageerror', (error) => pageErrors.push(error.message));
	await anonPage.goto('/dashboard/admin');
	assert.equal(
		new URL(anonPage.url()).pathname,
		'/login',
		'anonymous dashboard access must redirect'
	);
	await anonPage.locator('button[type="submit"]:not([disabled])').waitFor();
	await anonPage.screenshot({
		path: new URL('login-desktop.png', artifacts).pathname.replace(/^\/([A-Za-z]:)/, '$1'),
		fullPage: true
	});
	await anonymous.close();

	const accounts = [
		['patient', 'aulia', 'Perjalanan Saya'],
		['doctor', 'maya', 'Ruang Perawatan'],
		['lab', 'lab', 'Permintaan Masuk'],
		['pharmacy', 'apotek', 'Antrean Resep'],
		['admin', 'admin', 'Kelola Platform']
	];
	for (const [role, account, heading] of accounts) {
		const context = await browser.newContext({ baseURL, viewport: { width: 1440, height: 1000 } });
		const page = await context.newPage();
		page.on('pageerror', (error) => pageErrors.push(error.message));
		await page.goto('/login');
		await page.getByLabel('Email', { exact: true }).fill(`${account}@demo.hygeia.test`);
		await page.getByLabel('Password', { exact: true }).fill(password);
		await page.getByRole('button', { name: 'Masuk', exact: true }).click();
		await page.waitForURL(`**/dashboard/${role}`);
		assert.equal(await page.getByRole('heading', { level: 1 }).textContent(), heading);
		const session = (await context.cookies()).find((cookie) => cookie.name === 'hygeia_session');
		assert.ok(session?.httpOnly, 'session cookie must be HttpOnly');
		await page.reload();
		assert.equal(new URL(page.url()).pathname, `/dashboard/${role}`, 'session must survive reload');
		await page.goto(`/dashboard/${role === 'admin' ? 'patient' : 'admin'}`);
		assert.equal(
			new URL(page.url()).pathname,
			`/dashboard/${role}`,
			'wrong-role dashboard must redirect'
		);
		if (role === 'patient') {
			await page.screenshot({
				path: new URL('patient-desktop.png', artifacts).pathname.replace(/^\/([A-Za-z]:)/, '$1'),
				fullPage: true
			});
			await page.setViewportSize({ width: 390, height: 844 });
			assert.equal(
				await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
				true,
				'mobile dashboard must not overflow'
			);
			await page.screenshot({
				path: new URL('patient-mobile.png', artifacts).pathname.replace(/^\/([A-Za-z]:)/, '$1'),
				fullPage: true
			});
		}
		await page.getByRole('button', { name: 'Keluar', exact: true }).click();
		await page.waitForURL('**/login');
		assert.equal(
			(await context.request.get('/api/v1/me')).status(),
			401,
			'logout must revoke session'
		);
		await page.goto(`/dashboard/${role}`);
		assert.equal(new URL(page.url()).pathname, '/login', 'logout must prevent re-entry');
		console.log(`PASS: ${role} login, reload, role redirect, and logout`);
		await context.close();
	}
	const mobile = await browser.newContext({ baseURL, viewport: { width: 390, height: 844 } });
	const mobilePage = await mobile.newPage();
	await mobilePage.goto('/login');
	assert.equal(
		await mobilePage.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
		true,
		'mobile login must not overflow'
	);
	await mobilePage.screenshot({
		path: new URL('login-mobile.png', artifacts).pathname.replace(/^\/([A-Za-z]:)/, '$1'),
		fullPage: true
	});
	await mobile.close();
	assert.deepEqual(pageErrors, [], 'browser runtime errors');
} finally {
	if (pageErrors.length) console.error('Browser errors:', pageErrors);
	await browser.close();
}
