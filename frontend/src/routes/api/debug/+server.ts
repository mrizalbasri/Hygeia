import type { RequestHandler } from './$types';

export const GET: RequestHandler = async () => {
	try {
		const res = await fetch(
			'https://env-hygeia-jp.bravecliff-18f241ae.japaneast.azurecontainerapps.io/api/v1/health'
		);
		const text = await res.text();
		return new Response(JSON.stringify({ ok: true, status: res.status, body: text }), {
			headers: { 'Content-Type': 'application/json' }
		});
	} catch (err: any) {
		return new Response(
			JSON.stringify({
				ok: false,
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
