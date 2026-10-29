<script lang="ts">
	import PagedList from './PagedList.svelte';
	import DirectoryPicker from './DirectoryPicker.svelte';
	import CancelAction from './CancelAction.svelte';
	import { formText, statusText, dateText, type Run, type Medication } from './api.js';
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
		show: (kind: string, id: string) => void;
	} = $props();
	let episode = $state(''),
		pharmacy = $state(''),
		approving = $state(''),
		fields = $state([1]),
		nextField = 2;
	async function create(event: SubmitEvent) {
		event.preventDefault();
		const form = event.currentTarget as HTMLFormElement,
			data = new FormData(form);
		const items: Medication[] = fields.map((id) => ({
			medication_name: formText(data, `name-${id}`),
			dosage: formText(data, `dosage-${id}`),
			instructions: formText(data, `instructions-${id}`),
			quantity: Number(data.get(`quantity-${id}`))
		}));
		if (episode && (await run('/prescriptions', { episode_id: episode, items }))) {
			form.reset();
			episode = '';
			fields = [nextField++];
		}
	}
	async function approve(event: SubmitEvent) {
		event.preventDefault();
		if (
			pharmacy &&
			(await run(`/prescriptions/${approving}/approval`, {
				decision: 'approve',
				pharmacy_org_id: pharmacy
			}))
		) {
			approving = '';
			pharmacy = '';
		}
	}
</script>

{#if doctor}<details class="wf-form-panel">
		<summary>Buat resep</summary>
		<form onsubmit={create}>
			<DirectoryPicker
				path="/episodes?status=active"
				listKey="episodes"
				label="Perawatan aktif"
				bind:value={episode}
			/>
			<p>Resep hanya diterbitkan oleh dokter. Setelah diterbitkan, item tidak dapat diedit.</p>
			{#each fields as id, index (id)}<fieldset>
					<legend>Obat {index + 1}</legend><label
						>Nama obat<input name={`name-${id}`} required maxlength="200" /></label
					><label>Dosis<input name={`dosage-${id}`} required maxlength="200" /></label><label
						>Aturan pakai<textarea name={`instructions-${id}`} required maxlength="1000"
						></textarea></label
					><label
						>Jumlah<input
							name={`quantity-${id}`}
							type="number"
							min="1"
							max="10000"
							step="1"
							value="1"
							required
						/></label
					><button
						type="button"
						class="button secondary"
						disabled={busy || fields.length === 1}
						onclick={() => (fields = fields.filter((value) => value !== id))}
						>Hapus obat {index + 1}</button
					>
				</fieldset>{/each}
			<div class="wf-actions">
				<button
					type="button"
					class="button secondary"
					disabled={busy || fields.length === 20}
					onclick={() => (fields = [...fields, nextField++])}>Tambah obat</button
				><button class="button primary" disabled={busy || !episode}>Terbitkan resep</button>
			</div>
		</form>
	</details>{/if}
<PagedList
	path="/prescriptions"
	listKey="prescriptions"
	statuses={[
		'pending_patient_approval',
		'sent',
		'accepted',
		'preparing',
		'ready',
		'completed',
		'cancelled'
	]}
	{revision}
	empty="Belum ada resep pada daftar ini."
>
	{#snippet children(item)}<div class="wf-row-heading">
			<div>
				<h2>{item.items?.map((value) => value.medication_name).join(', ') || 'Resep'}</h2>
				<p>{item.patient_ref} · {item.pharmacy_name || 'Apotek belum dipilih'}</p>
			</div>
			<span class="health-status">{statusText[item.status || ''] || item.status}</span>
		</div>
		<p>Diterbitkan: {dateText(item.created_at)}</p>
		<button class="button secondary" onclick={() => show('prescription', item.id)}
			>Detail obat & aturan pakai</button
		>{#if !doctor && item.status === 'pending_patient_approval'}<div class="wf-actions">
				<button
					class="button primary"
					disabled={busy}
					onclick={() => {
						approving = item.id;
						pharmacy = '';
					}}>Pilih apotek</button
				><button
					class="button secondary"
					disabled={busy}
					onclick={() => run(`/prescriptions/${item.id}/approval`, { decision: 'reject' })}
					>Tolak resep</button
				>
			</div>
			{#if approving === item.id}<form onsubmit={approve}>
					<DirectoryPicker
						path="/pharmacies"
						listKey="pharmacies"
						label="Apotek tujuan"
						bind:value={pharmacy}
					/>
					<p>
						Persetujuan mengirim resep ini ke apotek pilihan Anda. Apotek tidak dapat diganti
						setelah disetujui.
					</p>
					<button class="button primary" disabled={busy || !pharmacy}>Setujui & kirim resep</button
					><button
						class="button secondary"
						type="button"
						onclick={() => {
							approving = '';
							pharmacy = '';
						}}>Batal memilih</button
					>
				</form>{/if}{/if}
		{#if item.status !== 'completed' && item.status !== 'cancelled'}
			<CancelAction path={`/prescriptions/${item.id}`} {run} {busy} />
		{/if}
		{#if item.status === 'cancelled'}
			{#if item.cancellation_note}<p>Alasan pembatalan: {item.cancellation_note}</p>{/if}
			{#if doctor}<details>
					<summary>Terbitkan resep pengganti</summary>
					<p>
						Terbitkan resep baru dengan obat dan aturan pakai yang sama. Pasien memilih apotek dan
						menyetujui kembali. Untuk mengubah obat, gunakan form Buat resep.
					</p>
					<button
						class="button primary"
						disabled={busy}
						onclick={() => {
							if (
								confirm(
									'Terbitkan resep pengganti? Tinjau obat dan aturan pakai terlebih dahulu. Pasien perlu menyetujui kembali.'
								)
							)
								void run('/prescriptions', { episode_id: item.episode_id, items: item.items });
						}}>Terbitkan resep baru untuk dipilih apoteknya</button
					>
				</details>{:else}<p>
					Hubungi dokter untuk resep pengganti. Apotek dapat dipilih kembali pada resep baru.
				</p>{/if}
		{/if}
	{/snippet}
</PagedList>
