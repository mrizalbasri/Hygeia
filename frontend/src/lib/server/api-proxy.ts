const hopHeaders = [
	'connection',
	'keep-alive',
	'proxy-authenticate',
	'proxy-authorization',
	'te',
	'trailer',
	'transfer-encoding',
	'upgrade'
];

function cleanHeaders(source: Headers) {
	const headers = new Headers(source);
	for (const name of (source.get('connection') || '').split(',')) {
		if (name.trim()) headers.delete(name.trim());
	}
	for (const name of hopHeaders) headers.delete(name);
	return headers;
}

// Fixed backend origin: the browser cannot choose the upstream destination.
export async function proxyApi(
	request: Request,
	url: URL,
	apiOrigin: string,
	upstreamFetch: typeof fetch = fetch
) {
	const target = new URL(apiOrigin);
	target.pathname = url.pathname;
	target.search = url.search;
	const headers = cleanHeaders(request.headers);
	for (const name of [...headers.keys()]) {
		if (name === 'host' || name === 'forwarded' || name.startsWith('x-forwarded-'))
			headers.delete(name);
	}
	const cookies = (headers.get('cookie') || '')
		.split(';')
		.filter((cookie) => /^(?:__session|hygeia_session|hygeia_csrf)=/.test(cookie.trim()));
	if (cookies.length) headers.set('cookie', cookies.join(';'));
	else headers.delete('cookie');
	headers.delete('content-length');
	headers.set('accept-encoding', 'identity');
	const stream = url.pathname === '/api/v1/notifications/stream';
	const init: RequestInit & { duplex?: 'half' } = {
		method: request.method,
		headers,
		redirect: 'manual',
		cache: 'no-store',
		signal: AbortSignal.any([request.signal, AbortSignal.timeout(stream ? 180000 : 15000)])
	};
	if (!['GET', 'HEAD'].includes(request.method)) {
		init.body = request.body;
		init.duplex = 'half';
	}
	try {
		const upstream = await upstreamFetch(target, init);
		const responseHeaders = cleanHeaders(upstream.headers);
		// fetch decompresses responses; stale transport headers corrupt the body.
		responseHeaders.delete('content-encoding');
		responseHeaders.delete('content-length');
		responseHeaders.delete('set-cookie');
		for (const cookie of upstream.headers.getSetCookie())
			responseHeaders.append('set-cookie', cookie);
		responseHeaders.set('cache-control', 'no-store');
		return new Response(upstream.body, { status: upstream.status, headers: responseHeaders });
	} catch {
		return Response.json(
			{ error: { code: 'unavailable', message: 'Layanan belum dapat dihubungi. Coba lagi.' } },
			{ status: 502, headers: { 'cache-control': 'no-store' } }
		);
	}
}
