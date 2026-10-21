<script lang="ts">
	import PagedList from './PagedList.svelte';
	import { formText, statusText, dateText, type Run } from './api.js';
	let {
		doctor,
		run,
		busy,
		revision
	}: { doctor: boolean; run: Run; busy: boolean; revision: number } = $props();
	async function invite(event: SubmitEvent) {
		event.preventDefault();
		const form = event.currentTarget as HTMLFormElement;
		if (await run('/episodes', { patient_ref: formText(new FormData(form), 'patient_ref') }))
			form.reset();
	}
</script>

{#if doctor}<details class="wf-form-panel">
		<summary>Undang pasien</summary>
		<form onsubmit={invite}>
			<p>Gunakan kode pasien persis. Tidak ada pencarian seluruh pasien.</p>
			<label
				>Kode pasien<input
					name="patient_ref"
					required
					maxlength="64"
					pattern="[A-Za-z0-9][A-Za-z0-9_-]*"
					autocomplete="off"
				/></label
			><button class="button primary" disabled={busy}>Kirim undangan</button>
		</form>
	</details>{/if}
<PagedList
	path="/episodes"
	listKey="episodes"
	statuses={['pending', 'active', 'rejected', 'ended']}
	{revision}
	empty={doctor
		? 'Belum ada pasien dalam perawatan Anda. Mulai dengan mengirim undangan.'
		: 'Belum ada undangan atau hubungan perawatan dari dokter.'}
>
	{#snippet children(item)}<div class="wf-row-heading">
			<div>
				<h2>{doctor ? item.patient_ref : item.doctor_name}</h2>
				<p>{item.organization_name}</p>
			</div>
			<span class="health-status">{statusText[item.status || ''] || item.status}</span>
		</div>
		<details>
			<summary>Detail hubungan perawatan</summary>
			<p>Dokter: {item.doctor_name} · Kode pasien: {item.patient_ref}</p>
			<p>Dibuat: {dateText(item.created_at)}</p>
			<code>{item.id}</code>
		</details>
		<div class="wf-actions">
			{#if !doctor && item.status === 'pending'}<button
					class="button primary"
					disabled={busy}
					onclick={() => run(`/episodes/${item.id}`, { action: 'accept' }, 'PATCH')}
					>Terima undangan</button
				><button
					class="button secondary"
					disabled={busy}
					onclick={() => run(`/episodes/${item.id}`, { action: 'reject' }, 'PATCH')}
					>Tolak undangan</button
				>{/if}{#if item.status === 'active'}<button
					class="button secondary"
					disabled={busy}
					onclick={() => {
						if (
							confirm(
								'Akhiri perawatan ini? Akses dokter melalui episode ini berakhir, dan permintaan lab/resep yang belum disetujui dibatalkan. Izin terpisah mungkin tetap berlaku.'
							)
						)
							void run(`/episodes/${item.id}`, { action: 'end' }, 'PATCH');
					}}>Akhiri perawatan</button
				>{/if}
		</div>{/snippet}
</PagedList>
