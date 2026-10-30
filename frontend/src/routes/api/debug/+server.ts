import type { RequestHandler } from './$types';
import { API_BASE_URL, AUTH_PROVIDER } from '$app/env/private';

export const GET: RequestHandler = async () => {
	try {
		const targetUrl = `${API_BASE_URL}/api/v1/health`;
		const res = await fetch(targetUrl);
		const text = await res.text();
		return new Response(
			JSON.stringify({
				ok: true,
				API_BASE_URL,
				AUTH_PROVIDER,
				targetUrl,
				status: res.status,
				body: text
			}),
			{
				headers: { 'Content-Type': 'application/json' }
			}
		);
	} catch (err: any) {
		return new Response(
			JSON.stringify({
				ok: false,
				API_BASE_URL,
				AUTH_PROVIDER,
				error: err?.message,
				cause: err?.cause?.message || String(err?.cause),
				code: err?.cause?.code
			}),
			{
				status: 500,
				headers: { 'Content-Type': 'application/json' }
			}
		);
	}
};
