<script lang="ts">
	import { onMount } from 'svelte';
	import ClerkPanel from '#lib/ClerkPanel.svelte';
	import { authConfig } from '#lib/clerk.js';
	let key = $state(''),
		error = $state('');
	onMount(() => {
		void authConfig()
			.then((config) => {
				if (config.provider !== 'clerk' || !config.publishable_key)
					throw new Error('Konfigurasi Clerk belum tersedia.');
				key = config.publishable_key;
			})
			.catch((cause) => (error = cause.message));
	});
</script>

<svelte:head
	><title>Menghubungkan akun · Hygeia</title><meta name="robots" content="noindex" /></svelte:head
>
<main class="auth-callback">
	<h1>Menghubungkan akun Hygeia</h1>
	{#if error}<p role="alert">{error}</p>{:else if key}<ClerkPanel publishableKey={key} />{:else}<p
			role="status"
		>
			Memeriksa sesi…
		</p>{/if}
</main>

<style>
	.auth-callback {
		max-width: 560px;
		margin: 60px auto;
		padding: 24px;
	}
</style>
