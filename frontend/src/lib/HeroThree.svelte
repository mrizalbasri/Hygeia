<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import HealthIcon from './HealthIcon.svelte';
	import HeroCarePreview from './HeroCarePreview.svelte';
	let { onSelectRole }: { onSelectRole?: (index: number) => void } = $props();
	let stage: HTMLDivElement;

	onMount(() => {
		const motion = window.matchMedia('(prefers-reduced-motion: reduce)');
		let frame = 0;
		function update() {
			frame = 0;
			if (motion.matches) {
				stage.style.setProperty('--hero-travel', '0');
				return;
			}
			const bounds = stage.getBoundingClientRect();
			if (bounds.bottom < 0 || bounds.top > window.innerHeight) return;
			const progress = Math.max(
				0,
				Math.min(1, (window.innerHeight - bounds.top) / (window.innerHeight + bounds.height))
			);
			stage.style.setProperty('--hero-travel', progress.toFixed(4));
		}
		function schedule() {
			if (!frame) frame = requestAnimationFrame(update);
		}
		window.addEventListener('scroll', schedule, { passive: true });
		window.addEventListener('resize', schedule);
		motion.addEventListener('change', schedule);
		update();
		return () => {
			cancelAnimationFrame(frame);
			window.removeEventListener('scroll', schedule);
			window.removeEventListener('resize', schedule);
			motion.removeEventListener('change', schedule);
		};
	});

	// Adapted from https://sv-blocks.vercel.app/d/hero-three.json.
	// Keep its centered heading, role cloud, and wide image with a centered phone.
	const roles = [
		{ label: 'Pasien', icon: 'care' as const },
		{ label: 'Dokter', icon: 'shield' as const },
		{ label: 'Laboratorium', icon: 'lab' as const },
		{ label: 'Apotek', icon: 'medicine' as const }
	];
</script>

<section class="hero-three" aria-labelledby="landing-title">
	<div class="hero-three-copy">
		<h1 id="landing-title">
			<span>Perawatan terhubung.</span><br />Kendali di tangan Anda.
		</h1>
		<p class="hero-three-description">
			Dari pemeriksaan sampai resep, ikuti perjalanan perawatan Anda dalam satu ruang.
		</p>
		<div class="hero-three-actions">
			<a class="button primary" href={resolve('/login')}>
				Masuk ke ruang Anda <span aria-hidden="true">↗</span>
			</a>
			<a class="hero-three-link" href="#cara-kerja">Kenali cara kerjanya</a>
		</div>
	</div>

	<ul class="hero-three-roles" aria-label="Menghubungkan empat peran layanan">
		{#each roles as role, index (role.label)}
			<li>
				<a href="#untuk-siapa" onclick={() => onSelectRole?.(index)}
					><HealthIcon name={role.icon} /><span>{role.label}</span></a
				>
			</li>
		{/each}
	</ul>

	<div class="hero-three-stage" bind:this={stage}>
		<img
			class="hero-three-backdrop"
			src="/hero-clinic-v2.webp"
			alt="Visual buatan AI: pasien dan dokter berdiskusi di ruang konsultasi bernuansa biru."
			width="1860"
			height="845"
			fetchpriority="high"
		/>
		<div class="hero-three-phone">
			<div class="hero-three-device"><HeroCarePreview /></div>
		</div>
	</div>
</section>

<style>
	.hero-three {
		padding: 88px 16px 0;
		max-width: 1600px;
		margin: 0 auto;
	}
	.hero-three-copy {
		max-width: 720px;
		margin: 0 auto;
		text-align: center;
	}
	.hero-three h1 {
		font-size: clamp(34px, 3.6vw, 52px);
		font-weight: 600;
		line-height: 1.2;
		letter-spacing: -0.035em;
		text-wrap: balance;
	}
	.hero-three h1 span {
		color: #526984;
	}
	.hero-three-description {
		max-width: 480px;
		margin: 22px auto 0;
		font-size: 15px;
		line-height: 1.8;
		color: #526984;
		text-wrap: pretty;
	}
	.hero-three-actions {
		display: flex;
		align-items: center;
		justify-content: center;
		flex-wrap: wrap;
		gap: 12px 24px;
		margin-top: 26px;
	}
	.hero-three-actions .button {
		background: var(--landing-action);
		min-height: 46px;
		padding-inline: 24px;
		font-size: 14px;
		font-weight: 600;
		border-radius: 9999px;
		box-shadow: 0 4px 14px -3px rgba(0, 82, 204, 0.38);
		transition:
			transform 200ms cubic-bezier(0.16, 1, 0.3, 1),
			box-shadow 200ms ease,
			background-color 200ms ease;
	}
	.hero-three-actions .button:hover {
		background: var(--landing-action-hover);
		transform: translateY(-2px);
		box-shadow: 0 8px 24px -3px rgba(0, 82, 204, 0.48);
	}
	.hero-three-actions .button:active {
		transform: translateY(0) scale(0.98);
	}
	.hero-three-link {
		display: inline-flex;
		align-items: center;
		min-height: 44px;
		font-size: 14px;
		font-weight: 500;
		color: var(--landing-action);
		text-decoration: underline;
		text-underline-offset: 5px;
		text-decoration-color: #adbfda;
		transition:
			color 180ms ease,
			text-decoration-color 180ms ease;
	}
	.hero-three-link:hover {
		text-decoration-color: currentColor;
	}
	.hero-three a:focus-visible {
		outline: 3px solid var(--landing-action);
		outline-offset: 5px;
	}
	.hero-three-roles {
		display: flex;
		align-items: center;
		justify-content: center;
		flex-wrap: wrap;
		gap: 16px 36px;
		padding: 0;
		margin: 54px auto 38px;
		list-style: none;
	}
	.hero-three-roles a {
		display: inline-flex;
		align-items: center;
		gap: 9px;
		font-size: 15px;
		font-weight: 600;
		color: #4b6482;
		min-height: 44px;
		padding: 6px 16px;
		border-radius: 9999px;
		text-decoration: none;
		background: rgba(255, 255, 255, 0.65);
		border: 1px solid rgba(173, 191, 218, 0.45);
		box-shadow: 0 2px 8px -2px rgba(10, 31, 68, 0.05);
		backdrop-filter: blur(8px);
		transition:
			color 200ms ease,
			background-color 200ms ease,
			border-color 200ms ease,
			transform 200ms cubic-bezier(0.16, 1, 0.3, 1),
			box-shadow 200ms ease;
	}
	.hero-three-roles a:hover {
		color: var(--landing-action);
		background: #ffffff;
		border-color: rgba(0, 102, 255, 0.32);
		transform: translateY(-2px);
		box-shadow: 0 6px 18px -3px rgba(0, 82, 204, 0.16);
	}
	.hero-three-roles a:active {
		transform: translateY(0);
	}
	.hero-three-roles :global(svg) {
		width: 22px;
		height: 22px;
		transition: transform 200ms cubic-bezier(0.16, 1, 0.3, 1);
	}
	.hero-three-roles a:hover :global(svg) {
		transform: scale(1.12);
	}
	.hero-three-stage {
		--hero-travel: 0;
		position: relative;
		isolation: isolate;
		padding: 64px 0 76px;
		overflow: hidden;
		border-radius: 24px;
		background:
			radial-gradient(ellipse 70% 50% at 50% 35%, rgba(0, 102, 255, 0.12), transparent 85%), #e4eef8;
		box-shadow:
			0 24px 54px -16px rgba(8, 28, 58, 0.14),
			0 2px 6px rgba(8, 28, 58, 0.04);
	}
	.hero-three-backdrop {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		object-fit: cover;
		object-position: center 44%;
		transform: scale(calc(1 + var(--hero-travel) * 0.045));
		transition: transform 200ms ease-out;
	}
	.hero-three-phone {
		position: relative;
		width: 288px;
		max-width: none;
		margin: 0 auto;
		transform: translateY(calc(var(--hero-travel) * -22px));
	}
	.hero-three-device {
		will-change: transform;
	}
	@keyframes hero-headline-enter {
		from {
			opacity: 0;
			transform: translateY(28px) scale(0.98);
			filter: blur(8px);
		}
		to {
			opacity: 1;
			transform: translateY(0) scale(1);
			filter: blur(0);
		}
	}
	@keyframes hero-copy-enter {
		from {
			opacity: 0;
			transform: translateY(20px);
			filter: blur(5px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
			filter: blur(0);
		}
	}
	@keyframes hero-actions-enter {
		from {
			opacity: 0;
			transform: translateY(16px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}
	@keyframes role-chip-enter {
		from {
			opacity: 0;
			transform: translateY(18px) scale(0.94);
		}
		to {
			opacity: 1;
			transform: translateY(0) scale(1);
		}
	}
	@keyframes hero-stage-enter {
		from {
			opacity: 0;
			transform: translateY(32px) scale(0.98);
		}
		to {
			opacity: 1;
			transform: translateY(0) scale(1);
		}
	}
	@keyframes hero-device-enter {
		from {
			opacity: 0;
			transform: translateY(44px) scale(0.96);
			filter: blur(4px);
		}
		to {
			opacity: 1;
			transform: translateY(0) scale(1);
			filter: blur(0);
		}
	}
	@keyframes hero-device-float {
		0% {
			transform: translateY(0);
		}
		100% {
			transform: translateY(-8px);
		}
	}
	@media (prefers-reduced-motion: no-preference) {
		.hero-three h1 {
			animation: hero-headline-enter 780ms cubic-bezier(0.16, 1, 0.3, 1) backwards;
		}
		.hero-three-description {
			animation: hero-copy-enter 700ms cubic-bezier(0.16, 1, 0.3, 1) 90ms backwards;
		}
		.hero-three-actions {
			animation: hero-actions-enter 640ms cubic-bezier(0.16, 1, 0.3, 1) 170ms backwards;
		}
		.hero-three-roles li {
			animation: role-chip-enter 580ms cubic-bezier(0.16, 1, 0.3, 1) backwards;
		}
		.hero-three-roles li:nth-child(1) {
			animation-delay: 220ms;
		}
		.hero-three-roles li:nth-child(2) {
			animation-delay: 280ms;
		}
		.hero-three-roles li:nth-child(3) {
			animation-delay: 340ms;
		}
		.hero-three-roles li:nth-child(4) {
			animation-delay: 400ms;
		}
		.hero-three-stage {
			animation: hero-stage-enter 880ms cubic-bezier(0.16, 1, 0.3, 1) 240ms backwards;
		}
		.hero-three-device {
			animation:
				hero-device-enter 900ms cubic-bezier(0.16, 1, 0.3, 1) 360ms backwards,
				hero-device-float 5s ease-in-out infinite alternate 1.3s;
		}
	}
	@media (min-width: 1700px) {
		.hero-three {
			padding-top: 112px;
		}
	}
	@media (max-width: 760px) {
		.hero-three {
			padding: 64px 12px 0;
		}
		.hero-three-copy {
			padding-inline: 12px;
		}
		.hero-three h1 {
			font-size: clamp(30px, 5.8vw, 42px);
		}
		.hero-three-description {
			max-width: 360px;
			font-size: 14px;
		}
		.hero-three-roles {
			display: grid;
			grid-template-columns: repeat(2, max-content);
			gap: 20px 28px;
			margin: 38px auto 30px;
			max-width: 420px;
		}
		.hero-three-roles a {
			font-size: 14px;
		}
		.hero-three-stage {
			padding: 52px 0 76px;
			border-radius: 16px;
		}
		.hero-three-backdrop {
			object-position: 47% center;
		}
		.hero-three-phone {
			width: min(264px, calc(100% - 48px));
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.hero-three-backdrop,
		.hero-three-phone {
			transform: none;
		}
	}
</style>
