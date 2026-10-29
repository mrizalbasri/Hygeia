import { afterEach, describe, expect, it, vi } from 'vitest';
import { pendingWork, readWorkSummary, type WorkSummary } from './patient-dashboard.js';

const summary: WorkSummary = {
	active_episodes: 31,
	labs_in_progress: 22,
	prescriptions_ready: 8,
	pending: { episodes: 25, lab_requests: 21, prescriptions: 3, consents: 7 }
};
afterEach(() => vi.unstubAllGlobals());

describe('patient work summary', () => {
	it('includes all four action categories and preserves totals beyond a page', () => {
		const cards = pendingWork(summary);
		expect(cards.map((card) => card.count)).toEqual([25, 21, 3, 7]);
		expect(cards.map((card) => card.view)).toEqual([
			'episodes',
			'labs',
			'prescriptions',
			'consents'
		]);
		expect(cards.reduce((total, card) => total + card.count, 0)).toBe(56);
	});
	it('does not invent actions when every count is zero', () => {
		expect(
			pendingWork({
				...summary,
				pending: { episodes: 0, lab_requests: 0, prescriptions: 0, consents: 0 }
			})
		).toEqual([]);
	});
	it('reads the complete backend summary', async () => {
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue(Response.json({ summary })));
		expect(await readWorkSummary()).toEqual({ data: summary, loading: false, error: '' });
	});
	it.each([{}, { ...summary, active_episodes: -1 }, { ...summary, pending: {} }])(
		'rejects malformed counters rather than displaying zero',
		async (invalid) => {
			vi.stubGlobal('fetch', vi.fn().mockResolvedValue(Response.json({ summary: invalid })));
			const result = await readWorkSummary();
			expect(result.data).toBeNull();
			expect(result.error).not.toBe('');
		}
	);
	it('keeps an unavailable summary distinct from an empty workload', async () => {
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('', { status: 503 })));
		const result = await readWorkSummary();
		expect(result.data).toBeNull();
		expect(result.loading).toBe(false);
		expect(result.error).not.toBe('');
	});
});
