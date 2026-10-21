<script lang="ts">
	import PagedList from './PagedList.svelte';
	import { formText, uuidPattern, statusText, dateText, type Run } from './api.js';
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
	let durations = $state<Record<string, string>>({});
	async function request(event: SubmitEvent) {
		event.preventDefault();
		const form = event.currentTarget as HTMLFormElement,
			data = new FormData(form);
		if (
			await run('/consents', {
				patient_ref: formText(data, 'patient_ref'),
				resource_type: 'lab_result',
				resource_id: formText(data, 'resource_id'),
				purpose: formText(data, 'purpose')
			})
		)
			form.reset();
	}
</script>

{#if doctor}<details class="wf-form-panel">
		<summary>Minta akses hasil lab</summary>
		<form onsubmit={request}>
			<p>
				Pasien memberikan kode pasien dan ID hasil lab yang ingin dibagikan. Permintaan ini tidak
				membuka seluruh riwayat.
			</p>
			<label>Kode pasien<input name="patient_ref" required maxlength="64" /></label><label
				>ID hasil lab<input
					name="resource_id"
					required
					pattern={uuidPattern}
					autocomplete="off"
				/></label
			><label>Tujuan akses<textarea name="purpose" required maxlength="1000"></textarea></label
			><button class="button primary" disabled={busy}>Kirim permintaan akses</button>
		</form>
	</details>{/if}
<PagedList
	path="/consents"
	listKey="consents"
	statuses={['pending', 'active', 'denied', 'revoked', 'expired']}
	{revision}
	empty="Belum ada permintaan izin dokter."
>
	{#snippet children(item)}<div class="wf-row-heading">
			<div>
				<h2>{doctor ? item.patient_ref : item.grantee_name}</h2>
				<p>{item.grantee_org_name} · {item.test_name || 'Satu hasil laboratorium'}</p>
			</div>
			<span class="health-status">{statusText[item.status || ''] || item.status}</span>
		</div>
		<p>Tujuan: {item.purpose}</p>
		<p>Masa akses berakhir: {dateText(item.expires_at)}</p>
		<details>
			<summary>Data yang diminta</summary>
			<p>Hanya membaca satu hasil laboratorium.</p>
			<code>{item.resource_id}</code>
		</details>
		{#if !doctor && item.status === 'pending'}<label
				>Durasi izin<select
					value={durations[item.id] || '1'}
					onchange={(event) => (durations[item.id] = event.currentTarget.value)}
					><option value="1">1 jam</option><option value="24">24 jam</option><option value="168"
						>7 hari</option
					></select
				></label
			>
			<div class="wf-actions">
				<button
					class="button primary"
					disabled={busy}
					onclick={() =>
						run(
							`/consents/${item.id}`,
							{ action: 'approve', duration_hours: Number(durations[item.id] || 1) },
							'PATCH'
						)}>Izinkan hasil ini</button
				><button
					class="button secondary"
					disabled={busy}
					onclick={() => run(`/consents/${item.id}`, { action: 'deny' }, 'PATCH')}
					>Tolak akses</button
				>
			</div>{/if}{#if item.status === 'active'}<div class="wf-actions">
				{#if !doctor}<button
						class="button secondary"
						disabled={busy}
						onclick={() => {
							if (
								confirm(
									'Cabut izin dokter untuk hasil ini? Akses melalui izin ini ditutup. Hubungan perawatan aktif yang terpisah tetap berlaku.'
								)
							)
								void run(`/consents/${item.id}`, { action: 'revoke' }, 'PATCH');
						}}>Cabut izin</button
					>{/if}{#if item.resource_id}<button
						class="button secondary"
						onclick={() =>
							show('lab_result', item.resource_id!, doctor ? item.expires_at : undefined)}
						>Buka hasil yang diizinkan</button
					>{/if}
			</div>{/if}{/snippet}
</PagedList>
