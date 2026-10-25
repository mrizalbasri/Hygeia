<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import type { User } from '../auth.js';
	import {
		api,
		ApiError,
		dateText,
		errorText,
		menus,
		statusText,
		type Entry,
		type Payload,
		type ResultReceipt
	} from './api.js';
	import PagedList from './PagedList.svelte';
	import Notifications from './Notifications.svelte';
	import LabCorrections from './LabCorrections.svelte';
	import LaboratoryOverview from './LaboratoryOverview.svelte';
	import './workflow.css';

	let { user }: { user: User } = $props();

	let menu = $derived(menus.lab);
	let view = $derived(
		menu.some(([id]) => id === page.url.searchParams.get('view'))
			? page.url.searchParams.get('view')!
			: menu[0][0]
	);

	let busy = $state(false),
		ready = $state(false),
		error = $state(''),
		success = $state(''),
		revision = $state(0),
		receipt = $state<ResultReceipt | null>(null);

	// Result submission dialog state
	let dialog: HTMLDialogElement | null = $state(null),
		activeRequest = $state<Entry | null>(null),
		modalError = $state(''),
		formValue = $state(''),
		formUnit = $state(''),
		formReferenceRange = $state(''),
		formFlag = $state<'normal' | 'high' | 'low' | 'abnormal' | 'unknown'>('normal'),
		formNote = $state('');

	onMount(() => {
		ready = true;
	});

	$effect(() => {
		const currentView = view;
		void currentView;
		error = '';
		success = '';
		modalError = '';
	});

	async function run(path: string, body: Payload, method: 'POST' | 'PATCH' = 'POST') {
		if (busy) return false;
		busy = true;
		error = '';
		success = '';
		try {
			await api(path, method, body);
			revision++;
			success = 'Status pemeriksaan berhasil diperbarui.';
			return true;
		} catch (reason) {
			error = errorText(reason);
			if (reason instanceof ApiError && reason.status === 403) {
				revision++;
				closeResultForm();
			}
			if (reason instanceof ApiError && reason.status === 409) {
				revision++;
				error += ' Status telah berubah. Periksa antrean terbaru sebelum mencoba kembali.';
			}
			return false;
		} finally {
			busy = false;
		}
	}

	function openResultForm(item: Entry) {
		activeRequest = item;
		modalError = '';
		formValue = '';
		formUnit = '';
		formReferenceRange = '';
		formFlag = 'normal';
		formNote = '';
		if (dialog) dialog.showModal();
	}

	function closeResultForm() {
		if (dialog?.open) dialog.close();
		activeRequest = null;
		modalError = '';
		formValue = '';
		formUnit = '';
		formReferenceRange = '';
		formFlag = 'normal';
		formNote = '';
	}

	async function submitResult(event: SubmitEvent) {
		event.preventDefault();
		if (!activeRequest || busy) return;
		const val = formValue.trim();
		if (!val) {
			modalError = 'Nilai pemeriksaan wajib diisi.';
			return;
		}
		busy = true;
		modalError = '';
		error = '';
		success = '';
		try {
			const res = await api<{ result: ResultReceipt }>(
				`/lab-requests/${activeRequest.id}/result`,
				'POST',
				{
					value: val,
					unit: formUnit.trim(),
					reference_range: formReferenceRange.trim(),
					flag: formFlag,
					note: formNote.trim()
				}
			);
			receipt = res.result;
			success = 'Hasil pemeriksaan berhasil diserahkan ke sistem.';
			revision++;
			closeResultForm();
		} catch (reason) {
			if (reason instanceof ApiError && reason.status === 403) {
				error = errorText(reason);
				revision++;
				closeResultForm();
			} else if (reason instanceof ApiError && reason.status === 409) {
				revision++;
				modalError = errorText(reason) + ' Permintaan ini telah selesai atau dibatalkan.';
			} else {
				modalError = errorText(reason);
			}
		} finally {
			busy = false;
		}
	}
</script>

{#if view === 'overview'}<LaboratoryOverview {user} />{:else}<div class="wf-heading">
		<p class="muted">{user.organization?.name || 'Laboratorium'} · {user.full_name}</p>
		<h1>Ruang Laboratorium</h1>
		<h2>{menu.find(([id]) => id === view)?.[1] || 'Permintaan Masuk'}</h2>
	</div>
{/if}

{#if error}<p class="wf-error" role="alert">{error}</p>{/if}
{#if success}<p class="wf-success" role="status">{success}</p>{/if}
{#if busy}<p role="status">Memproses tindakan…</p>{/if}

{#if receipt}
	<section class="wf-form-panel" aria-labelledby="receipt-title">
		<div class="wf-row-heading">
			<div>
				<p class="eyebrow muted">BUKTI PENYERAHAN SELESAI</p>
				<h2 id="receipt-title">Tanda Terima Hasil Pemeriksaan</h2>
			</div>
			<span class="health-status">Selesai</span>
		</div>
		<dl class="wf-facts">
			<div>
				<dt>ID Tanda Terima</dt>
				<dd><code>{receipt.id}</code></dd>
			</div>
			<div>
				<dt>Waktu Penyerahan</dt>
				<dd>{dateText(receipt.submitted_at)}</dd>
			</div>
		</dl>
		<p class="muted">
			Permintaan telah selesai dan keluar dari antrean aktif. Berdasarkan kebijakan privasi dan
			audit Hygeia, hasil yang telah diserahkan bersifat tetap (immutable) dan tidak dapat dibaca
			kembali oleh laboratorium. Pasien dan dokter episode dapat mengakses hasil melalui hak akses
			masing-masing.
		</p>
		<div class="wf-actions">
			<a
				class="button primary"
				href={resolve(
					`/dashboard/[role]?view=corrections&result_id=${encodeURIComponent(receipt.id)}`,
					{ role: 'lab' }
				)}>Koreksi hasil ini</a
			>
			<button type="button" class="button secondary" onclick={() => (receipt = null)}
				>Tutup tanda terima</button
			>
		</div>
	</section>
{/if}

{#key view}
	<fieldset class="wf-content" disabled={busy || !ready} aria-busy={busy || !ready}>
		{#if view === 'queue'}
			<PagedList
				path="/lab-requests"
				listKey="lab_requests"
				statuses={['sent', 'accepted', 'in_progress']}
				{revision}
				empty="Belum ada permintaan pemeriksaan pada antrean laboratorium Anda."
			>
				{#snippet children(item)}
					<div class="wf-row-heading">
						<div>
							<h2>{item.test_name}</h2>
							<p>
								Kode tes: <code>{item.test_code}</code> · Kode pasien:
								<strong>{item.patient_ref}</strong>
							</p>
						</div>
						<span class="health-status">{statusText[item.status || ''] || item.status}</span>
					</div>
					<div class="wf-actions">
						{#if item.status === 'sent'}
							<button
								type="button"
								class="button primary"
								disabled={busy}
								onclick={() =>
									run(`/lab-requests/${item.id}/status`, { status: 'accepted' }, 'PATCH')}
							>
								Terima permintaan
							</button>
						{:else if item.status === 'accepted'}
							<button
								type="button"
								class="button primary"
								disabled={busy}
								onclick={() =>
									run(`/lab-requests/${item.id}/status`, { status: 'in_progress' }, 'PATCH')}
							>
								Mulai pemeriksaan
							</button>
						{:else if item.status === 'in_progress'}
							<button
								type="button"
								class="button primary"
								disabled={busy}
								onclick={() => openResultForm(item)}
							>
								Kirim hasil pemeriksaan
							</button>
						{/if}
					</div>
				{/snippet}
			</PagedList>
		{:else if view === 'corrections'}
			<LabCorrections initialResultID={page.url.searchParams.get('result_id') || ''} />
		{:else if view === 'notifications'}
			<Notifications {run} {busy} {revision} />
		{/if}
	</fieldset>
{/key}

<dialog
	bind:this={dialog}
	class="wf-dialog"
	onclose={closeResultForm}
	aria-labelledby="result-modal-title"
>
	<header>
		<h2 id="result-modal-title">Kirim Hasil Pemeriksaan</h2>
		<button type="button" class="button secondary" disabled={busy} onclick={closeResultForm}
			>Tutup</button
		>
	</header>
	{#if activeRequest}
		<p class="muted">
			Pemeriksaan: <strong>{activeRequest.test_name}</strong> ({activeRequest.test_code}) · Pasien:
			<strong>{activeRequest.patient_ref}</strong>
		</p>
	{/if}
	{#if modalError}
		<p class="wf-error" role="alert">{modalError}</p>
	{/if}
	<form onsubmit={submitResult}>
		<label>
			Nilai hasil pemeriksaan (wajib, maks 500 karakter)
			<input
				name="value"
				required
				maxlength="500"
				placeholder="Contoh: 110 atau Negatif"
				bind:value={formValue}
				disabled={busy}
			/>
		</label>
		<label>
			Satuan (opsional, maks 100 karakter)
			<input
				name="unit"
				maxlength="100"
				placeholder="Contoh: mg/dL, mmol/L"
				bind:value={formUnit}
				disabled={busy}
			/>
		</label>
		<label>
			Rentang rujukan (opsional, maks 200 karakter)
			<input
				name="reference_range"
				maxlength="200"
				placeholder="Contoh: 70–99"
				bind:value={formReferenceRange}
				disabled={busy}
			/>
		</label>
		<label>
			Penanda laboratorium (Flag)
			<select name="flag" required bind:value={formFlag} disabled={busy}>
				<option value="normal">Normal</option>
				<option value="high">Tinggi (High)</option>
				<option value="low">Rendah (Low)</option>
				<option value="abnormal">Abnormal</option>
				<option value="unknown">Tidak Diketahui (Unknown)</option>
			</select>
		</label>
		<label>
			Catatan teknis laboratorium (opsional, maks 2000 karakter)
			<textarea
				name="note"
				maxlength="2000"
				placeholder="Catatan teknis kondisi spesimen atau pengujian"
				bind:value={formNote}
				disabled={busy}></textarea>
		</label>
		<p class="muted">
			Setelah diserahkan, hasil bersifat tetap dan tidak dapat disunting kembali oleh laboratorium.
		</p>
		<div class="wf-actions">
			<button type="submit" class="button primary" disabled={busy || !formValue.trim()}>
				{busy ? 'Menyerahkan hasil…' : 'Serahkan hasil pemeriksaan'}
			</button>
			<button type="button" class="button secondary" disabled={busy} onclick={closeResultForm}>
				Batal
			</button>
		</div>
	</form>
</dialog>
