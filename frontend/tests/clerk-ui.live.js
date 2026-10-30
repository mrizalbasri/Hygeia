import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { spawn } from 'node:child_process';
import { chromium } from 'playwright';
import { waitForServer } from './server-ready.js';

// Loads only the real development authentication UI. Does not submit credentials,
// send verification emails, create people, or impersonate an existing user.
if (process.env.HYGEIA_CLERK_LIVE !== '1') throw new Error('Set HYGEIA_CLERK_LIVE=1.');
const base = 'http://127.0.0.1:5188';
const vite = spawn(
	process.execPath,
	['node_modules/vite/bin/vite.js', 'dev', '--host', '127.0.0.1', '--port', '5188', '--strictPort'],
	{
		env: { ...process.env, AUTH_PROVIDER: 'clerk', API_BASE_URL: 'http://127.0.0.1:8080' },
		stdio: ['ignore', 'pipe', 'pipe'],
		windowsHide: true
	}
);
let output = '',
	browser;
vite.stdout.on('data', (data) => (output += data));
vite.stderr.on('data', (data) => (output += data));
try {
	await waitForServer(base, { output: () => output, isAlive: () => vite.exitCode === null });
	browser = await chromium.launch({ headless: true });
	const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
	const errors = [];
	page.on('pageerror', (error) => errors.push(error.message));
	await page.goto(`${base}/login`);
	await page.locator('.cl-signIn-root').waitFor({ timeout: 60000 });
	await page.locator('.cl-signIn-root input').first().waitFor();
	await mkdir('.test-results/clerk', { recursive: true });
	await page.screenshot({ path: '.test-results/clerk/login-desktop.png', fullPage: true });
	await page.setViewportSize({ width: 390, height: 844 });
	assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
	await page.screenshot({ path: '.test-results/clerk/login-mobile.png', fullPage: true });
	await page.goto(`${base}/register`);
	await page.locator('.cl-signUp-root').waitFor({ timeout: 60000 });
	assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
	await page.screenshot({ path: '.test-results/clerk/register-mobile.png', fullPage: true });
	assert.deepEqual(errors, []);
	console.log(
		'PASS real Clerk development sign-in/sign-up UI, desktop/mobile, no browser runtime errors'
	);
} finally {
	await browser?.close();
	vite.kill();
}
