import { AUTH_PROVIDER } from '$app/env/private';
import { PUBLIC_CLERK_PUBLISHABLE_KEY } from '$app/env/public';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = () => {
	const hasClerkKey = Boolean(PUBLIC_CLERK_PUBLISHABLE_KEY && PUBLIC_CLERK_PUBLISHABLE_KEY.trim());
	const provider = AUTH_PROVIDER === 'clerk' && hasClerkKey ? 'clerk' : 'local';
	return {
		authProvider: provider,
		clerkPublishableKey: provider === 'clerk' ? PUBLIC_CLERK_PUBLISHABLE_KEY : ''
	};
};
