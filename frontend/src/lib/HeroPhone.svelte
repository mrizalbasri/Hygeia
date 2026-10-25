<script lang="ts">
	import HealthIcon from './HealthIcon.svelte';
</script>

<figure class="hero-phone">
	<div class="hero-phone-container">
		<div class="phone-ambient-glow" aria-hidden="true"></div>

		<svg viewBox="110 110 910 1160" role="img" aria-labelledby="phone-title">
			<title id="phone-title"
				>Ilustrasi ruang pasien Hygeia pada aset ponsel yang diberikan pemilik project</title
			>
			<defs>
				<linearGradient
					id="phone-edge"
					x1="110"
					y1="0"
					x2="1020"
					y2="0"
					gradientUnits="userSpaceOnUse"
				>
					<stop offset="0" stop-color="black" /><stop offset="0.1" stop-color="white" /><stop
						offset="0.9"
						stop-color="white"
					/><stop offset="1" stop-color="black" />
				</linearGradient>
				<mask
					id="phone-image-mask"
					x="110"
					y="110"
					width="910"
					height="1160"
					maskUnits="userSpaceOnUse"
					><rect x="110" y="110" width="910" height="1160" fill="url(#phone-edge)" /></mask
				>
				<clipPath id="phone-screen-clip"
					><rect x="342" y="199" width="440" height="964" rx="66" /></clipPath
				>
				<mask
					id="phone-camera-mask"
					maskUnits="userSpaceOnUse"
					x="342"
					y="199"
					width="440"
					height="964"
					><rect x="342" y="199" width="440" height="964" fill="white" /><rect
						x="487"
						y="209"
						width="150"
						height="40"
						rx="20"
						fill="black"
					/></mask
				>
			</defs>
			<image href="/hero-phone-front.png" width="1122" height="1402" mask="url(#phone-image-mask)" />
			<foreignObject
				x="342"
				y="199"
				width="440"
				height="964"
				clip-path="url(#phone-screen-clip)"
				mask="url(#phone-camera-mask)"
				aria-hidden="true"
			>
				<div class="phone-screen">
					<div class="phone-ui-brand">
						<img src="/logo-light.png" alt="" width="34" height="34" /><strong>Hygeia</strong><span
							>Pratinjau</span
						>
					</div>
					<p class="phone-greeting">Ruang pasien Anda</p>
					<p class="phone-ui-title">Perjalanan Saya</p>
					<p class="phone-ui-intro">Ikuti perawatan Anda,<br />satu langkah pada satu waktu.</p>
					<div class="phone-care">
						<HealthIcon name="care" />
						<p>Perawatan Anda</p>
						<strong>Terhubung<br />dengan dokter.</strong>
					</div>
					<div class="phone-ui-row">
						<HealthIcon name="lab" />
						<div><strong>Pemeriksaan</strong><span>Permintaan dan status lab</span></div>
					</div>
					<div class="phone-ui-row">
						<HealthIcon name="medicine" />
						<div><strong>Resep & obat</strong><span>Aturan pakai dan status resep</span></div>
					</div>
					<div class="phone-ui-row">
						<HealthIcon name="history" />
						<div><strong>Riwayat perawatan</strong><span>Ikuti perkembangan terbaru</span></div>
					</div>
					<p class="phone-ui-footer">Perawatan terhubung.<br />Kendali di tangan Anda.</p>
				</div>
			</foreignObject>
		</svg>
	</div>
</figure>

<style>
	.hero-phone {
		margin: 0;
		width: 100%;
		position: relative;
	}
	.hero-phone-container {
		position: relative;
		width: 100%;
	}
	.phone-ambient-glow {
		position: absolute;
		top: 50%;
		left: 50%;
		width: 80%;
		height: 80%;
		transform: translate(-50%, -50%);
		background: radial-gradient(circle, rgba(0, 102, 255, 0.12) 0%, rgba(0, 214, 255, 0.04) 50%, transparent 70%);
		border-radius: 50%;
		filter: blur(40px);
		z-index: 0;
		pointer-events: none;
	}
	svg {
		display: block;
		width: 100%;
		height: auto;
		position: relative;
		z-index: 1;
		filter: drop-shadow(0 20px 30px rgba(10, 31, 68, 0.08));
	}
	.phone-screen {
		/* 360 CSS pixels fit the asset's 440-pixel screen; the SVG mask preserves its camera. */
		transform: scale(1.222222222);
		transform-origin: 0 0;
		width: 360px;
		height: 789px;
		border-radius: 54px;
		overflow: hidden;
		background: var(--landing-paper);
		color: var(--text);
		padding: 64px 24px 24px;
		font-family: inherit;
	}
	.phone-ui-brand {
		display: flex;
		gap: 8px;
		align-items: center;
		font-size: 24px;
	}
	.phone-ui-brand span {
		margin-left: auto;
		font-size: 11px;
		color: var(--muted);
	}
	.phone-greeting {
		font-size: 15px;
		margin-top: 24px;
		color: var(--muted);
	}
	.phone-ui-title {
		font-size: 32px;
		font-weight: 700;
		line-height: 1.12;
		letter-spacing: -1.5px;
		margin-top: 12px;
	}
	.phone-ui-intro {
		font-size: 14px;
		line-height: 1.8;
		margin-top: 16px;
		color: var(--muted);
	}
	.phone-care {
		background: var(--landing-selected);
		border-radius: 18px;
		padding: 18px;
		margin-top: 20px;
	}
	.phone-care > :global(svg) {
		color: var(--landing-action);
		width: 28px;
		height: 28px;
	}
	.phone-care p {
		font-size: 14px;
		color: var(--landing-action);
		margin-top: 12px;
	}
	.phone-care strong {
		font-size: 23px;
		line-height: 1.35;
		display: block;
		margin-top: 8px;
	}
	.phone-ui-row {
		display: flex;
		gap: 14px;
		padding: 16px 0;
		border-bottom: 1px solid var(--landing-rule);
	}
	.phone-ui-row > :global(svg) {
		width: 22px;
		height: 22px;
		flex-shrink: 0;
		color: var(--landing-action);
	}
	.phone-ui-row strong {
		font-size: 16px;
	}
	.phone-ui-row span {
		display: block;
		margin-top: 6px;
		font-size: 12px;
		color: var(--muted);
	}
	.phone-ui-footer {
		margin-top: 16px;
		font-size: 12px;
		line-height: 1.7;
		color: var(--muted);
	}
</style>
