import { defineEnvVars } from '@sveltejs/kit/env';

export const variables = defineEnvVars({
	AUTH_PROVIDER: {
		description: 'Authentication provider: local (default) or clerk.',
		schema: (value: string | undefined) => {
			const clean = value?.trim().toLowerCase();
			if (clean === 'clerk') return 'clerk' as const;
			return 'local' as const;
		}
	},
	PUBLIC_CLERK_PUBLISHABLE_KEY: {
		public: true,
		description: 'Public Clerk instance key; never use the Clerk secret key here.',
		schema: (value: string | undefined) => {
			const clean = value?.trim() || '';
			if (clean && !/^pk_(test|live)_[A-Za-z0-9+/=]+$/.test(clean)) {
				return '';
			}
			return clean;
		}
	},
	API_BASE_URL: {
		description: 'Private Go API origin for SSR identity checks and the same-origin API proxy.',
		schema: (value: string | undefined) => {
			let raw = (value || '').trim();
			if (
				(raw.startsWith('"') && raw.endsWith('"')) ||
				(raw.startsWith("'") && raw.endsWith("'"))
			) {
				raw = raw.slice(1, -1).trim();
			}
			if (!raw) {
				return 'https://env-hygeia-jp.bravecliff-18f241ae.japaneast.azurecontainerapps.io';
			}
			if (!raw.startsWith('http://') && !raw.startsWith('https://')) {
				raw = `https://${raw}`;
			}
			try {
				const url = new URL(raw);
				return url.origin;
			} catch {
				return 'https://env-hygeia-jp.bravecliff-18f241ae.japaneast.azurecontainerapps.io';
			}
		}
	}
});
