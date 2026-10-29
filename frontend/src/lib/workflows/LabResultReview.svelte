<script lang="ts">
	import { onMount } from 'svelte';
	import type { LabResultInput } from './api.js';
	let {
		result,
		reference,
		reason,
		pending,
		confirmLabel,
		onconfirm,
		onedit
	}: {
		result: LabResultInput;
		reference: string;
		reason?: string;
		pending: boolean;
		confirmLabel: string;
		onconfirm: () => void;
		onedit: () => void;
	} = $props();
	let heading: HTMLHeadingElement;
	const flags: Record<LabResultInput['flag'], string> = {
		normal: 'Normal',
		high: 'Tinggi',
		low: 'Rendah',
		abnormal: 'Abnormal',
		unknown: 'Tidak diketahui'
	};
	onMount(() => heading.focus());
</script>

<section aria-labelledby="lab-review-title">
	<h3 id="lab-review-title" tabindex="-1" bind:this={heading}>Tinjau sebelum mengirim</h3>
	<p>{reference}</p>
	<dl class="wf-facts">
		{#if reason}<div>
				<dt>Alasan koreksi</dt>
				<dd>{reason}</dd>
			</div>{/if}
		<div>
			<dt>Nilai hasil</dt>
			<dd>{result.value}</dd>
		</div>
		<div>
			<dt>Satuan</dt>
			<dd>{result.unit || 'Tidak diisi'}</dd>
		</div>
		<div>
			<dt>Rentang rujukan</dt>
			<dd>{result.reference_range || 'Tidak diisi'}</dd>
		</div>
		<div>
			<dt>Penanda laboratorium</dt>
			<dd>{flags[result.flag]}</dd>
		</div>
		<div>
			<dt>Catatan teknis</dt>
			<dd>{result.note || 'Tidak diisi'}</dd>
		</div>
	</dl>
	<p>
		Pastikan referensi dan seluruh nilai benar. Kiriman tidak dapat disunting atau dihapus;
		perubahan berikutnya harus melalui koreksi baru.
	</p>
	<div class="wf-actions">
		<button type="button" class="button primary" disabled={pending} onclick={onconfirm}
			>{pending ? 'Mengirim…' : confirmLabel}</button
		>
		<button type="button" class="button secondary" disabled={pending} onclick={onedit}
			>Kembali mengedit</button
		>
	</div>
</section>
