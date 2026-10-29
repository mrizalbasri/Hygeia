<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { loadClerk } from './clerk.js';
	import { authAppearance } from './clerk-appearance.js';
	import type { Clerk } from '@clerk/clerk-js';
	let {
		publishableKey,
		mode = 'login'
	}: { publishableKey: string; mode?: 'login' | 'register' | 'profile' } = $props();
	let node: HTMLDivElement;
	let error = $state('');
	let busy = $state(true);
	let retry = $state(0);
	let ready = $state(false);
	onMount(() => {
		ready = true;
	});
	async function proceed(client: Clerk) {
		const token = await client.session?.getToken();
		if (!token) return;
		const response = await fetch('/api/v1/auth/status', {
			headers: { Authorization: `Bearer ${token}` },
			cache: 'no-store'
		});
		const body = await response.json();
		if (!response.ok) throw new Error(body.error?.message || 'Akun belum dapat dihubungkan.');
		if (body.requires_mfa) window.location.assign(resolve('/account/security'));
		else window.location.assign(resolve('/dashboard/[role]', { role: body.user.role }));
	}
	$effect(() => {
		if (!ready) return;
		const currentMode = mode,
			key = publishableKey,
			attempt = retry;
		void attempt;
		let disposed = false,
			client: Clerk | undefined;
		busy = true;
		error = '';
		void loadClerk(key)
			.then(async (loaded) => {
				if (disposed) return;
				client = loaded;
				if (currentMode === 'profile') {
					if (!loaded.isSignedIn) {
						window.location.assign(resolve('/login'));
						return;
					}
					loaded.mountUserProfile(node, { routing: 'hash' });
				} else if (loaded.isSignedIn) await proceed(loaded);
				else if (currentMode === 'register')
					loaded.mountSignUp(node, {
						appearance: authAppearance,
						routing: 'hash',
						signInUrl: resolve('/login'),
						forceRedirectUrl: resolve('/auth/callback')
					});
				else
					loaded.mountSignIn(node, {
						appearance: authAppearance,
						routing: 'hash',
						signUpUrl: resolve('/register'),
						forceRedirectUrl: resolve('/auth/callback')
					});
			})
			.catch((cause) => {
				if (!disposed) error = cause instanceof Error ? cause.message : 'Autentikasi gagal dimuat.';
			})
			.finally(() => {
				if (!disposed) busy = false;
			});
		return () => {
			disposed = true;
			if (currentMode === 'profile') client?.unmountUserProfile(node);
			else if (currentMode === 'register') client?.unmountSignUp(node);
			else client?.unmountSignIn(node);
		};
	});
</script>

{#if busy}<p role="status">Memuat keamanan akun…</p>{/if}
{#if error}<p role="alert">{error}</p>
	<button class="button secondary" onclick={() => retry++}>Coba lagi</button><button
		class="button secondary"
		onclick={async () => {
			const clerk = await loadClerk(publishableKey);
			await clerk.signOut({ redirectUrl: resolve('/login') });
		}}>Keluar dan masuk ulang</button
	>{/if}
<div class="clerk-panel"><div bind:this={node}></div></div>

<style>
	.clerk-panel {
		width: 100%;
		min-width: 0;
	}
	.clerk-panel :global(.hygeia-auth-title) {
		font-size: 26px;
		font-weight: 700;
		line-height: 1.3;
		letter-spacing: -0.03em;
		color: #0a1f44;
		overflow-wrap: anywhere;
	}
	.clerk-panel :global(.hygeia-auth-subtitle) {
		color: #526176;
		font-size: 14px;
		line-height: 1.75;
		text-align: left;
	}
	.clerk-panel :global(.hygeia-auth-input) {
		min-height: 48px;
		padding: 12px 14px;
		margin: 0;
		border: 1px solid #b9cce3;
		border-radius: 12px;
		box-shadow: inset 0 0 0 1px #d7e5f5;
		background: #fff;
		color: #0a1f44;
		font-size: 16px;
	}
	.clerk-panel :global(.hygeia-auth-input:focus) {
		border-color: #0052cc;
		outline: 3px solid #e8f3ff;
		outline-offset: 1px;
	}
	.clerk-panel :global(.hygeia-auth-submit) {
		min-height: 48px;
		border: 0;
		border-radius: 12px;
		background: #0052cc;
		background-image: none;
		box-shadow: none;
		color: #fff;
		font-size: 14px;
		font-weight: 600;
		text-transform: none;
	}
	.clerk-panel :global(.hygeia-auth-submit:hover) {
		background: #0041a8;
	}
	.clerk-panel :global(.hygeia-auth-submit::after) {
		background-image: none;
	}
	.clerk-panel :global(.hygeia-auth-social) {
		min-height: 48px;
		border: 1px solid #d7e5f5;
		border-radius: 12px;
		background: #fff;
		box-shadow: inset 0 0 0 1px #d7e5f5;
		color: #0a1f44;
	}
	.clerk-panel :global(.hygeia-auth-social:hover) {
		background: #f8fbff;
	}
	.clerk-panel :global(.hygeia-auth-submit:focus-visible),
	.clerk-panel :global(.hygeia-auth-social:focus-visible) {
		outline: 3px solid #0052cc;
		outline-offset: 3px;
	}
</style>
