import { api, errorText, type Entry, type Payload } from './api.js';

export type OverviewResource = { items: Entry[]; loading: boolean; error: string };
export const emptyOverview = (): OverviewResource => ({ items: [], loading: true, error: '' });

export async function readOverview(
	path: string,
	key: string,
	signal: AbortSignal
): Promise<OverviewResource> {
	try {
		const data = await api<Payload>(path, 'GET', undefined, signal);
		if (!Array.isArray(data[key])) throw new Error('Invalid list');
		return { items: data[key] as Entry[], loading: false, error: '' };
	} catch (error) {
		return { items: [], loading: false, error: errorText(error) };
	}
}
