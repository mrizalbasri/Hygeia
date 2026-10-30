<script lang="ts">
	import { onMount } from 'svelte';
	import { subscribeRealtime } from './realtime.js';
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
		retry = $state(0),
		historyOpen = $state(false),
		lastResourceId = '';
	let latestCorrection = $derived(record?.corrections?.at(-1));
	let currentResult = $derived(latestCorrection || record);
	onMount(() => subscribeRealtime(() => retry++));
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
		if (lastResourceId !== resource) {
			lastResourceId = resource;
			record = null;
			loading = true;
			historyOpen = false;
		}
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
		<div class="wf-dialog-actions">
			<button
				type="button"
				class="button secondary no-print"
				disabled={loading || !record || !!error}
				onclick={() => window.print()}>Cetak / PDF</button
			>
			<button type="button" class="button secondary no-print" onclick={() => dialog.close()}
				>Tutup</button
			>
		</div>
	</header>
	{#if loading}<p role="status">Memuat detail…</p>{:else if error}<p class="wf-error" role="alert">
			{error}
		</p>
		<button class="button secondary" onclick={() => retry++}>Coba lagi</button
		>{:else if record}{#if kind === 'lab_result'}<h3>{record.test_name}</h3>
			<dl class="wf-facts">
				<div>
					<dt>{latestCorrection ? 'Nilai terbaru setelah koreksi' : 'Nilai'}</dt>
					<dd>{currentResult?.value} {currentResult?.unit}</dd>
				</div>
				<div>
					<dt>Rentang rujukan</dt>
					<dd>{currentResult?.reference_range || 'Tidak dicantumkan'}</dd>
				</div>
				<div>
					<dt>Penanda dari laboratorium</dt>
					<dd>{currentResult?.flag || 'unknown'}</dd>
				</div>
				<div>
					<dt>Catatan</dt>
					<dd>{currentResult?.note || 'Tidak ada catatan'}</dd>
				</div>
				<div>
					<dt>Dikirim</dt>
					<dd>{dateText(latestCorrection?.corrected_at || record.submitted_at)}</dd>
				</div>
			</dl>
			{#if latestCorrection}<p role="status">Hasil ini telah dikoreksi oleh laboratorium.</p>
				{#if latestCorrection.reason}<p>Alasan koreksi terbaru: {latestCorrection.reason}</p>{/if}
				<details bind:open={historyOpen}>
					<summary>Riwayat hasil dan koreksi</summary>
					<section class="wf-medication">
						<h3>Hasil awal · {dateText(record.submitted_at)}</h3>
						<p>{record.value} {record.unit} · {record.flag}</p>
						<p>Rentang rujukan: {record.reference_range || 'Tidak dicantumkan'}</p>
						{#if record.note}<p>Catatan: {record.note}</p>{/if}
					</section>
					{#each [...(record.corrections || [])].reverse() as correction (correction.id)}
						<section class="wf-medication">
							<h3>Koreksi hasil · {dateText(correction.corrected_at)}</h3>
							<p>{correction.value} {correction.unit} · {correction.flag}</p>
							<p>Rentang rujukan: {correction.reference_range || 'Tidak dicantumkan'}</p>
							{#if correction.reason}<p>Alasan: {correction.reason}</p>{/if}
							{#if correction.note}<p>Catatan: {correction.note}</p>{/if}
						</section>
					{/each}
				</details>
			{/if}
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
