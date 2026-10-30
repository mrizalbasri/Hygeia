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
		padding-inline: 22px;
		font-size: 13px;
	}
	.hero-three-actions .button:hover {
		background: var(--landing-action-hover);
	}
	.hero-three-link {
		display: inline-flex;
		align-items: center;
		min-height: 44px;
		font-size: 13px;
		color: var(--landing-action);
		text-decoration: underline;
		text-underline-offset: 5px;
		text-decoration-color: #adbfda;
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
		gap: 24px 64px;
		padding: 0;
		margin: 58px auto 42px;
		list-style: none;
	}
	.hero-three-roles a {
		display: flex;
		align-items: center;
		gap: 10px;
		font-size: 16px;
		font-weight: 600;
		color: #526984;
		min-height: 44px;
		text-decoration: none;
		transition: color 180ms ease;
	}
	.hero-three-roles a:hover {
		color: var(--landing-action);
	}
	.hero-three-roles :global(svg) {
		width: 24px;
		height: 24px;
	}
	.hero-three-stage {
		--hero-travel: 0;
		position: relative;
		isolation: isolate;
		padding: 64px 0 76px;
		overflow: hidden;
		border-radius: 24px;
		background: #e4eef8;
	}
	.hero-three-backdrop {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		object-fit: cover;
		object-position: center 44%;
		transform: scale(calc(1 + var(--hero-travel) * 0.045));
	}
	.hero-three-phone {
		position: relative;
		width: 288px;
		max-width: none;
		margin: 0 auto;
		transform: translateY(calc(var(--hero-travel) * -22px));
	}
	@keyframes hero-enter {
		from {
			opacity: 0.35;
			translate: 0 14px;
			filter: blur(1.5px);
		}
		to {
			opacity: 1;
			translate: 0 0;
			filter: blur(0);
		}
	}
	@media (prefers-reduced-motion: no-preference) {
		.hero-three h1,
		.hero-three-description,
		.hero-three-actions,
		.hero-three-roles,
		.hero-three-device {
			animation: hero-enter 560ms cubic-bezier(0.16, 1, 0.3, 1) backwards;
		}
		.hero-three-description {
			animation-delay: 80ms;
		}
		.hero-three-actions {
			animation-delay: 140ms;
		}
		.hero-three-roles {
			animation-delay: 200ms;
		}
		.hero-three-device {
			animation-delay: 260ms;
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
