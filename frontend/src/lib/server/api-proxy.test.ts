import { describe, expect, it } from 'vitest';
import { proxyApi } from './api-proxy.js';

describe('deployment API proxy', () => {
	it('preserves authentication and CSRF while keeping the upstream fixed', async () => {
		const request = new Request('https://app.example.test/api/v1/consents?offset=2', {
			method: 'POST',
			headers: {
				cookie: '__session=clerk-token; hygeia_csrf=csrf-cookie; analytics=private',
				origin: 'https://app.example.test',
				'x-csrf-token': 'csrf-token',
				'x-forwarded-for': 'spoofed-ip',
				host: 'attacker.example.test',
				'content-type': 'application/json'
			},
			body: '{"action":"approve"}'
		});
		const upstream: typeof fetch = async (input, init) => {
			const headers = new Headers(init?.headers);
			expect(String(input)).toBe('https://api.example.test/api/v1/consents?offset=2');
			expect(headers.get('host')).toBeNull();
			expect(headers.get('x-forwarded-for')).toBeNull();
			expect(headers.get('cookie')).toBe('__session=clerk-token; hygeia_csrf=csrf-cookie');
			expect(headers.get('origin')).toBe('https://app.example.test');
			expect(headers.get('x-csrf-token')).toBe('csrf-token');
			expect(await new Response(init?.body).text()).toBe('{"action":"approve"}');
			expect(init?.redirect).toBe('manual');
			return Response.json({ ok: true });
		};
		const response = await proxyApi(
			request,
			new URL(request.url),
			'https://api.example.test',
			upstream
		);
		expect(response.status).toBe(200);
		expect(response.headers.get('cache-control')).toBe('no-store');
	});

	it('preserves separate cookies and streams SSE without waiting for completion', async () => {
		let controller: ReadableStreamDefaultController<Uint8Array>;
		const body = new ReadableStream<Uint8Array>({ start: (value) => (controller = value) });
		const headers = new Headers({ 'content-type': 'text/event-stream' });
		headers.append('set-cookie', 'hygeia_csrf=first; Path=/; HttpOnly');
		headers.append('set-cookie', 'hygeia_session=second; Path=/; HttpOnly');
		const upstream: typeof fetch = async () => new Response(body, { headers });
		const request = new Request('https://app.example.test/api/v1/notifications/stream');
		const response = await proxyApi(
			request,
			new URL(request.url),
			'https://api.example.test',
			upstream
		);
		expect(response.headers.getSetCookie()).toHaveLength(2);
		controller!.enqueue(new TextEncoder().encode('event: notification\ndata: {}\n\n'));
		const reader = response.body!.getReader();
		const first = await reader.read();
		expect(new TextDecoder().decode(first.value)).toContain('event: notification');
		await reader.cancel();
	});

	it('returns backend errors without caching or stale encoding headers', async () => {
		const upstream: typeof fetch = async () =>
			new Response('denied', {
				status: 403,
				headers: { 'content-encoding': 'gzip', 'content-length': '999' }
			});
		const request = new Request('https://app.example.test/api/v1/me');
		const response = await proxyApi(
			request,
			new URL(request.url),
			'https://api.example.test',
			upstream
		);
		expect(response.status).toBe(403);
		expect(response.headers.get('content-encoding')).toBeNull();
		expect(response.headers.get('content-length')).toBeNull();
		expect(await response.text()).toBe('denied');
	});

	it('hides upstream connection errors', async () => {
		const upstream: typeof fetch = async () => {
			throw new Error('private backend details');
		};
		const request = new Request('https://app.example.test/api/v1/me');
		const response = await proxyApi(
			request,
			new URL(request.url),
			'https://api.example.test',
			upstream
		);
		expect(response.status).toBe(502);
		expect(await response.text()).not.toContain('private backend details');
	});
});
