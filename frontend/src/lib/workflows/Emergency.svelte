<script lang="ts">
	import PagedList from './PagedList.svelte';
	import { dateText, formText, type Entry, type Run } from './api.js';
	let {
		doctor,
		run,
		busy,
		revision,
		show
	}: {
		doctor: boolean;
		run: Run;
		busy: boolean;
		revision: number;
		show: (kind: string, id: string, expires?: string) => void;
	} = $props();
	let selected = $state<Entry | null>(null);
	async function start(event: SubmitEvent) {
		event.preventDefault();
		const form = event.currentTarget as HTMLFormElement,
			data = new FormData(form);
		if (
			!confirm(
				'Mulai akses darurat selama 60 menit? Akses dan alasan dicatat untuk ditinjau; pasien akan mendapat notifikasi.'
			)
		)
			return;
		if (
			await run('/break-glass', {
				patient_ref: formText(data, 'patient_ref'),
				reason_code: formText(data, 'reason_code'),
				reason_text: formText(data, 'reason_text')
			})
		) {
			form.reset();
			selected = null;
		}
	}
</script>

<p class="wf-warning">
	Akses darurat hanya untuk kebutuhan mendesak yang dapat dipertanggungjawabkan. Grant tidak memberi
	hak mengubah data.
</p>
{#if doctor}<details class="wf-form-panel">
		<summary>Mulai akses darurat</summary>
		<form onsubmit={start}>
			<label>Kode pasien<input name="patient_ref" required maxlength="128" /></label><label
				>Alasan<select name="reason_code" aria-label="Alasan" required
					><option value="">Pilih alasan</option><option value="unconscious"
						>Pasien tidak sadar</option
					><option value="unreachable_patient">Pasien tidak dapat dihubungi</option><option
						value="life_threatening">Kondisi mengancam nyawa</option
					></select
				></label
			><label
				>Penjelasan alasan<textarea name="reason_text" required minlength="10" maxlength="1000"
				></textarea></label
			><button class="button primary" disabled={busy}>Mulai akses darurat 60 menit</button>
		</form>
	</details>{/if}
<PagedList
	path="/break-glass"
	listKey="sessions"
	{revision}
	empty="Belum ada riwayat akses darurat."
	>{#snippet children(item)}<div class="wf-row-heading">
			<h2>{doctor ? item.patient_ref : 'Sesi akses darurat'}</h2>
			<span class="health-status"
				>{Date.parse(item.expires_at || '') > Date.now() ? 'Masih berlaku' : 'Berakhir'}</span
			>
		</div>
		<p>Mulai: {dateText(item.starts_at)} · Berakhir: {dateText(item.expires_at)}</p>
		<p>Alasan: {item.reason_text}</p>
		<p>Tinjauan: {item.review_decision || 'Belum ditinjau'}</p>
		{#if doctor}<button class="button secondary" onclick={() => (selected = item)}
				>Lihat data dalam cakupan</button
			>{/if}{/snippet}</PagedList
>
{#if doctor && selected}<section class="wf-form-panel">
		<h2>Data dalam sesi yang dipilih</h2>
		<p>
			Berlaku sampai {dateText(selected.expires_at)}. Server memeriksa izin pada setiap pembacaan.
		</p>
		<button class="button secondary" onclick={() => (selected = null)}>Tutup daftar data</button
		>{#key selected.id}<PagedList
				path={`/break-glass/${selected.id}/resources`}
				listKey="resources"
				{revision}
				empty="Tidak ada data dalam cakupan sesi ini."
				>{#snippet children(item)}<p>
						{item.resource_type === 'lab_result' ? 'Hasil pemeriksaan' : 'Resep'}
					</p>
					<code>{item.id}</code><button
						class="button secondary"
						onclick={() => show(item.resource_type!, item.id, selected?.expires_at)}
						>Buka detail data</button
					>{/snippet}</PagedList
			>{/key}
	</section>{/if}
