<script lang="ts">
	import { untrack } from 'svelte';
	import { resolve } from '$app/paths';
	import type { User } from '../auth.js';
	import WorkspaceIcon from '../WorkspaceIcon.svelte';
	import { statusText } from './api.js';
	import { emptyOverview, readOverview } from './overview.js';
	import { useNotifications } from './notification-state.svelte.js';
	const notifications = useNotifications();
	let { user }: { user: User } = $props();
	let episodes = $state(emptyOverview()),
		labs = $state(emptyOverview()),
		prescriptions = $state(emptyOverview());
	let refreshing = $state(false);
	let controller: AbortController | undefined;
	let metrics = $derived([
		{
			label: 'Perawatan aktif',
			value:
				episodes.loading || episodes.error
					? '—'
					: episodes.items.filter((item) => item.status === 'active').length,
			icon: 'episodes',
			view: 'episodes'
		},
		{
			label: 'Undangan tertunda',
			value:
				episodes.loading || episodes.error
					? '—'
					: episodes.items.filter((item) => item.status === 'pending').length,
			icon: 'consents',
			view: 'episodes'
		},
		{
			label: 'Hasil lab tercatat',
			value:
				labs.loading || labs.error
					? '—'
					: labs.items.filter((item) => item.status === 'completed' && item.result_id).length,
			icon: 'labs',
			view: 'labs'
		},
		{
			label: 'Resep terbaru',
			value: prescriptions.loading || prescriptions.error ? '—' : prescriptions.items.length,
			icon: 'prescriptions',
			view: 'prescriptions'
		}
	]);
	async function refresh() {
		if (refreshing) return;
		controller?.abort();
		const current = new AbortController();
		controller = current;
		refreshing = true;
		episodes.loading = labs.loading = prescriptions.loading = true;
		const [nextEpisodes, nextLabs, nextPrescriptions] = await Promise.all([
			readOverview('/episodes?limit=6&offset=0', 'episodes', current.signal),
			readOverview('/lab-requests?limit=6&offset=0', 'lab_requests', current.signal),
			readOverview('/prescriptions?limit=6&offset=0', 'prescriptions', current.signal)
		]);
		if (current.signal.aborted) return;
		episodes = nextEpisodes;
		labs = nextLabs;
		prescriptions = nextPrescriptions;
		refreshing = false;
	}
	$effect(() => {
		const revision = notifications.revision;
		void revision;
		untrack(() => {
			refreshing = false;
			void refresh();
		});
		return () => controller?.abort();
	});
</script>

<section class="role-overview" aria-labelledby="doctor-overview-title">
	<div class="patient-heading">
		<div>
			<p class="dashboard-date">{user.organization?.name || 'Klinik'} · {user.full_name}</p>
			<h1 id="doctor-overview-title">Ringkasan dokter</h1>
			<p class="dashboard-description">
				Kelola perawatan pasien, tinjau pemeriksaan, dan lanjutkan resep.
			</p>
		</div>
		<button class="button secondary" onclick={refresh} disabled={refreshing}
			><WorkspaceIcon name="refresh" />{refreshing ? 'Memperbarui…' : 'Perbarui ringkasan'}</button
		>
	</div>
	<div class="role-context">
		<WorkspaceIcon name="episodes" />
		<p>
			<strong>Ruang kerja klinis.</strong> Anda mengundang pasien, mengajukan pemeriksaan, meninjau hasil
			sesuai izin, dan membuat resep. Laboratorium memproses pemeriksaan serta mengirim hasilnya.
		</p>
	</div>
	<div class="overview-metrics">
		{#each metrics as metric (metric.label)}<a
				class="overview-metric"
				href={resolve(`/dashboard/[role]?view=${metric.view}`, { role: 'doctor' })}
				><WorkspaceIcon name={metric.icon} /><strong>{metric.value}</strong><span
					>{metric.label}</span
				><span class="metric-arrow"><WorkspaceIcon name="arrow" /></span></a
			>{/each}
	</div>
	<p class="overview-limit">
		Dihitung dari maksimal 6 entri terbaru per kategori, bukan total seluruh pasien atau
		pemeriksaan.
	</p>
	<div class="role-quick-links" aria-label="Pekerjaan dokter">
		<a href={resolve('/dashboard/[role]?view=episodes', { role: 'doctor' })}
			><WorkspaceIcon name="episodes" /><span>Undangan & perawatan pasien</span><WorkspaceIcon
				name="arrow"
			/></a
		><a href={resolve('/dashboard/[role]?view=labs', { role: 'doctor' })}
			><WorkspaceIcon name="labs" /><span>Permintaan & hasil lab</span><WorkspaceIcon
				name="arrow"
			/></a
		><a href={resolve('/dashboard/[role]?view=prescriptions', { role: 'doctor' })}
			><WorkspaceIcon name="prescriptions" /><span>Penulisan resep</span><WorkspaceIcon
				name="arrow"
			/></a
		>
	</div>
	<div class="role-overview-grid">
		<section class="health-panel" aria-labelledby="doctor-patients-title">
			<div class="health-panel-heading">
				<div class="section-title">
					<WorkspaceIcon name="episodes" />
					<h2 id="doctor-patients-title">Perawatan pasien terbaru</h2>
				</div>
				<a class="panel-link" href={resolve('/dashboard/[role]?view=episodes', { role: 'doctor' })}
					>Lihat semua<WorkspaceIcon name="arrow" /></a
				>
			</div>
			{#if episodes.loading}<p class="resource-message" role="status">
					Memuat perawatan…
				</p>{:else if episodes.error}<div class="resource-message resource-error" role="alert">
					<p>{episodes.error}</p>
					<button class="text-button" onclick={refresh}>Coba lagi</button>
				</div>{:else if !episodes.items.length}<div class="compact-empty">
					<h3>Belum ada perawatan pasien</h3>
					<p>
						Mulai dengan mengundang pasien menggunakan kode pasien. Pasien perlu menerima undangan
						sebelum perawatan aktif.
					</p>
					<a
						class="panel-link"
						href={resolve('/dashboard/[role]?view=episodes', { role: 'doctor' })}
						>Buka undangan pasien<WorkspaceIcon name="arrow" /></a
					>
				</div>{:else}<ul class="role-resource-list">
					{#each episodes.items as item (item.id)}<li>
							<div>
								<h3>{item.patient_ref || 'Pasien'}</h3>
								<p>{item.organization_name || user.organization?.name}</p>
							</div>
							<span
								class="health-status"
								class:status-active={item.status === 'active'}
								class:status-pending={item.status === 'pending'}
								>{statusText[item.status || ''] || item.status}</span
							>
						</li>{/each}
				</ul>{/if}
		</section>
		<section class="health-panel" aria-labelledby="doctor-labs-title">
			<div class="health-panel-heading">
				<div class="section-title">
					<WorkspaceIcon name="labs" />
					<h2 id="doctor-labs-title">Permintaan & hasil lab</h2>
				</div>
				<a class="panel-link" href={resolve('/dashboard/[role]?view=labs', { role: 'doctor' })}
					>Buka pemeriksaan<WorkspaceIcon name="arrow" /></a
				>
			</div>
			{#if labs.loading}<p class="resource-message" role="status">
					Memuat pemeriksaan…
				</p>{:else if labs.error}<div class="resource-message resource-error" role="alert">
					<p>{labs.error}</p>
					<button class="text-button" onclick={refresh}>Coba lagi</button>
				</div>{:else if !labs.items.length}<div class="compact-empty">
					<h3>Belum ada permintaan lab</h3>
					<p>
						Ajukan pemeriksaan dalam perawatan aktif. Permintaan diteruskan ke laboratorium setelah
						disetujui pasien.
					</p>
				</div>{:else}<ul class="role-resource-list">
					{#each labs.items as item (item.id)}<li>
							<div>
								<h3>{item.test_name}</h3>
								<p>{item.patient_ref} · {item.lab_name}</p>
							</div>
							<span class="health-status" class:status-active={item.status === 'completed'}
								>{statusText[item.status || ''] || item.status}</span
							>
						</li>{/each}
				</ul>{/if}
		</section>
		<section class="health-panel" aria-labelledby="doctor-prescriptions-title">
			<div class="health-panel-heading">
				<div class="section-title">
					<WorkspaceIcon name="prescriptions" />
					<h2 id="doctor-prescriptions-title">Resep terbaru</h2>
				</div>
				<a
					class="panel-link"
					href={resolve('/dashboard/[role]?view=prescriptions', { role: 'doctor' })}
					>Kelola resep<WorkspaceIcon name="arrow" /></a
				>
			</div>
			{#if prescriptions.loading}<p class="resource-message" role="status">
					Memuat resep…
				</p>{:else if prescriptions.error}<div class="resource-message resource-error" role="alert">
					<p>{prescriptions.error}</p>
					<button class="text-button" onclick={refresh}>Coba lagi</button>
				</div>{:else if !prescriptions.items.length}<div class="compact-empty">
					<h3>Belum ada resep</h3>
					<p>
						Resep dibuat dalam perawatan aktif. Pasien memilih apotek dan menyetujui pengirimannya.
					</p>
				</div>{:else}<ul class="role-resource-list">
					{#each prescriptions.items as item (item.id)}<li>
							<div>
								<h3>{item.patient_ref}</h3>
								<p>
									{item.items?.map((medicine) => medicine.medication_name).join(', ') || 'Resep'} · {item.pharmacy_name ||
										'Apotek belum dipilih'}
								</p>
							</div>
							<span class="health-status">{statusText[item.status || ''] || item.status}</span>
						</li>{/each}
				</ul>{/if}
		</section>
		<section class="health-panel role-access-note">
			<div class="section-title">
				<WorkspaceIcon name="audit" />
				<h2>Akses data pasien</h2>
			</div>
			<p>
				Hasil ditinjau melalui pemeriksaan dengan izin yang diperiksa server. Untuk data di luar
				perawatan, gunakan permintaan akses atau CarePass. Akses darurat memerlukan alasan dan
				tercatat untuk ditinjau.
			</p>
			<a class="panel-link" href={resolve('/dashboard/[role]?view=consents', { role: 'doctor' })}
				>Kelola permintaan akses<WorkspaceIcon name="arrow" /></a
			>
		</section>
	</div>
</section>
