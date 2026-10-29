<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import ClerkPanel from '#lib/ClerkPanel.svelte';
	import { authConfig, loadClerk } from '#lib/clerk.js';
	let key = $state(''),
		error = $state(''),
		checking = $state(false);
	onMount(() => {
		void authConfig()
			.then((config) => {
				if (config.provider !== 'clerk' || !config.publishable_key)
					throw new Error('Pengaturan Clerk belum diaktifkan.');
				key = config.publishable_key;
			})
			.catch((cause) => (error = cause.message));
	});
	async function signOut() {
		if (!key) return;
		const clerk = await loadClerk(key);
		await clerk.signOut({ redirectUrl: resolve('/login') });
	}
	async function recheck() {
		checking = true;
		error = '';
		try {
			const clerk = await loadClerk(key);
			const token = await clerk.session?.getToken({ skipCache: true });
			const response = await fetch('/api/v1/auth/status', {
				headers: { Authorization: `Bearer ${token || ''}` },
				cache: 'no-store'
			});
			const body = await response.json();
			if (!response.ok) throw new Error(body.error?.message || 'Tidak dapat memeriksa akun.');
			if (body.requires_mfa)
				throw new Error(
					'Aktifkan aplikasi autentikator, lalu keluar dan masuk kembali dengan kode MFA.'
				);
			window.location.assign(resolve('/dashboard/[role]', { role: body.user.role }));
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Pemeriksaan gagal.';
		} finally {
			checking = false;
		}
	}
</script>

<svelte:head
	><title>Keamanan akun · Hygeia</title><meta name="robots" content="noindex" /></svelte:head
>
<main class="account-security">
	<h1>Keamanan akun</h1>
	<p>
		Kelola password, alamat email, perangkat aktif, dan autentikasi dua faktor. Admin dan staf wajib
		mengaktifkan MFA dan masuk dengan faktor kedua sebelum mengakses layanan.
	</p>
	{#if error}<p role="alert">{error}</p>{/if}
	<div class="account-actions">
		<button class="button primary" onclick={recheck} disabled={!key || checking}
			>Lanjut ke ruang saya</button
		><button class="button secondary" onclick={signOut} disabled={!key}>Keluar dari akun</button>
	</div>
	{#if key}<ClerkPanel publishableKey={key} mode="profile" />{/if}
</main>

<style>
	.account-security {
		max-width: 1000px;
		margin: 40px auto;
		padding: 24px;
	}
	.account-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 12px;
		margin: 24px 0;
	}
</style>
