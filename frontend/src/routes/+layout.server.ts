import { AUTH_PROVIDER } from '$app/env/private';
import { PUBLIC_CLERK_PUBLISHABLE_KEY } from '$app/env/public';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = () => ({
	authProvider: AUTH_PROVIDER,
	clerkPublishableKey: AUTH_PROVIDER === 'clerk' ? PUBLIC_CLERK_PUBLISHABLE_KEY : ''
});
