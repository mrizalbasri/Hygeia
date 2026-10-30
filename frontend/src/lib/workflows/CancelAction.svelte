<script lang="ts">
	import { type Run } from './api.js';
	let {
		path,
		run,
		busy,
		provider = false
	}: { path: string; run: Run; busy: boolean; provider?: boolean } = $props();
	let reason = $state('');
	let error = $state('');
	async function cancel(event: SubmitEvent) {
		event.preventDefault();
		const note = reason.trim();
		if (Array.from(note).length < 5 || Array.from(note).length > 500) {
			error = 'Isi alasan antara 5 dan 500 karakter.';
			return;
		}
		error = '';
		if (
			!confirm(
				'Batalkan layanan ini? Pemrosesan ditutup. Layanan pengganti memerlukan permintaan dan persetujuan baru.'
			)
		)
			return;
		if (await run(`${path}/cancel`, { reason: note })) reason = '';
	}
</script>

<details>
	<summary>{provider ? 'Tidak dapat melayani' : 'Batalkan layanan'}</summary>
	<form onsubmit={cancel}>
		<p>
			Pembatalan menutup pemrosesan dan memberi tahu pihak terkait. Dokter dapat mengajukan layanan
			pengganti; pasien menyetujui kembali penyedianya.
		</p>
		<label
			>Alasan pembatalan<textarea bind:value={reason} required maxlength="1000" disabled={busy}
			></textarea></label
		>
		<small>5–500 karakter. Alasan dapat dilihat pasien dan dokter yang menangani.</small>
		{#if error}<p class="wf-error" role="alert">{error}</p>{/if}
		<button class="button secondary" disabled={busy}>Konfirmasi pembatalan layanan</button>
	</form>
</details>
