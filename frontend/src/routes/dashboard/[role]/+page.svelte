<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { authMutation, roles } from '#lib/auth.js';
	import ClinicalWorkspace from '#lib/workflows/ClinicalWorkspace.svelte';
	import LaboratoryWorkspace from '#lib/workflows/LaboratoryWorkspace.svelte';
	import PharmacyWorkspace from '#lib/workflows/PharmacyWorkspace.svelte';
	import AdminWorkspace from '#lib/workflows/AdminWorkspace.svelte';
	import WorkspaceIcon from '#lib/WorkspaceIcon.svelte';
	import '#lib/workspace.css';
	import { menus } from '#lib/workflows/api.js';
	import { provideNotifications } from '#lib/workflows/notification-state.svelte.js';
	import { page } from '$app/state';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	const notifications = provideNotifications();
	$effect(() => {
		const userID = data.user.id;
		void userID;
		return untrack(() => notifications.connect());
	});
	let role = $derived(roles[data.user.role]);
	let clinical = $derived(data.user.role === 'patient' || data.user.role === 'doctor');
	let isLab = $derived(data.user.role === 'lab');
	let isPharmacy = $derived(data.user.role === 'pharmacy');
	let isAdmin = $derived(data.user.role === 'admin');
	let hasWorkspace = $derived(clinical || isLab || isPharmacy || isAdmin);
	let menu = $derived(
		data.user.role === 'doctor'
			? menus.doctor
			: data.user.role === 'patient'
				? menus.patient
				: data.user.role === 'lab'
					? menus.lab
					: data.user.role === 'pharmacy'
						? menus.pharmacy
						: data.user.role === 'admin'
							? menus.admin
							: []
	);
	let activeView = $derived(
		menu.some(([id]) => id === page.url.searchParams.get('view'))
			? page.url.searchParams.get('view')
			: menu[0]?.[0] || 'overview'
	);
	let pending = $state(false);
	let ready = $state(false);
	let message = $state('');
	let mobileMenuOpen = $state(false);
	let menuToggle: HTMLButtonElement | undefined = $state();
	let initials = $derived(
		data.user.full_name
			.split(' ')
			.filter(Boolean)
			.slice(0, 2)
			.map((name) => name[0])
			.join('')
	);
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

<svelte:window
	onkeydown={(event) => {
		if (event.key === 'Escape' && mobileMenuOpen) {
			mobileMenuOpen = false;
			menuToggle?.focus();
		}
	}}
/>

<a class="workspace-skip-link" href="#workspace-main-content">Langsung ke isi dashboard</a>
<div class="workspace dashboard-workspace" class:mobile-menu-open={mobileMenuOpen}>
	<aside class="sidebar">
		<div class="sidebar-mobile-heading">
			<a class="brand" href={resolve('/dashboard/[role]', { role: data.user.role })}
				><img src="/logo-light.png" alt="" width="40" height="40" /><span>Hygeia</span></a
			>
			<button
				class="workspace-menu-toggle"
				bind:this={menuToggle}
				aria-expanded={mobileMenuOpen}
				aria-controls="workspace-navigation"
				onclick={() => (mobileMenuOpen = !mobileMenuOpen)}
				><WorkspaceIcon name={mobileMenuOpen ? 'close' : 'menu'} />{mobileMenuOpen
					? 'Tutup menu'
					: 'Menu'}</button
			>
		</div>
		<p class="workspace-label">Ruang {role.label}</p>
		<nav id="workspace-navigation" aria-label="Menu utama">
			{#if hasWorkspace}
				{#each menu as [id, label] (id)}<a
						class:nav-divider={id === 'notifications' || id === 'audit'}
						class:nav-active={activeView === id}
						aria-current={activeView === id ? 'page' : undefined}
						onclick={() => (mobileMenuOpen = false)}
						href={resolve(`/dashboard/[role]?view=${id}`, { role: data.user.role })}
						><WorkspaceIcon name={id === 'queue' && isPharmacy ? 'prescriptions' : id} /><span
							>{label}</span
						>{#if activeView === id}<span class="nav-current-dot" aria-hidden="true"></span>{/if}</a
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
		<div class="sidebar-bottom">
			<div class="sidebar-account">
				<span class="account-avatar" aria-hidden="true">{initials}</span>
				<div><strong>{data.user.full_name}</strong><span>{role.label}</span></div>
			</div>
		</div>
	</aside>
	<div class="workspace-main">
		<header class="workspace-header">
			<div class="workspace-breadcrumb">
				<span>{data.user.organization?.name || 'Ruang pribadi'}</span><span aria-hidden="true"
					>/</span
				><strong>{menu.find(([id]) => id === activeView)?.[1] || role.title}</strong>
			</div>
			<div class="account-actions">
				<a
					class="header-notifications"
					aria-label={notifications.unread === null
						? 'Buka notifikasi; jumlah belum dibaca belum tersedia'
						: `Buka notifikasi; ${notifications.unread} belum dibaca`}
					href={resolve('/dashboard/[role]?view=notifications', { role: data.user.role })}
					><WorkspaceIcon name="notifications" />
					{#if notifications.unread !== null && notifications.unread > 0}
						<span class="notification-badge" aria-hidden="true"
							>{notifications.unread > 99 ? '99+' : notifications.unread}</span
						>
					{/if}</a
				>
				<div class="header-account">
					<span class="account-avatar" aria-hidden="true">{initials}</span><span
						>{data.user.full_name}</span
					>
				</div>
				<button class="button secondary logout-button" onclick={logout} disabled={pending || !ready}
					><WorkspaceIcon name="logout" />{pending ? 'Keluar…' : 'Keluar'}</button
				>
			</div>
		</header>
		<main
			id="workspace-main-content"
			tabindex="-1"
			class="dashboard-content"
			class:patient-content={data.user.role === 'patient'}
		>
			{#if notifications.failed}
				<div class="notification-connection" role="status">
					<p>{notifications.connection}</p>
					<button type="button" class="button secondary" onclick={() => notifications.connect()}
						>Coba lagi notifikasi</button
					>
				</div>
			{/if}
			{#key data.user.id}
				{#if clinical}
					{#if message}<p class="error-banner" role="alert">{message}</p>{/if}
					<ClinicalWorkspace user={data.user} />
				{:else if isLab}
					{#if message}<p class="error-banner" role="alert">{message}</p>{/if}
					<LaboratoryWorkspace user={data.user} />
				{:else if isPharmacy}
					{#if message}<p class="error-banner" role="alert">{message}</p>{/if}
					<PharmacyWorkspace user={data.user} />
				{:else if isAdmin}
					{#if message}<p class="error-banner" role="alert">{message}</p>{/if}
					<AdminWorkspace user={data.user} />
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
			{/key}
			<p class="dashboard-footer">Hygeia · Perawatan terhubung, akses terkendali.</p>
		</main>
	</div>
</div>
