<script lang="ts">
	import { onMount } from 'svelte';
	import { api, errorText, dateText, statusText, type Entry } from './api.js';
	let {
		kind,
		id,
		expiresAt,
		close
	}: { kind: string; id: string; expiresAt?: string; close: () => void } = $props();
	let dialog: HTMLDialogElement,
		record = $state<Entry | null>(null),
		error = $state(''),
		loading = $state(true),
		retry = $state(0);
	onMount(() => {
		dialog.showModal();
		return () => {
			record = null;
		};
	});
	$effect(() => {
		const resource = id,
			resourceKind = kind,
			expiry = expiresAt,
			refresh = retry;
		void refresh;
		const controller = new AbortController();
		record = null;
		loading = true;
		error = '';
		if (expiry && Date.parse(expiry) <= Date.now()) {
			loading = false;
			error = 'Masa akses berakhir. Tutup panel dan periksa izin terbaru.';
			return;
		}
		void api<{ result?: Entry; prescription?: Entry }>(
			`/${resourceKind === 'lab_result' ? 'lab-results' : 'prescriptions'}/${encodeURIComponent(resource)}`,
			'GET',
			undefined,
			controller.signal
		)
			.then((data) => {
				if (!controller.signal.aborted && (!expiry || Date.parse(expiry) > Date.now()))
					record = data.result || data.prescription || null;
			})
			.catch((reason) => {
				if (!controller.signal.aborted) error = errorText(reason);
			})
			.finally(() => {
				if (!controller.signal.aborted) loading = false;
			});
		return () => controller.abort();
	});
	$effect(() => {
		if (!expiresAt) return;
		const timer = setInterval(() => {
			if (Date.parse(expiresAt) <= Date.now()) {
				record = null;
				error = 'Masa akses berakhir. Tutup panel dan periksa izin terbaru.';
			}
		}, 1000);
		return () => clearInterval(timer);
	});
</script>

<dialog
	bind:this={dialog}
	class="wf-dialog"
	onclose={close}
	aria-labelledby="clinical-detail-title"
>
	<header>
		<h2 id="clinical-detail-title">
			{kind === 'lab_result' ? 'Hasil pemeriksaan' : 'Detail resep'}
		</h2>
		<button type="button" class="button secondary" onclick={() => dialog.close()}>Tutup</button>
	</header>
	{#if loading}<p role="status">Memuat detail…</p>{:else if error}<p class="wf-error" role="alert">
			{error}
		</p>
		<button class="button secondary" onclick={() => retry++}>Coba lagi</button
		>{:else if record}{#if kind === 'lab_result'}<h3>{record.test_name}</h3>
			<dl class="wf-facts">
				<div>
					<dt>Nilai</dt>
					<dd>{record.value} {record.unit}</dd>
				</div>
				<div>
					<dt>Rentang rujukan</dt>
					<dd>{record.reference_range || 'Tidak dicantumkan'}</dd>
				</div>
				<div>
					<dt>Penanda dari laboratorium</dt>
					<dd>{record.flag || 'unknown'}</dd>
				</div>
				<div>
					<dt>Catatan</dt>
					<dd>{record.note || 'Tidak ada catatan'}</dd>
				</div>
				<div>
					<dt>Dikirim</dt>
					<dd>{dateText(record.submitted_at)}</dd>
				</div>
			</dl>
			<p class="muted">Interpretasi hasil dilakukan oleh dokter yang menangani Anda.</p>{:else}<p>
				{statusText[record.status || ''] || record.status} · {record.pharmacy_name ||
					'Apotek belum dipilih'}
			</p>
			{#each record.items || [] as item, index (index)}<section class="wf-medication">
					<h3>{item.medication_name}</h3>
					<p>{item.dosage} · Jumlah: {item.quantity}</p>
					<p>{item.instructions}</p>
				</section>{/each}{/if}
		<details><summary>ID resource untuk berbagi</summary><code>{id}</code></details>{/if}
</dialog>
