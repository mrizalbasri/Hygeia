<script lang="ts">
	import './layout.css';
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { loadClerk } from '#lib/clerk.js';
	import type { LayoutProps } from './$types';

	let { children, data }: LayoutProps = $props();
	onMount(() => {
		if (data.authProvider !== 'clerk' || !data.clerkPublishableKey) return;
		if (
			!window.location.pathname.startsWith('/dashboard') &&
			!window.location.pathname.startsWith('/account')
		)
			return;
		let disposed = false;
		let unsubscribe: (() => void) | undefined;
		void loadClerk(data.clerkPublishableKey)
			.then((client) => {
				if (disposed) return;
				let signedIn = client.isSignedIn;
				unsubscribe = client.addListener(({ session }) => {
					if (signedIn && !session) window.location.assign(resolve('/login'));
					signedIn = !!session;
				});
			})
			.catch(() => {
				/* Auth panels and API failures provide recovery feedback. */
			});
		return () => {
			disposed = true;
			unsubscribe?.();
		};
	});
</script>

<svelte:head>
	<link
		rel="icon"
		href="/favicon.ico"
		type="image/x-icon"
		sizes="16x16 24x24 32x32 48x48 64x64 128x128 256x256"
	/>
	<link rel="preconnect" href="https://fonts.googleapis.com" />
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
	<link
		href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700&display=swap"
		rel="stylesheet"
	/>
</svelte:head>

{@render children()}
