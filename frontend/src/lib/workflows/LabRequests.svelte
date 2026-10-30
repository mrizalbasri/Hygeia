<script lang="ts">
	import PagedList from './PagedList.svelte';
	import DirectoryPicker from './DirectoryPicker.svelte';
	import CancelAction from './CancelAction.svelte';
	import { formText, statusText, dateText, type Run } from './api.js';
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
	let episode = $state(''),
		lab = $state('');
	let replacementLabs = $state<Record<string, string>>({});
	async function create(event: SubmitEvent) {
		event.preventDefault();
		if (!episode || !lab) return;
		const form = event.currentTarget as HTMLFormElement,
			data = new FormData(form);
		if (
			await run('/lab-requests', {
				episode_id: episode,
				lab_org_id: lab,
				test_code: formText(data, 'test_code'),
				test_name: formText(data, 'test_name')
			})
		) {
			form.reset();
			episode = '';
			lab = '';
		}
	}
</script>

{#if doctor}<details class="wf-form-panel">
		<summary>Ajukan pemeriksaan lab</summary>
		<form onsubmit={create}>
			<DirectoryPicker
				path="/episodes?status=active"
				listKey="episodes"
				label="Perawatan aktif"
				bind:value={episode}
			/><DirectoryPicker
				path="/laboratories"
				listKey="laboratories"
				label="Laboratorium"
				bind:value={lab}
			/><label
				>Kode tes<input
					name="test_code"
					required
					maxlength="64"
					pattern="[A-Za-z0-9][A-Za-z0-9_.-]*"
				/></label
			><label>Nama pemeriksaan<input name="test_name" required maxlength="200" /></label>
			<p>Permintaan diteruskan ke laboratorium setelah pasien menyetujui.</p>
			<button class="button primary" disabled={busy || !episode || !lab}
				>Kirim permintaan lab</button
			>
		</form>
	</details>{/if}
<PagedList
	path="/lab-requests"
	listKey="lab_requests"
	statuses={[
		'pending_patient_approval',
		'sent',
		'accepted',
		'in_progress',
		'completed',
		'cancelled'
	]}
	{revision}
	empty="Belum ada permintaan pemeriksaan pada daftar ini."
>
	{#snippet children(item)}<div class="wf-row-heading">
			<div>
				<h2>{item.test_name}</h2>
				<p>{item.lab_name} · {item.patient_ref}</p>
			</div>
			<span class="health-status">{statusText[item.status || ''] || item.status}</span>
		</div>
		<p>Diajukan: {dateText(item.created_at)}</p>
		<div class="wf-actions">
			{#if !doctor && item.status === 'pending_patient_approval'}<button
					class="button primary"
					disabled={busy}
					onclick={() => run(`/lab-requests/${item.id}/approval`, { decision: 'approve' })}
					>Setujui pengiriman ke lab</button
				><button
					class="button secondary"
					disabled={busy}
					onclick={() => run(`/lab-requests/${item.id}/approval`, { decision: 'reject' })}
					>Tolak pemeriksaan</button
				>{/if}{#if item.result_id}<button
					class="button secondary"
					onclick={() => show('lab_result', item.result_id!)}>Buka hasil pemeriksaan</button
				>{/if}
		</div>
		{#if item.status !== 'completed' && item.status !== 'cancelled'}
			<CancelAction path={`/lab-requests/${item.id}`} {run} {busy} />
		{/if}
		{#if item.status === 'cancelled'}<p class="muted">
				Permintaan dibatalkan; pemeriksaan ini tidak dapat dilanjutkan.
			</p>
			{#if item.cancellation_note}<p>Alasan pembatalan: {item.cancellation_note}</p>{/if}
			{#if doctor}<details>
					<summary>Ajukan pemeriksaan pengganti</summary>
					<form
						onsubmit={async (event) => {
							event.preventDefault();
							if (!replacementLabs[item.id]) return;
							if (
								await run('/lab-requests', {
									episode_id: item.episode_id,
									lab_org_id: replacementLabs[item.id],
									test_code: item.test_code,
									test_name: item.test_name
								})
							)
								replacementLabs[item.id] = '';
						}}
					>
						<DirectoryPicker
							path="/laboratories"
							listKey="laboratories"
							label="Laboratorium pengganti"
							bind:value={
								() => replacementLabs[item.id] || '', (value) => (replacementLabs[item.id] = value)
							}
						/>
						<p>
							Permintaan baru untuk {item.test_name}. Pasien perlu menyetujui pengiriman kembali.
						</p>
						<button class="button primary" disabled={busy || !replacementLabs[item.id]}
							>Kirim pemeriksaan pengganti</button
						>
					</form>
				</details>{:else}<p>
					Hubungi dokter untuk permintaan baru jika masih memerlukan pemeriksaan.
				</p>{/if}
		{/if}{/snippet}
</PagedList>
