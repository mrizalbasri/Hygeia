import { redirect } from '@sveltejs/kit';
import { requireUser } from '#lib/server/identity.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies, fetch, params, setHeaders }) => {
	setHeaders({ 'cache-control': 'no-store' });
	const user = await requireUser(cookies, fetch);
	if (params.role !== user.role) redirect(303, `/dashboard/${user.role}`);
	return { user };
};
