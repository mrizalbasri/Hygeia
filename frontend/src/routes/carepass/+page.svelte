<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import type { User } from '#lib/auth.js';
	import { api, errorText } from '#lib/workflows/api.js';
	import Sharing from '#lib/workflows/Sharing.svelte';
	import ClinicalDetail from '#lib/workflows/ClinicalDetail.svelte';
	import '#lib/workflows/workflow.css';
	let token = $state(''),
		user = $state<User | null>(null),
		error = $state(''),
		loading = $state(true),
		detail = $state<{ kind: string; id: string; expires?: string } | null>(null);
	onMount(() => {
		token = new URLSearchParams(location.hash.slice(1)).get('token') || '';
		history.replaceState(history.state, '', location.pathname + location.search);
		void api<{ user: User }>('/me')
			.then((data) => (user = data.user))
			.catch((reason) => (error = errorText(reason)))
			.finally(() => (loading = false));
		return () => {
			token = '';
			detail = null;
		};
	});
</script>

<svelte:head
	><title>Tebus CarePass · Hygeia</title><meta name="referrer" content="no-referrer" /></svelte:head
>
<main class="carepass-page">
	<a class="brand" href={resolve('/')}
		><img src="/logo-light.png" alt="" width="40" height="40" />Hygeia</a
	>
	<h1>CarePass pasien</h1>
	{#if loading}<p role="status">Memeriksa sesi Anda…</p>{:else if error}<p
			class="wf-error"
			role="alert"
		>
			{error}
		</p>
		<a href={resolve('/login')}>Masuk terlebih dahulu, lalu buka ulang tautan CarePass.</a
		>{:else if user?.role === 'doctor'}<fieldset class="wf-content">
			<Sharing
				doctor={true}
				run={async () => false}
				busy={false}
				revision={0}
				initialToken={token}
				consumed={() => (token = '')}
				show={(kind, id, expires) => (detail = { kind, id, expires })}
			/>
		</fieldset>
		<a class="button secondary" href={resolve('/dashboard/[role]', { role: 'doctor' })}
			>Kembali ke ruang dokter</a
		>{:else}<p>CarePass hanya dapat ditebus oleh dokter. Masuk dengan akun dokter yang dituju.</p>
		<a class="button secondary" href={resolve('/login')}>Ke halaman masuk</a
		>{/if}{#if detail}<ClinicalDetail
			kind={detail.kind}
			id={detail.id}
			expiresAt={detail.expires}
			close={() => (detail = null)}
		/>{/if}
</main>
