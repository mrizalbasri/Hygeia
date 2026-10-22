<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import type { User } from '../auth.js';
	import { api, ApiError, errorText, menus, statusText, type Entry, type Payload } from './api.js';
	import PagedList from './PagedList.svelte';
	import Notifications from './Notifications.svelte';
	import './workflow.css';

	let { user }: { user: User } = $props();

	let menu = $derived(menus.pharmacy);
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
		completedNotice = $state<{ id: string; patient_ref?: string } | null>(null);

	// Confirmation dialog for completion
	let confirmDialog: HTMLDialogElement | null = $state(null),
		pendingCompletionItem = $state<Entry | null>(null);

	onMount(() => {
		ready = true;
	});

	$effect(() => {
		const currentView = view;
		void currentView;
		error = '';
		success = '';
	});

	async function run(path: string, body: Payload, method: 'POST' | 'PATCH' = 'PATCH') {
		if (busy) return false;
		busy = true;
		error = '';
		success = '';
		try {
			await api(path, method, body);
			revision++;
			success = 'Status resep berhasil diperbarui.';
			return true;
		} catch (reason) {
			error = errorText(reason);
			if (reason instanceof ApiError && reason.status === 409) {
				revision++;
				error += ' Status telah berubah. Periksa antrean terbaru sebelum mencoba kembali.';
			}
			return false;
		} finally {
			busy = false;
		}
	}

	function promptComplete(item: Entry) {
		pendingCompletionItem = item;
		if (confirmDialog) confirmDialog.showModal();
	}

	function closeCompletePrompt() {
		if (confirmDialog?.open) confirmDialog.close();
		pendingCompletionItem = null;
	}

	async function confirmComplete() {
		if (!pendingCompletionItem || busy) return;
		const target = pendingCompletionItem;
		closeCompletePrompt();
		const ok = await run(`/prescriptions/${target.id}/status`, { status: 'completed' }, 'PATCH');
		if (ok) {
			completedNotice = { id: target.id, patient_ref: target.patient_ref };
			success = `Penyerahan resep untuk pasien ${target.patient_ref} telah selesai.`;
		}
	}
</script>

<div class="wf-heading">
	<p class="muted">{user.organization?.name || 'Apotek'} · {user.full_name}</p>
	<h1>Ruang Apotek</h1>
	<h2>{menu.find(([id]) => id === view)?.[1] || 'Antrean Resep'}</h2>
</div>

{#if error}<p class="wf-error" role="alert">{error}</p>{/if}
{#if success}<p class="wf-success" role="status">{success}</p>{/if}
{#if busy}<p role="status">Memproses tindakan…</p>{/if}

{#if completedNotice}
	<section class="wf-form-panel" aria-labelledby="completed-title">
		<div class="wf-row-heading">
			<div>
				<p class="eyebrow muted">PENYERAHAN RESEP SELESAI</p>
				<h2 id="completed-title">Resep Berhasil Diserahkan</h2>
			</div>
			<span class="health-status">Selesai</span>
		</div>
		<p>
			Resep untuk kode pasien <strong>{completedNotice.patient_ref}</strong> (ID:
			<code>{completedNotice.id}</code>) telah berhasil diserahkan kepada pasien.
		</p>
		<p class="muted">
			Sesuai kebijakan proteksi akses resep Hygeia, resep yang telah selesai otomatis keluar dari
			antrean dan akses peninjauan apotek ditutup untuk menjaga privasi pasien. Riwayat resep tetap
			dapat dilihat oleh pasien.
		</p>
		<div class="wf-actions">
			<button type="button" class="button secondary" onclick={() => (completedNotice = null)}
				>Tutup pemberitahuan</button
			>
		</div>
	</section>
{/if}

{#key view}
	<fieldset class="wf-content" disabled={busy || !ready} aria-busy={busy || !ready}>
		{#if view === 'queue'}
			<PagedList
				path="/prescriptions"
				listKey="prescriptions"
				statuses={['sent', 'accepted', 'preparing', 'ready']}
				{revision}
				empty="Belum ada resep aktif pada antrean apotek Anda."
			>
				{#snippet children(item)}
					<div class="wf-row-heading">
						<div>
							<h2>
								{item.items?.map((med) => med.medication_name).join(', ') || 'Resep Obat'}
							</h2>
							<p>Kode pasien: <strong>{item.patient_ref}</strong></p>
						</div>
						<span class="health-status">{statusText[item.status || ''] || item.status}</span>
					</div>

					{#if item.items && item.items.length}
						{#each item.items as med, idx (idx)}
							<section class="wf-medication">
								<h3>{med.medication_name}</h3>
								<p>
									<strong>Dosis:</strong>
									{med.dosage} · <strong>Jumlah:</strong>
									{med.quantity}
								</p>
								<p><strong>Aturan pakai:</strong> {med.instructions}</p>
							</section>
						{/each}
					{/if}

					<div class="wf-actions">
						{#if item.status === 'sent'}
							<button
								type="button"
								class="button primary"
								disabled={busy}
								onclick={() =>
									run(`/prescriptions/${item.id}/status`, { status: 'accepted' }, 'PATCH')}
							>
								Terima resep
							</button>
						{:else if item.status === 'accepted'}
							<button
								type="button"
								class="button primary"
								disabled={busy}
								onclick={() =>
									run(`/prescriptions/${item.id}/status`, { status: 'preparing' }, 'PATCH')}
							>
								Mulai penyiapan obat
							</button>
						{:else if item.status === 'preparing'}
							<button
								type="button"
								class="button primary"
								disabled={busy}
								onclick={() =>
									run(`/prescriptions/${item.id}/status`, { status: 'ready' }, 'PATCH')}
							>
								Tandai siap diambil
							</button>
						{:else if item.status === 'ready'}
							<button
								type="button"
								class="button primary"
								disabled={busy}
								onclick={() => promptComplete(item)}
							>
								Selesaikan penyerahan obat
							</button>
						{/if}
					</div>
				{/snippet}
			</PagedList>
		{:else if view === 'notifications'}
			<Notifications {run} {busy} {revision} />
		{/if}
	</fieldset>
{/key}

<dialog
	bind:this={confirmDialog}
	class="wf-dialog"
	onclose={closeCompletePrompt}
	aria-labelledby="confirm-modal-title"
>
	<header>
		<h2 id="confirm-modal-title">Konfirmasi Penyerahan Obat</h2>
		<button type="button" class="button secondary" disabled={busy} onclick={closeCompletePrompt}
			>Batal</button
		>
	</header>
	{#if pendingCompletionItem}
		<p>
			Pastikan seluruh obat untuk pasien <strong>{pendingCompletionItem.patient_ref}</strong> telah diverifikasi
			dan diserahkan secara lengkap.
		</p>
		<p class="muted">
			Tindakan ini tidak dapat dibatalkan. Setelah diserahkan, resep akan keluar dari antrean dan
			akses apotek ke resep ini ditutup.
		</p>
	{/if}
	<div class="wf-actions">
		<button type="button" class="button primary" disabled={busy} onclick={confirmComplete}>
			{busy ? 'Menyimpan…' : 'Ya, selesaikan penyerahan'}
		</button>
		<button type="button" class="button secondary" disabled={busy} onclick={closeCompletePrompt}>
			Batal
		</button>
	</div>
</dialog>
