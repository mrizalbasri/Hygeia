import { API_BASE_URL } from '$app/env/private';
import { error, redirect } from '@sveltejs/kit';
import { roles, type User } from '#lib/auth.js';
import type { Cookies } from '@sveltejs/kit';

export async function requireUser(cookies: Cookies, serverFetch: typeof fetch): Promise<User> {
	const session = cookies.get('hygeia_session');
	if (!session) redirect(303, '/login');
	let response: Response;
	try {
		response = await serverFetch(`${API_BASE_URL}/api/v1/me`, {
			headers: { Cookie: `hygeia_session=${encodeURIComponent(session)}` },
			cache: 'no-store'
		});
	} catch {
		error(503, 'Layanan sedang tidak tersedia. Coba lagi sebentar.');
	}
	if (response.status === 401) redirect(303, '/login');
	if (!response.ok) error(503, 'Layanan sedang tidak tersedia.');
	const { user } = (await response.json()) as { user: User };
	if (!user || !Object.hasOwn(roles, user.role)) error(403, 'Peran akun tidak dikenali.');
	return user;
}
