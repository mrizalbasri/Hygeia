import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { goto } from '$app/navigation';
import { api, ApiError, errorText, type Entry, type ResultReceipt } from './api.js';

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/paths', () => ({ resolve: (path: string) => path }));

const fetchMock = vi.fn<typeof fetch>();
const json = (body: unknown, status = 200, headers = {}) =>
	new Response(JSON.stringify(body), { status, headers });

beforeEach(() => {
	vi.resetAllMocks();
	vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => vi.unstubAllGlobals());

describe('shared laboratory and pharmacy API client', () => {
	it.each([
		['/lab-requests', 'lab_requests'],
		['/prescriptions', 'prescriptions']
	])('loads a filtered, paginated %s queue', async (path, listKey) => {
		const item = { id: 'synthetic-item', patient_ref: 'HYG-TEST-001', status: 'sent' };
		fetchMock.mockResolvedValue(json({ [listKey]: [item] }));
		const query = `${path}?status=sent&limit=20&offset=20`;

		const result = await api<Record<string, Entry[]>>(query);

		expect(result[listKey]).toEqual([item]);
		expect(fetchMock).toHaveBeenCalledExactlyOnceWith(
			`/api/v1${query}`,
			expect.objectContaining({ method: 'GET', credentials: 'same-origin', cache: 'no-store' })
		);
	});

	it.each(['/lab-requests/lab-1/status', '/prescriptions/rx-1/status'])(
		'sends the CSRF token and status body to %s',
		async (path) => {
			fetchMock
				.mockResolvedValueOnce(json({ csrf_token: 'synthetic-csrf' }))
				.mockResolvedValueOnce(json({ status: 'accepted' }));

			await api(path, 'PATCH', { status: 'accepted' });

			expect(fetchMock).toHaveBeenNthCalledWith(
				1,
				'/api/v1/auth/csrf',
				expect.objectContaining({ method: 'GET', credentials: 'same-origin' })
			);
			expect(fetchMock).toHaveBeenNthCalledWith(
				2,
				`/api/v1${path}`,
				expect.objectContaining({
					method: 'PATCH',
					credentials: 'same-origin',
					headers: { 'X-CSRF-Token': 'synthetic-csrf', 'Content-Type': 'application/json' },
					body: JSON.stringify({ status: 'accepted' })
				})
			);
		}
	);

	it('submits a structured lab result and returns its receipt', async () => {
		const result = {
			value: '95',
			unit: 'mg/dL',
			reference_range: '70–99',
			flag: 'normal',
			note: 'Synthetic result'
		};
		const receipt = { id: 'receipt-1', submitted_at: '2026-10-29T01:00:00Z' };
		fetchMock
			.mockResolvedValueOnce(json({ csrf_token: 'synthetic-csrf' }))
			.mockResolvedValueOnce(json({ result: receipt }, 201));

		expect(
			await api<{ result: ResultReceipt }>('/lab-requests/lab-1/result', 'POST', result)
		).toEqual({ result: receipt });
		expect(fetchMock).toHaveBeenLastCalledWith(
			'/api/v1/lab-requests/lab-1/result',
			expect.objectContaining({ method: 'POST', body: JSON.stringify(result) })
		);
	});

	it('obtains a fresh CSRF token for each mutation', async () => {
		fetchMock
			.mockResolvedValueOnce(json({ csrf_token: 'first-csrf' }))
			.mockResolvedValueOnce(json({}))
			.mockResolvedValueOnce(json({ csrf_token: 'second-csrf' }))
			.mockResolvedValueOnce(json({}));

		await api('/lab-requests/lab-1/status', 'PATCH', { status: 'accepted' });
		await api('/lab-requests/lab-1/status', 'PATCH', { status: 'in_progress' });

		expect(fetchMock).toHaveBeenNthCalledWith(
			4,
			'/api/v1/lab-requests/lab-1/status',
			expect.objectContaining({
				headers: expect.objectContaining({ 'X-CSRF-Token': 'second-csrf' })
			})
		);
	});

	it('preserves HTTP 409 and the request ID for conflict recovery', async () => {
		fetchMock.mockResolvedValueOnce(json({ csrf_token: 'synthetic-csrf' })).mockResolvedValueOnce(
			json(
				{
					error: {
						message: 'Status has changed',
						code: 'status_conflict',
						request_id: 'request-409'
					}
				},
				409
			)
		);

		const failure = await api('/prescriptions/rx-1/status', 'PATCH', { status: 'accepted' }).catch(
			(error: unknown) => error
		);

		expect(failure).toBeInstanceOf(ApiError);
		expect(failure).toMatchObject({
			status: 409,
			code: 'status_conflict',
			requestId: 'request-409'
		});
		expect(errorText(failure)).toBe('Status has changed (ID: request-409)');
	});

	it('does not send a mutation when CSRF acquisition fails', async () => {
		fetchMock.mockResolvedValue(json({ error: { message: 'Forbidden' } }, 403));
		await expect(api('/lab-requests/lab-1/result', 'POST', { value: '95' })).rejects.toMatchObject({
			status: 403
		});
		expect(fetchMock).toHaveBeenCalledTimes(1);
	});

	it('redirects an expired session to login', async () => {
		fetchMock.mockResolvedValue(json({ error: { message: 'Session expired' } }, 401));
		await expect(api('/prescriptions')).rejects.toMatchObject({ status: 401 });
		expect(goto).toHaveBeenCalledWith('/login', { replace: true, refreshAll: true });
	});

	it('reports connectivity failures as an API error', async () => {
		fetchMock.mockRejectedValue(new TypeError('Failed to fetch'));
		await expect(api('/lab-requests')).rejects.toMatchObject({ status: 0 });
	});
});
