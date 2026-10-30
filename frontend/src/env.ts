import { defineEnvVars } from '@sveltejs/kit/env';

export const variables = defineEnvVars({
	AUTH_PROVIDER: {
		description: 'Clerk for deployment, local only for development fixtures.',
		schema: (value: string | undefined) => {
			if (!value || value === 'local') return 'local' as const;
			if (value === 'clerk') return 'clerk' as const;
			throw new Error('AUTH_PROVIDER must be local or clerk');
		}
	},
	PUBLIC_CLERK_PUBLISHABLE_KEY: {
		public: true,
		description: 'Public Clerk instance key; never use the Clerk secret key here.',
		schema: (value: string | undefined) => {
			if (value && !/^pk_(test|live)_[A-Za-z0-9+/=]+$/.test(value))
				throw new Error('Invalid Clerk publishable key');
			return value || '';
		}
	},
	API_BASE_URL: {
		description: 'Private Go API origin for SSR identity checks and the same-origin API proxy.',
		schema: (value: string | undefined) => {
			let url: URL;
			try {
				url = new URL(value || 'http://127.0.0.1:8080');
			} catch {
				throw new Error('API_BASE_URL must be a valid URL (e.g. https://api.example.com)');
			}
			if (
				!['http:', 'https:'].includes(url.protocol) ||
				url.username ||
				url.password ||
				url.pathname !== '/' ||
				url.search ||
				url.hash
			) {
				throw new Error('API_BASE_URL must be an HTTP(S) origin without credentials or a path');
			}
			return url.origin;
		}
	}
});
