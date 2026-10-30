import { API_BASE_URL } from '$app/env/private';
import { error, redirect } from '@sveltejs/kit';
import { roles, type User } from '#lib/auth.js';
import type { Cookies } from '@sveltejs/kit';
import { AUTH_PROVIDER } from '$app/env/private';
import { PUBLIC_CLERK_PUBLISHABLE_KEY } from '$app/env/public';

export async function requireUser(cookies: Cookies, serverFetch: typeof fetch): Promise<User> {
	const clerk =
		AUTH_PROVIDER === 'clerk' &&
		Boolean(PUBLIC_CLERK_PUBLISHABLE_KEY && PUBLIC_CLERK_PUBLISHABLE_KEY.trim());
	const session = cookies.get(clerk ? '__session' : 'hygeia_session');
	if (!session) redirect(303, '/login');
	let response: Response;
	try {
		response = await serverFetch(`${API_BASE_URL}/api/v1/me`, {
			headers: clerk
				? { Authorization: `Bearer ${session}` }
				: { Cookie: `hygeia_session=${encodeURIComponent(session)}` },
			cache: 'no-store'
		});
	} catch {
		error(503, 'Layanan sedang tidak tersedia. Coba lagi sebentar.');
	}
	if (response.status === 401) redirect(303, '/login');
	if (clerk && response.status === 403) {
		const body = await response.json();
		if (body.error?.code === 'mfa_required') redirect(303, '/account/security');
		error(403, 'Akses akun tidak diizinkan.');
	}
	if (clerk && response.status === 409) redirect(303, '/auth/callback');
	if (!response.ok) error(503, 'Layanan sedang tidak tersedia.');
	const { user } = (await response.json()) as { user: User };
	if (!user || !Object.hasOwn(roles, user.role)) error(403, 'Peran akun tidak dikenali.');
	return user;
}
