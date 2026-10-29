<script lang="ts">
	import { api, errorText, formText, uuidPattern, type Entry, type LabResultInput } from './api.js';
	import LabResultReview from './LabResultReview.svelte';
	let { initialResultID = '' }: { initialResultID?: string } = $props();
	let pending = $state(false),
		error = $state(''),
		confirmation = $state<{ id: string; resultID: string } | null>(null);
	let review = $state<{ resultID: string; reason: string; result: LabResultInput } | null>(null);
	let formElement: HTMLFormElement;

	function prepare(event: SubmitEvent) {
		event.preventDefault();
		if (pending) return;
		const form = event.currentTarget as HTMLFormElement;
		const data = new FormData(form);
		const resultID = formText(data, 'result_id');
		const reason = formText(data, 'reason');
		const value = formText(data, 'value');
		const flag = formText(data, 'flag');
		error = '';
		confirmation = null;
		if (
			reason.length < 5 ||
			!value ||
			!['normal', 'high', 'low', 'abnormal', 'unknown'].includes(flag)
		) {
			error = 'Isi alasan koreksi minimal 5 karakter, nilai pengganti, dan pilihan penanda.';
			return;
		}
		review = {
			resultID,
			reason,
			result: {
				value,
				unit: formText(data, 'unit'),
				reference_range: formText(data, 'reference_range'),
				flag: flag as LabResultInput['flag'],
				note: formText(data, 'note')
			}
		};
	}

	async function submit() {
		if (pending || !review) return;
		const { resultID, reason, result } = review;
		error = '';
		pending = true;
		try {
			const response = await api<{ correction: Entry }>(
				`/lab-results/${encodeURIComponent(resultID)}/corrections`,
				'POST',
				{
					reason,
					...result
				}
			);
			confirmation = { id: response.correction.id, resultID };
			formElement.reset();
			review = null;
		} catch (cause) {
			error = errorText(cause);
		} finally {
			pending = false;
		}
	}
</script>

<section class="wf-form-panel" aria-labelledby="correction-title">
	<h2 id="correction-title">Tambahkan koreksi hasil</h2>
	<p class="muted">
		Gunakan ID hasil dari tanda terima laboratorium Anda. Koreksi menambahkan versi pengganti
		beserta alasannya; hasil awal dan riwayat koreksi tetap tersimpan.
	</p>
	<p class="muted">
		Hasil yang telah dikirim tidak dapat dibaca kembali oleh laboratorium. Periksa ID dan isi
		seluruh nilai pengganti sebelum mengirim.
	</p>
	{#if confirmation}
		<div class="wf-success" role="status">
			<p>Koreksi hasil berhasil dikirim.</p>
			<p>ID hasil: <code>{confirmation.resultID}</code></p>
			<p>ID koreksi: <code>{confirmation.id}</code></p>
		</div>
	{/if}
	{#if error}<p class="wf-error" role="alert">{error}</p>{/if}
	{#if review}
		<LabResultReview
			result={review.result}
			reference={`ID hasil: ${review.resultID}`}
			reason={review.reason}
			{pending}
			confirmLabel="Kirim koreksi hasil"
			onconfirm={() => void submit()}
			onedit={() => {
				review = null;
				error = '';
			}}
		/>
	{/if}
	<form bind:this={formElement} onsubmit={prepare} aria-busy={pending} hidden={!!review}>
		<label
			>ID hasil pemeriksaan<input
				name="result_id"
				value={initialResultID}
				pattern={uuidPattern}
				required
				maxlength="36"
				spellcheck="false"
				autocomplete="off"
				aria-describedby="result-id-hint"
				disabled={pending}
			/></label
		>
		<p id="result-id-hint">Salin ID tanda terima hasil, bukan ID permintaan pemeriksaan.</p>
		<label
			>Alasan koreksi<textarea
				name="reason"
				required
				minlength="5"
				maxlength="500"
				rows="3"
				disabled={pending}></textarea></label
		>
		<label
			>Nilai hasil pengganti<input
				name="value"
				required
				maxlength="500"
				disabled={pending}
			/></label
		>
		<label>Satuan (opsional)<input name="unit" maxlength="100" disabled={pending} /></label>
		<label
			>Rentang rujukan (opsional)<input
				name="reference_range"
				maxlength="200"
				disabled={pending}
			/></label
		>
		<label
			>Penanda laboratorium<select name="flag" required disabled={pending}
				><option value="" disabled selected>Pilih penanda hasil</option><option value="normal"
					>Normal</option
				><option value="high">Tinggi</option><option value="low">Rendah</option><option
					value="abnormal">Abnormal</option
				><option value="unknown">Tidak diketahui</option></select
			></label
		>
		<label
			>Catatan teknis (opsional)<textarea name="note" maxlength="2000" rows="3" disabled={pending}
			></textarea></label
		>
		<p>
			Koreksi yang dikirim tidak dapat disunting atau dihapus. Pasien dan dokter yang memiliki izin
			dapat melihat riwayatnya.
		</p>
		<button class="button primary" type="submit" disabled={pending}>Tinjau koreksi hasil</button>
	</form>
</section>
