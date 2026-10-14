import { defineEnvVars } from '@sveltejs/kit/env';

export const variables = defineEnvVars({
	API_BASE_URL: {
		description: 'Private origin of the Go API for server-side identity checks.',
		schema: (value: string | undefined) => {
			const url = new URL(value || 'http://127.0.0.1:8080');
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
