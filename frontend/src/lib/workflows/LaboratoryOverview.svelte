<script lang="ts">
	import { onMount } from 'svelte';
	import { subscribeRealtime } from './realtime.js';
	import { resolve } from '$app/paths';
	import type { User } from '../auth.js';
	import WorkspaceIcon from '../WorkspaceIcon.svelte';
	import { statusText } from './api.js';
	import { emptyOverview, readOverview } from './overview.js';
	let { user }: { user: User } = $props();
	let queue = $state(emptyOverview());
	let refreshing = $state(false);
	let controller: AbortController | undefined;
	let metrics = $derived([
		{ label: 'Permintaan masuk', status: 'sent' },
		{ label: 'Diterima lab', status: 'accepted' },
		{ label: 'Sedang diperiksa', status: 'in_progress' }
	]);
	onMount(() => subscribeRealtime(() => void refresh()));
	async function refresh() {
		if (refreshing) return;
		controller?.abort();
		const current = new AbortController();
		controller = current;
		refreshing = true;
		queue.loading = true;
		const next = await readOverview(
			'/lab-requests?limit=12&offset=0',
			'lab_requests',
			current.signal
		);
		if (current.signal.aborted) return;
		queue = next;
		refreshing = false;
	}
	onMount(() => {
		void refresh();
		return () => controller?.abort();
	});
</script>

<section class="role-overview" aria-labelledby="lab-overview-title">
	<div class="patient-heading">
		<div>
			<p class="dashboard-date">{user.organization?.name || 'Laboratorium'} · {user.full_name}</p>
			<h1 id="lab-overview-title">Ringkasan laboratorium</h1>
			<p class="dashboard-description">
				Terima penugasan, proses pemeriksaan, dan serahkan hasil terstruktur.
			</p>
		</div>
		<a class="button primary" href={resolve('/dashboard/[role]?view=queue', { role: 'lab' })}
			><WorkspaceIcon name="labs" />Buka antrean pemeriksaan</a
		>
	</div>
	<div class="role-context">
		<WorkspaceIcon name="labs" />
		<p>
			<strong>Ruang kerja laboratorium.</strong> Hanya permintaan yang telah disetujui pasien dan ditujukan
			ke laboratorium Anda yang muncul. Permintaan pemeriksaan dibuat oleh dokter.
		</p>
	</div>
	<div class="overview-metrics lab-overview-metrics">
		{#each metrics as metric (metric.status)}<a
				class="overview-metric"
				href={resolve('/dashboard/[role]?view=queue', { role: 'lab' })}
				><WorkspaceIcon name="labs" /><strong
					>{queue.loading || queue.error
						? '—'
						: queue.items.filter((item) => item.status === metric.status).length}</strong
				><span>{metric.label}</span><span class="metric-arrow"><WorkspaceIcon name="arrow" /></span
				></a
			>{/each}
	</div>
	<p class="overview-limit">
		Status dari maksimal 12 penugasan terbaru, bukan total seluruh antrean. Hasil yang diserahkan
		keluar dari antrean aktif.
	</p>
	<section class="health-panel" aria-labelledby="lab-queue-preview-title">
		<div class="health-panel-heading">
			<div class="section-title">
				<WorkspaceIcon name="labs" />
				<h2 id="lab-queue-preview-title">Penugasan pemeriksaan terbaru</h2>
			</div>
			<button class="button secondary" onclick={refresh} disabled={refreshing}
				><WorkspaceIcon name="refresh" />{refreshing
					? 'Memperbarui…'
					: 'Perbarui ringkasan'}</button
			>
		</div>
		{#if queue.loading}<p class="resource-message" role="status">
				Memuat penugasan laboratorium…
			</p>{:else if queue.error}<div class="resource-message resource-error" role="alert">
				<p>{queue.error}</p>
				<button class="text-button" onclick={refresh}>Coba lagi</button>
			</div>{:else if !queue.items.length}<div class="compact-empty">
				<h3>Belum ada penugasan pemeriksaan</h3>
				<p>
					Permintaan akan muncul setelah dokter memilih lab Anda dan pasien menyetujui
					pengirimannya. Tidak ada pasien yang perlu didaftarkan oleh lab di sini.
				</p>
			</div>{:else}<ul class="role-resource-list lab-queue-preview">
				{#each queue.items as item (item.id)}<li>
						<div>
							<h3>{item.test_name}</h3>
							<p>Kode tes: {item.test_code} · Pasien: {item.patient_ref}</p>
						</div>
						<span class="health-status" class:status-progress={item.status === 'in_progress'}
							>{statusText[item.status || ''] || item.status}</span
						>
					</li>{/each}
			</ul>{/if}
	</section>
	<div class="role-overview-grid lab-workflow-guidance">
		<section class="health-panel role-access-note">
			<div class="section-title">
				<WorkspaceIcon name="labs" />
				<h2>Urutan kerja pemeriksaan</h2>
			</div>
			<ol class="lab-process-steps">
				<li>
					<strong>Terima permintaan</strong>
					<p>Konfirmasi penugasan yang baru masuk.</p>
				</li>
				<li>
					<strong>Mulai pemeriksaan</strong>
					<p>Ubah status menjadi sedang diperiksa saat proses dimulai.</p>
				</li>
				<li>
					<strong>Kirim hasil pemeriksaan</strong>
					<p>Isi nilai, satuan, rentang rujukan, dan penanda. Sistem mengembalikan tanda terima.</p>
				</li>
			</ol>
			<a class="panel-link" href={resolve('/dashboard/[role]?view=queue', { role: 'lab' })}
				>Proses penugasan di antrean<WorkspaceIcon name="arrow" /></a
			>
		</section>
		<section class="health-panel role-access-note">
			<div class="section-title">
				<WorkspaceIcon name="corrections" />
				<h2>Koreksi hasil yang diserahkan</h2>
			</div>
			<p>
				Gunakan ID hasil dari tanda terima, alasan koreksi, dan nilai pengganti lengkap. Koreksi
				ditambahkan ke riwayat; hasil awal tetap tersimpan.
			</p>
			<p>
				Laboratorium tidak membuat resep atau mengundang pasien. Tanda terima juga tidak memberi
				izin untuk membaca kembali nilai hasil.
			</p>
			<a class="panel-link" href={resolve('/dashboard/[role]?view=corrections', { role: 'lab' })}
				>Buka koreksi hasil<WorkspaceIcon name="arrow" /></a
			>
		</section>
	</div>
</section>
