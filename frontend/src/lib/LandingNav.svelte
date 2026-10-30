<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import HealthIcon from '#lib/HealthIcon.svelte';
	let { onSelectRole }: { onSelectRole?: (index: number) => void } = $props();
	let activeMenu = $state<'features' | 'roles' | null>(null);
	let mobileOpen = $state(false);
	let header: HTMLElement;
	let lastTrigger: HTMLButtonElement | undefined;
	let closeTimer: ReturnType<typeof setTimeout> | undefined;

	function cancelClose() {
		clearTimeout(closeTimer);
		closeTimer = undefined;
	}

	function openOnHover(menu: 'features' | 'roles', event: PointerEvent) {
		if (event.pointerType !== 'mouse' || !window.matchMedia('(hover: hover)').matches) return;
		cancelClose();
		lastTrigger = event.currentTarget as HTMLButtonElement;
		activeMenu = menu;
	}

	function closeOnLeave(event: PointerEvent) {
		if (event.pointerType !== 'mouse') return;
		cancelClose();
		closeTimer = setTimeout(close, 180);
	}

	onMount(() => {
		header.addEventListener('pointerenter', cancelClose);
		header.addEventListener('pointerleave', closeOnLeave);
		return () => {
			cancelClose();
			header.removeEventListener('pointerenter', cancelClose);
			header.removeEventListener('pointerleave', closeOnLeave);
		};
	});

	const features = [
		{
			title: 'Perawatan',
			text: 'Hubungan dengan dokter dan perjalanan perawatan.',
			icon: 'care' as const,
			href: '#fitur'
		},
		{
			title: 'Pemeriksaan',
			text: 'Permintaan, proses, dan hasil laboratorium.',
			icon: 'lab' as const,
			href: '#fitur'
		},
		{
			title: 'Resep & apotek',
			text: 'Detail resep dan status penyiapan obat.',
			icon: 'medicine' as const,
			href: '#fitur'
		},
		{
			title: 'Persetujuan akses',
			text: 'Atur izin berbagi data perawatan Anda.',
			icon: 'shield' as const,
			href: '#izin-data'
		}
	] as const;

	const roles = [
		{ title: 'Pasien', text: 'Ikuti perawatan dan kelola persetujuan Anda.' },
		{ title: 'Dokter', text: 'Lanjutkan perawatan dengan konteks pasien.' },
		{ title: 'Laboratorium', text: 'Terima permintaan dan kirim hasil pemeriksaan.' },
		{ title: 'Apotek', text: 'Terima resep dan perbarui penyiapan obat.' }
	];

	function close() {
		cancelClose();
		activeMenu = null;
		mobileOpen = false;
	}

	function toggle(menu: 'features' | 'roles', event: MouseEvent) {
		cancelClose();
		lastTrigger = event.currentTarget as HTMLButtonElement;
		activeMenu = activeMenu === menu ? null : menu;
	}
</script>

<svelte:window
	onpointerdown={(event) => {
		if (header && !header.contains(event.target as Node)) close();
	}}
	onkeydown={(event) => {
		if (event.key === 'Escape' && (activeMenu || mobileOpen)) {
			if (activeMenu) {
				activeMenu = null;
				lastTrigger?.focus();
			} else {
				close();
			}
		}
	}}
/>

<header
	class="landing-header site-header"
	bind:this={header}
	onfocusout={(event) => {
		if (event.relatedTarget && !header.contains(event.relatedTarget as Node)) close();
	}}
>
	<a class="brand" href={resolve('/')} aria-label="Hygeia, beranda">
		<img src="/logo-light.webp" width="34" height="34" alt="" /><span>Hygeia</span>
	</a>
	<nav id="landing-navigation" class:mobile-open={mobileOpen} aria-label="Navigasi beranda">
		<button
			aria-expanded={activeMenu === 'features'}
			aria-controls="nav-features"
			onpointerenter={(event) => openOnHover('features', event)}
			onclick={(event) => toggle('features', event)}
		>
			Fitur <svg
				class:turned={activeMenu === 'features'}
				width="12"
				height="12"
				viewBox="0 0 16 16"
				fill="none"
				aria-hidden="true"><path d="m4 6 4 4 4-4" stroke="currentColor" stroke-width="1.5" /></svg
			>
		</button>
		<button
			aria-expanded={activeMenu === 'roles'}
			aria-controls="nav-roles"
			onpointerenter={(event) => openOnHover('roles', event)}
			onclick={(event) => toggle('roles', event)}
		>
			Untuk siapa <svg
				class:turned={activeMenu === 'roles'}
				width="12"
				height="12"
				viewBox="0 0 16 16"
				fill="none"
				aria-hidden="true"><path d="m4 6 4 4 4-4" stroke="currentColor" stroke-width="1.5" /></svg
			>
		</button>
		<a href="#cara-kerja" onclick={close}>Cara kerja</a>
		<a href="#tentang" onclick={close}>Tentang</a>
	</nav>
	<div class="header-actions">
		<a class="button primary nav-login" href={resolve('/login')}
			>Masuk <span aria-hidden="true">→</span></a
		>
	</div>

	{#if activeMenu}
		<div class="nav-panel" id={activeMenu === 'features' ? 'nav-features' : 'nav-roles'}>
			<div class="panel-intro">
				<h2>
					{activeMenu === 'features'
						? 'Perawatan, saling terhubung.'
						: 'Satu alur, untuk setiap peran.'}
				</h2>
				<p>
					{activeMenu === 'features'
						? 'Dari pemeriksaan sampai resep, ikuti setiap langkah dengan informasi yang jelas.'
						: 'Pasien, dokter, laboratorium, dan apotek melanjutkan perjalanan yang sama.'}
				</p>
				<a href={activeMenu === 'features' ? '#cara-kerja' : '#untuk-siapa'} onclick={close}
					>Lihat {activeMenu === 'features' ? 'cara kerjanya' : 'semua peran'}
					<span aria-hidden="true">→</span></a
				>
			</div>
			<div class="panel-links">
				{#if activeMenu === 'features'}
					{#each features as feature (feature.title)}
						<a href={resolve(`/${feature.href}`)} onclick={close}
							><span class="nav-icon"><HealthIcon name={feature.icon} /></span><span
								><strong>{feature.title}</strong><small>{feature.text}</small></span
							></a
						>
					{/each}
				{:else}
					{#each roles as role, index (role.title)}
						<a
							href="#untuk-siapa"
							onclick={() => {
								if (onSelectRole) onSelectRole(index);
								close();
							}}
							><span><strong>{role.title}</strong><small>{role.text}</small></span><span
								class="role-arrow"
								aria-hidden="true">↗</span
							></a
						>
					{/each}
				{/if}
			</div>
		</div>
	{/if}
</header>

<style>
	.site-header {
		position: sticky;
		top: 14px;
		z-index: 50;
		max-width: 1180px;
		margin: 12px auto 0;
		padding: 10px 24px;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 28px;
		background: rgba(255, 255, 255, 0.88);
		backdrop-filter: blur(20px);
		-webkit-backdrop-filter: blur(20px);
		border: 1px solid rgba(216, 229, 243, 0.9);
		border-radius: 9999px;
		box-shadow:
			0 10px 30px -8px rgba(10, 31, 68, 0.08),
			0 2px 6px -1px rgba(10, 31, 68, 0.04);
		transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
	}
	.brand {
		display: inline-flex;
		align-items: center;
		gap: 10px;
		font-size: 21px;
		font-weight: 700;
		letter-spacing: -0.5px;
		color: var(--brand-navy);
		text-decoration: none;
		min-height: 44px;
		white-space: nowrap;
	}
	nav {
		display: flex;
		align-items: center;
		gap: 20px;
		margin: 0 auto;
	}
	nav > a,
	nav > button {
		display: inline-flex;
		align-items: center;
		justify-content: space-between;
		gap: 6px;
		min-height: 44px;
		padding: 0 10px;
		font-size: 14px;
		font-weight: 500;
		color: var(--muted);
		background: none;
		border: 0;
		border-radius: 20px;
		text-decoration: none;
		cursor: pointer;
		white-space: nowrap;
		transition:
			color 0.18s ease,
			background-color 0.18s ease;
	}
	nav > a:hover,
	nav > button:hover,
	nav > button[aria-expanded='true'] {
		color: var(--landing-action);
		background: rgba(0, 102, 255, 0.06);
	}
	.turned {
		transform: rotate(180deg);
	}
	.header-actions {
		display: flex;
		align-items: center;
		gap: 12px;
	}
	.nav-login {
		min-height: 44px;
		padding: 10px 22px;
		display: inline-flex;
		align-items: center;
		gap: 8px;
		border-radius: 9999px;
		background: var(--landing-action);
		color: white;
		font-size: 14px;
		font-weight: 600;
		text-decoration: none;
		white-space: nowrap;
		box-shadow: 0 4px 14px -3px rgba(0, 82, 204, 0.4);
		transition:
			transform 0.15s ease,
			box-shadow 0.2s ease,
			background-color 0.2s ease;
	}
	.nav-login:hover {
		background: #0047b3;
		transform: translateY(-1px);
		box-shadow: 0 6px 18px -3px rgba(0, 82, 204, 0.5);
	}
	.nav-login:active {
		transform: scale(0.98);
	}
	.nav-panel {
		position: absolute;
		top: calc(100% + 10px);
		left: 20px;
		right: 20px;
		display: grid;
		grid-template-columns: 0.9fr 2fr;
		background: var(--landing-surface);
		border: 1px solid var(--landing-rule);
		border-radius: 20px;
		box-shadow: 0 24px 48px -12px rgba(10, 31, 68, 0.18);
		overflow: hidden;
		animation: dropdownFade 0.2s ease-out;
	}
	@keyframes dropdownFade {
		from {
			opacity: 0;
			transform: translateY(-6px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}
	.panel-intro {
		padding: 32px;
		background: var(--landing-paper);
		border-right: 1px solid var(--landing-rule-soft);
	}
	.panel-intro h2 {
		font-size: 20px;
		line-height: 1.35;
		letter-spacing: -0.5px;
		font-weight: 700;
	}
	.panel-intro p {
		margin-top: 10px;
		color: var(--muted);
		font-size: 13px;
		line-height: 1.8;
	}
	.panel-intro > a {
		display: inline-flex;
		align-items: center;
		min-height: 44px;
		gap: 8px;
		margin-top: 16px;
		font-size: 13px;
		font-weight: 600;
		color: var(--landing-action);
		text-decoration: none;
		white-space: nowrap;
	}
	.panel-intro > a:hover {
		text-decoration: underline;
		text-underline-offset: 4px;
	}
	.panel-links {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 8px;
		padding: 24px;
		align-content: center;
	}
	.panel-links > a {
		display: flex;
		align-items: flex-start;
		gap: 14px;
		min-height: 72px;
		padding: 14px 16px;
		border-radius: 12px;
		text-decoration: none;
		color: var(--text);
		white-space: nowrap;
		transition:
			background-color 0.15s ease,
			transform 0.15s ease;
	}
	.panel-links > a:hover {
		background: var(--landing-selected);
		transform: translateY(-1px);
	}
	.panel-links strong {
		display: block;
		font-size: 14px;
		font-weight: 600;
		color: var(--brand-navy);
	}
	.panel-links small {
		display: block;
		margin-top: 4px;
		color: var(--muted);
		font-size: 12px;
		line-height: 1.6;
	}
	.nav-icon {
		color: var(--landing-action);
		padding-top: 2px;
	}
	.role-arrow {
		margin-left: auto;
		color: var(--landing-action);
	}
	@media (max-width: 1050px) {
		.site-header {
			margin-inline: 24px;
			gap: 16px;
		}
		nav {
			gap: 12px;
		}
	}
	@media (max-width: 760px) {
		.site-header {
			top: 8px;
			margin: 8px 16px 0;
			padding: 10px 16px;
			border-radius: 20px;
			flex-wrap: wrap;
			gap: 10px;
		}
		.nav-login {
			margin-left: auto;
			padding: 8px 16px;
			min-height: 44px;
		}
		nav {
			width: 100%;
			order: 3;
			justify-content: flex-start;
			overflow-x: auto;
			padding-bottom: 2px;
			gap: 8px;
		}
		.nav-panel {
			position: static;
			order: 4;
			width: 100%;
			grid-template-columns: 1fr;
			border: 0;
			border-top: 1px solid var(--landing-rule-soft);
			border-radius: 14px;
			box-shadow: none;
			padding: 0;
		}
		.panel-intro {
			display: none;
		}
		.panel-links {
			padding: 8px 0;
			grid-template-columns: 1fr;
			gap: 4px;
		}
		.panel-links > a {
			min-height: 52px;
			padding: 10px 12px;
		}
	}
	@media (max-width: 380px) {
		.site-header {
			margin: 6px 10px 0;
			padding: 8px 12px;
		}
		.brand {
			font-size: 18px;
		}
		.nav-login {
			padding: 8px 12px;
			font-size: 13px;
		}
		nav {
			gap: 4px;
		}
		nav > a,
		nav > button {
			padding: 0 4px;
			font-size: 13px;
		}
	}
</style>
