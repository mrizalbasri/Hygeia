<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { authMutation, roles } from '#lib/auth.js';
	import ClinicalWorkspace from '#lib/workflows/ClinicalWorkspace.svelte';
	import { menus } from '#lib/workflows/api.js';
	import { page } from '$app/state';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	let role = $derived(roles[data.user.role]);
	let clinical = $derived(data.user.role === 'patient' || data.user.role === 'doctor');
	let menu = $derived(data.user.role === 'doctor' ? menus.doctor : menus.patient);
	let activeView = $derived(
		menu.some(([id]) => id === page.url.searchParams.get('view'))
			? page.url.searchParams.get('view')
			: menu[0][0]
	);
	let pending = $state(false);
	let ready = $state(false);
	let message = $state('');
	onMount(() => {
		ready = true;
	});

	async function logout() {
		if (pending) return;
		pending = true;
		message = '';
		try {
			const response = await authMutation('logout');
			if (!response.ok) throw new Error();
			await goto(resolve('/login'), { replace: true, refreshAll: true });
		} catch {
			message = 'Belum dapat keluar. Coba lagi.';
		} finally {
			pending = false;
		}
	}
</script>

<svelte:head><title>{role.title} · Hygeia</title></svelte:head>

<div class="workspace">
	<aside class="sidebar">
		<a class="brand" href={resolve('/dashboard/[role]', { role: data.user.role })}
			><img src="/logo-light.png" alt="" width="40" height="40" /><span>Hygeia</span></a
		>
		<p class="eyebrow muted workspace-label">RUANG {role.label.toUpperCase()}</p>
		<nav aria-label="Menu utama">
			{#if clinical}
				{#each menu as [id, label] (id)}<a
						class:nav-active={activeView === id}
						aria-current={activeView === id ? 'page' : undefined}
						href={resolve(`/dashboard/[role]?view=${id}`, { role: data.user.role })}>{label}</a
					>{/each}
			{:else}
				<a
					class="nav-active"
					href={resolve('/dashboard/[role]', { role: data.user.role })}
					aria-current="page"><span aria-hidden="true">⌂</span>Beranda</a
				>{#each role.items as item (item)}<button
						class="nav-upcoming"
						disabled
						title="Segera tersedia">{item}<span aria-hidden="true">·</span></button
					>{/each}{/if}
		</nav>
		<div class="sidebar-note">
			<span class="notice-dot" aria-hidden="true"></span>Demo · data sintetis
		</div>
	</aside>
	<div class="workspace-main">
		<header class="workspace-header">
			<span class="muted">{data.user.organization?.name || 'Hygeia'}</span>
			<div class="account-actions">
				<span class="role-chip">{role.label}</span><button
					class="button secondary"
					onclick={logout}
					disabled={pending || !ready}>{pending ? 'Keluar…' : 'Keluar'}</button
				>
			</div>
		</header>
		<main class="dashboard-content" class:patient-content={data.user.role === 'patient'}>
			{#if clinical}
				{#if message}<p class="error-banner" role="alert">{message}</p>{/if}
				<ClinicalWorkspace user={data.user} />
			{:else}
				<p class="eyebrow muted">SELAMAT DATANG, {data.user.full_name.toUpperCase()}</p>
				<h1>{role.title}</h1>
				<p class="muted dashboard-intro">{role.description}</p>
				{#if message}<p class="error-banner" role="alert">{message}</p>{/if}
				<section class="welcome-panel" aria-labelledby="welcome-title">
					<div class="panel-marker" aria-hidden="true">+</div>
					<div>
						<p class="eyebrow muted">RUANG ANDA SUDAH SIAP</p>
						<h2 id="welcome-title">Mulai dari satu perjalanan.</h2>
						<p class="muted">
							Fitur {role.items[0].toLowerCase()} akan tersedia pada tahap berikutnya. Aktivitas perawatan
							akan muncul di ruang ini.
						</p>
						<span class="upcoming-chip">Segera tersedia</span>
					</div>
				</section>
				<section class="identity-panel" aria-labelledby="identity-title">
					<div>
						<p class="eyebrow muted">AKUN AKTIF</p>
						<h2 id="identity-title">{data.user.full_name}</h2>
					</div>
					<dl>
						<div>
							<dt>Email</dt>
							<dd>{data.user.email}</dd>
						</div>
						<div>
							<dt>Peran</dt>
							<dd>{role.label}</dd>
						</div>
						<div>
							<dt>Organisasi</dt>
							<dd>{data.user.organization?.name || '—'}</dd>
						</div>
					</dl>
				</section>
			{/if}
			<p class="dashboard-footer">Hygeia · Perawatan terhubung, akses terkendali.</p>
		</main>
	</div>
</div>
