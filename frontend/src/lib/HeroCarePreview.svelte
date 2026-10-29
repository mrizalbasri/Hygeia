<script lang="ts">
	import HealthIcon from './HealthIcon.svelte';
</script>

<figure class="care-preview" aria-label="Ruang pasien Hygeia">
	<div class="preview-notice">
		<HealthIcon name="shield" /><span>Data Anda. Persetujuan Anda.</span>
	</div>
	<div class="preview-body">
		<div class="preview-brand">
			<img src="/logo-light.png" width="18" height="18" alt="" /><span>HYGEIA</span>
		</div>
		<p class="preview-title">Perawatan<br /><span>Anda.</span></p>
		<p class="preview-description">Satu perjalanan.<br />Terhubung di setiap langkah.</p>
		<div class="care-path" aria-label="Alur dokter, laboratorium, dan apotek">
			<div><span class="path-line"></span><HealthIcon name="care" /><span>Dokter</span></div>
			<div><span class="path-line"></span><HealthIcon name="lab" /><span>Laboratorium</span></div>
			<div class="path-current">
				<span class="path-line"></span><HealthIcon name="medicine" /><span>Apotek</span>
			</div>
		</div>
	</div>
	<div class="preview-summary">
		<p class="summary-label">DALAM SATU RUANG</p>
		<div class="summary-row"><span>Pemeriksaan</span><HealthIcon name="lab" /></div>
		<div class="summary-row"><span>Resep & obat</span><HealthIcon name="medicine" /></div>
		<div class="summary-row"><span>Persetujuan akses</span><HealthIcon name="shield" /></div>
	</div>
</figure>

<style>
	.care-preview {
		position: relative;
		width: 100%;
		margin: 0;
		padding: 16px 14px 0;
		border-radius: 28px;
		background: #080d14;
		color: #f5f8fc;
		font-size: 12px;
		box-shadow: 0 24px 48px rgb(7 19 35 / 24%);
	}
	.care-preview::before {
		content: '';
		position: absolute;
		inset: -11px;
		border: 1px solid rgb(255 255 255 / 48%);
		border-radius: 38px;
		pointer-events: none;
	}
	.preview-notice {
		display: flex;
		gap: 7px;
		align-items: center;
		padding: 9px;
		border-radius: 8px;
		background: #202730;
		color: #e1e9f3;
		font-size: 9px;
	}
	.preview-notice :global(svg) {
		width: 12px;
		height: 12px;
	}
	.preview-body {
		padding: 30px 3px 24px;
	}
	.preview-brand {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 9px;
		letter-spacing: 0.12em;
	}
	.preview-title {
		margin-top: 20px;
		font-size: 42px;
		font-weight: 500;
		line-height: 1.05;
		letter-spacing: -0.035em;
	}
	.preview-title span {
		color: #91a0b3;
	}
	.preview-description {
		margin-top: 16px;
		color: #acb9ca;
		font-size: 11px;
		line-height: 1.75;
	}
	.care-path {
		display: flex;
		align-items: flex-end;
		gap: 16px;
		margin: 28px 0 0;
		padding: 0;
		max-width: none;
	}
	.care-path::before {
		display: none;
	}
	.care-path > div {
		position: relative;
		display: flex;
		flex: 1;
		flex-direction: column;
		align-items: flex-start;
		gap: 10px;
		color: #b1bed0;
		font-size: 8px;
	}
	.path-line {
		display: block;
		width: 1px;
		height: 44px;
		background: #687a91;
	}
	.care-path > div:nth-child(2) .path-line {
		height: 68px;
	}
	.care-path > div:nth-child(3) .path-line {
		height: 94px;
		background: #78b8ff;
	}
	.care-path :global(svg) {
		width: 15px;
		height: 15px;
	}
	.care-path .path-current {
		color: #8fc4ff;
	}
	.preview-summary {
		margin-inline: -14px;
		padding: 22px 17px 20px;
		border-radius: 0 0 28px 28px;
		background: #151d27;
	}
	.summary-label {
		color: #b1bed0;
		letter-spacing: 0.12em;
		font-size: 8px;
		margin-bottom: 14px;
	}
	.summary-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 0;
		border-top: 1px solid #344150;
		font-size: 12px;
	}
	.summary-row :global(svg) {
		width: 16px;
		height: 16px;
		color: #9fb6d0;
	}
</style>
