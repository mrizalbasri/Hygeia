<script lang="ts">
	import WorkspaceIcon from './WorkspaceIcon.svelte';
	import { resolve } from '$app/paths';
	let {
		name,
		role,
		initials,
		showProfile,
		profileActive,
		pending,
		ready,
		onlogout
	}: {
		name: string;
		role: string;
		initials: string;
		showProfile: boolean;
		profileActive: boolean;
		pending: boolean;
		ready: boolean;
		onlogout: () => void;
	} = $props();
	let open = $state(false);
	let container: HTMLDivElement;
	let trigger: HTMLButtonElement;
	function close(restoreFocus = false) {
		open = false;
		if (restoreFocus) trigger.focus();
	}
</script>

<svelte:window
	onpointerdown={(event) => {
		if (open && !container.contains(event.target as Node)) close();
	}}
	onkeydown={(event) => {
		if (open && event.key === 'Escape') {
			event.preventDefault();
			event.stopImmediatePropagation();
			close(true);
		}
	}}
/>

<div
	class="account-dropdown"
	bind:this={container}
	onfocusout={(event) => {
		if (event.relatedTarget && !container.contains(event.relatedTarget as Node)) close();
	}}
>
	<button
		class="account-trigger"
		bind:this={trigger}
		type="button"
		aria-label={`Menu akun ${name}`}
		aria-expanded={open}
		aria-controls="account-navigation"
		disabled={!ready || pending}
		onclick={() => (open = !open)}
	>
		<span class="account-avatar" aria-hidden="true">{initials}</span>
		<span class="account-trigger-name">{name}</span>
		<svg
			class:expanded={open}
			width="14"
			height="14"
			viewBox="0 0 16 16"
			fill="none"
			aria-hidden="true"><path d="m4 6 4 4 4-4" stroke="currentColor" stroke-width="1.5" /></svg
		>
	</button>
	{#if open}
		<nav class="account-panel" id="account-navigation" aria-label="Menu akun">
			<div class="account-panel-identity"><strong>{name}</strong><span>{role}</span></div>
			{#if showProfile}
				<a
					href={resolve('/dashboard/[role]?view=profile', { role: 'patient' })}
					aria-current={profileActive ? 'page' : undefined}
					onclick={() => close()}><WorkspaceIcon name="profile" />Profil saya</a
				>
			{/if}
			<button
				class="account-logout"
				type="button"
				disabled={pending}
				onclick={() => {
					close();
					onlogout();
				}}><WorkspaceIcon name="logout" />{pending ? 'Keluar…' : 'Keluar'}</button
			>
		</nav>
	{/if}
</div>

<style>
	.account-dropdown {
		position: relative;
	}
	.account-trigger {
		display: flex;
		align-items: center;
		gap: 10px;
		min-height: 44px;
		max-width: 280px;
		padding: 6px 10px 6px 6px;
		border: 1px solid transparent;
		border-radius: 999px;
		background: transparent;
		color: var(--dashboard-ink);
		font: inherit;
		font-size: 13px;
		font-weight: 600;
	}
	.account-trigger:hover,
	.account-trigger[aria-expanded='true'] {
		background: var(--dashboard-hover);
		border-color: var(--dashboard-line);
	}
	.account-trigger-name {
		overflow: hidden;
		white-space: nowrap;
		text-overflow: ellipsis;
	}
	.account-trigger > svg {
		flex-shrink: 0;
		transition: transform 0.15s;
	}
	.account-trigger > svg.expanded {
		transform: rotate(180deg);
	}
	.account-panel {
		position: absolute;
		right: 0;
		top: calc(100% + 10px);
		z-index: 60;
		width: min(272px, calc(100vw - 32px));
		padding: 6px;
		background: var(--dashboard-surface);
		border: 1px solid var(--dashboard-line);
		border-radius: 14px;
		box-shadow: 0 12px 36px #0a1f441a;
	}
	.account-panel-identity {
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 12px 12px 16px;
		margin-bottom: 6px;
		border-bottom: 1px solid var(--dashboard-line);
		overflow-wrap: anywhere;
	}
	.account-panel-identity strong {
		font-size: 14px;
		line-height: 1.5;
	}
	.account-panel-identity > span {
		font-size: 12px;
		color: var(--dashboard-muted);
	}
	.account-panel a,
	.account-panel button {
		display: flex;
		align-items: center;
		gap: 12px;
		width: 100%;
		min-height: 44px;
		padding: 10px 12px;
		border: 0;
		border-radius: 8px;
		background: transparent;
		color: var(--dashboard-ink);
		font: inherit;
		font-size: 14px;
		text-decoration: none;
	}
	.account-panel a:hover,
	.account-panel button:hover,
	.account-panel a[aria-current='page'] {
		background: var(--dashboard-hover);
	}
	.account-panel a[aria-current='page'] {
		color: var(--dashboard-action);
	}
	.account-panel .account-logout {
		color: #b42318;
	}
	@media (max-width: 1050px) {
		.account-trigger-name {
			display: none;
		}
	}
	@media (max-width: 760px) {
		.account-trigger {
			padding: 4px 6px;
			gap: 6px;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.account-trigger > svg {
			transition: none;
		}
	}
</style>
