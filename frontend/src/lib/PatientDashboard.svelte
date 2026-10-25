<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import WorkspaceIcon from './WorkspaceIcon.svelte';
	import type { User } from './auth.js';
	import {
		readPatientList,
		statuses,
		type Resource,
		type Episode,
		type LabRequest,
		type Prescription,
		type TimelineEvent
	} from './patient-dashboard.js';
	let { user }: { user: User } = $props();
	let episodes = $state<Resource<Episode>>({ items: [], loading: true, error: '' });
	let labs = $state<Resource<LabRequest>>({ items: [], loading: true, error: '' });
	let prescriptions = $state<Resource<Prescription>>({ items: [], loading: true, error: '' });
	let timeline = $state<Resource<TimelineEvent>>({ items: [], loading: true, error: '' });
	let filter = $state('all');
	let refreshing = $state(false);
	let today = $state('');
	let filteredEvents = $derived(
		timeline.items.filter((event) => filter === 'all' || event.kind === filter)
	);
	let pendingItems = $derived([
		...episodes.items
			.filter((item) => item.status === 'pending')
			.map((item) => ({
				id: item.id,
				title: `Undangan dari ${item.doctor_name}`,
				text: 'Tinjau hubungan perawatan dengan dokter.',
				view: 'episodes',
				icon: 'episodes'
			})),
		...labs.items
			.filter((item) => item.status === 'pending_patient_approval')
			.map((item) => ({
				id: item.id,
				title: item.test_name,
				text: 'Tinjau pengiriman pemeriksaan ke laboratorium.',
				view: 'labs',
				icon: 'labs'
			})),
		...prescriptions.items
			.filter((item) => item.status === 'pending_patient_approval')
			.map((item) => ({
				id: item.id,
				title: 'Persetujuan resep',
				text: 'Pilih apotek dan tinjau pengiriman resep.',
				view: 'prescriptions',
				icon: 'prescriptions'
			}))
	]);
	let pendingUnavailable = $derived(
		episodes.loading ||
			labs.loading ||
			prescriptions.loading ||
			!!episodes.error ||
			!!labs.error ||
			!!prescriptions.error
	);
	let metrics = $derived([
		{
			label: 'Perawatan aktif',
			value:
				episodes.loading || episodes.error
					? '—'
					: episodes.items.filter((item) => item.status === 'active').length,
			view: 'episodes',
			icon: 'episodes'
		},
		{
			label: 'Perlu persetujuan',
			value: pendingUnavailable ? '—' : pendingItems.length,
			view: 'consents',
			icon: 'consents'
		},
		{
			label: 'Lab dalam proses',
			value:
				labs.loading || labs.error
					? '—'
					: labs.items.filter((item) => ['sent', 'accepted', 'in_progress'].includes(item.status))
							.length,
			view: 'labs',
			icon: 'labs'
		},
		{
			label: 'Resep siap diambil',
			value:
				prescriptions.loading || prescriptions.error
					? '—'
					: prescriptions.items.filter((item) => item.status === 'ready').length,
			view: 'prescriptions',
			icon: 'prescriptions'
		}
	]);
	const kinds: Record<string, string> = {
		episode: 'Perawatan',
		laboratory: 'Pemeriksaan',
		prescription: 'Resep',
		consent: 'Persetujuan'
	};
	const date = (value: string) =>
		Number.isFinite(Date.parse(value))
			? new Intl.DateTimeFormat('id-ID', {
					day: 'numeric',
					month: 'short',
					hour: '2-digit',
					minute: '2-digit',
					timeZone: 'Asia/Jakarta'
				}).format(new Date(value))
			: 'Waktu tidak tersedia';
	async function refresh() {
		if (refreshing) return;
		refreshing = true;
		episodes.loading = labs.loading = prescriptions.loading = timeline.loading = true;
		try {
			await Promise.all([
				readPatientList<Episode>('episodes?limit=6', 'episodes').then(
					(value) => (episodes = value)
				),
				readPatientList<LabRequest>('lab-requests?limit=6', 'lab_requests').then(
					(value) => (labs = value)
				),
				readPatientList<Prescription>('prescriptions?limit=6', 'prescriptions').then(
					(value) => (prescriptions = value)
				),
				readPatientList<TimelineEvent>('patient/timeline?limit=8', 'events').then(
					(value) => (timeline = value)
				)
			]);
		} finally {
			refreshing = false;
		}
	}
	onMount(() => {
		today = new Intl.DateTimeFormat('id-ID', {
			weekday: 'long',
			day: 'numeric',
			month: 'long',
			year: 'numeric',
			timeZone: 'Asia/Jakarta'
		}).format(new Date());
		void refresh();
	});
</script>

<div class="patient-heading">
	<div>
		<p class="dashboard-date">{today || 'Ringkasan perawatan Anda'}</p>
		<h1>Halo, {user.full_name}.</h1>
		<p class="dashboard-description">Ini perkembangan perawatan Anda. Semua dalam satu tempat.</p>
	</div>
	<button class="button secondary" onclick={refresh} disabled={refreshing}
		><WorkspaceIcon name="refresh" />{refreshing ? 'Memperbarui…' : 'Perbarui data'}</button
	>
</div>

<div class="overview-metrics" aria-label="Ringkasan data terbaru">
	{#each metrics as metric (metric.label)}<a
			class="overview-metric"
			href={metric.label === 'Perlu persetujuan'
				? '#perlu-tindakan'
				: resolve(`/dashboard/[role]?view=${metric.view}`, { role: 'patient' })}
			><WorkspaceIcon name={metric.icon} /><strong>{metric.value}</strong><span>{metric.label}</span
			><span class="metric-arrow"><WorkspaceIcon name="arrow" /></span></a
		>{/each}
</div>
<p class="overview-limit">
	Ringkasan dari maksimal 6 perawatan, pemeriksaan, dan resep terbaru. Buka setiap menu untuk daftar
	lengkap.
</p>

<section class="health-panel care-table-panel" id="perawatan" aria-labelledby="care-title">
	<div class="health-panel-heading">
		<div class="section-title">
			<WorkspaceIcon name="episodes" />
			<h2 id="care-title">Perawatan Anda</h2>
			<span class="panel-count"
				>{episodes.loading || episodes.error ? '—' : episodes.items.length}</span
			>
		</div>
		<a class="panel-link" href={resolve('/dashboard/[role]?view=episodes', { role: 'patient' })}
			>Lihat semua<WorkspaceIcon name="arrow" /></a
		>
	</div>
	{#if episodes.loading}<p class="resource-message" role="status">Memuat perawatan Anda…</p>
	{:else if episodes.error}<div class="resource-message resource-error" role="alert">
			<p>{episodes.error}</p>
			<button class="text-button" onclick={refresh}>Coba lagi</button>
		</div>
	{:else if !episodes.items.length}<div class="care-empty">
			<span class="empty-symbol"><WorkspaceIcon name="episodes" /></span>
			<div>
				<h3>Perjalanan Anda dimulai di sini.</h3>
				<p>Berikan kode pasien Anda kepada dokter. Undangan perawatan akan muncul di bagian ini.</p>
			</div>
			<a class="panel-link" href={resolve('/dashboard/[role]?view=episodes', { role: 'patient' })}
				>Buka perawatan<WorkspaceIcon name="arrow" /></a
			>
		</div>
	{:else}<div class="care-table-wrap">
			<table class="care-table">
				<thead
					><tr
						><th scope="col">Dokter</th><th scope="col">Organisasi</th><th scope="col">Status</th
						><th scope="col"><span class="sr-only">Tindakan</span></th></tr
					></thead
				><tbody
					>{#each episodes.items as episode (episode.id)}<tr
							><td
								><div class="table-person">
									<span class="person-initial" aria-hidden="true"
										>{episode.doctor_name.charAt(0)}</span
									>
									<h3>{episode.doctor_name}</h3>
								</div></td
							><td>{episode.organization_name || '—'}</td><td
								><span
									class="health-status"
									class:status-active={episode.status === 'active'}
									class:status-pending={episode.status === 'pending'}
									>{statuses[episode.status] || episode.status}</span
								></td
							><td
								><a
									class="table-action"
									href={resolve('/dashboard/[role]?view=episodes', { role: 'patient' })}
									aria-label={`Lihat perawatan ${episode.doctor_name}`}
									><WorkspaceIcon name="arrow" /></a
								></td
							></tr
						>{/each}</tbody
				>
			</table>
		</div>
		<ul class="care-mobile-list" aria-label="Daftar perawatan">
			{#each episodes.items as episode (episode.id)}
				<li>
					<div class="table-person">
						<span class="person-initial" aria-hidden="true">{episode.doctor_name.charAt(0)}</span>
						<div>
							<h3>{episode.doctor_name}</h3>
							<p>{episode.organization_name || 'Organisasi belum tersedia'}</p>
						</div>
					</div>
					<div class="care-mobile-actions">
						<span
							class="health-status"
							class:status-active={episode.status === 'active'}
							class:status-pending={episode.status === 'pending'}
							>{statuses[episode.status] || episode.status}</span
						><a
							class="panel-link"
							href={resolve('/dashboard/[role]?view=episodes', { role: 'patient' })}
							aria-label={`Lihat perawatan ${episode.doctor_name}`}
							>Lihat perawatan<WorkspaceIcon name="arrow" /></a
						>
					</div>
				</li>
			{/each}
		</ul>{/if}
</section>

<div class="clinical-panels">
	<section class="health-panel" id="pemeriksaan" aria-labelledby="lab-title">
		<div class="health-panel-heading">
			<div class="section-title">
				<WorkspaceIcon name="labs" />
				<h2 id="lab-title">Pemeriksaan</h2>
			</div>
			<a class="panel-link" href={resolve('/dashboard/[role]?view=labs', { role: 'patient' })}
				>Lihat semua<WorkspaceIcon name="arrow" /></a
			>
		</div>
		{#if labs.loading}<p class="resource-message" role="status">
				Memuat pemeriksaan…
			</p>{:else if labs.error}<div class="resource-message resource-error" role="alert">
				<p>{labs.error}</p>
				<button class="text-button" onclick={refresh}>Coba lagi</button>
			</div>{:else if !labs.items.length}<div class="compact-empty">
				<WorkspaceIcon name="labs" />
				<h3>Belum ada pemeriksaan</h3>
				<p>Permintaan dari dokter dan perkembangannya akan tampil di sini.</p>
			</div>{:else}<div class="clinical-list">
				{#each labs.items as lab (lab.id)}<article>
						<div class="clinical-row-title">
							<h3>{lab.test_name}</h3>
							<span
								class="health-status"
								class:status-active={lab.status === 'completed'}
								class:status-progress={['in_progress', 'accepted', 'sent'].includes(lab.status)}
								>{statuses[lab.status] || lab.status}</span
							>
						</div>
						<p>{lab.lab_name}</p>
					</article>{/each}
			</div>{/if}
		<p class="panel-footnote">Hingga 6 permintaan terbaru</p>
	</section>
	<section class="health-panel" id="obat" aria-labelledby="medicine-title">
		<div class="health-panel-heading">
			<div class="section-title">
				<WorkspaceIcon name="prescriptions" />
				<h2 id="medicine-title">Resep & obat</h2>
			</div>
			<a
				class="panel-link"
				href={resolve('/dashboard/[role]?view=prescriptions', { role: 'patient' })}
				>Lihat semua<WorkspaceIcon name="arrow" /></a
			>
		</div>
		{#if prescriptions.loading}<p class="resource-message" role="status">
				Memuat resep…
			</p>{:else if prescriptions.error}<div class="resource-message resource-error" role="alert">
				<p>{prescriptions.error}</p>
				<button class="text-button" onclick={refresh}>Coba lagi</button>
			</div>{:else if !prescriptions.items.length}<div class="compact-empty">
				<WorkspaceIcon name="prescriptions" />
				<h3>Belum ada resep</h3>
				<p>Resep dokter dan status penyiapan oleh apotek akan tampil di sini.</p>
			</div>{:else}<div class="clinical-list">
				{#each prescriptions.items as prescription (prescription.id)}<article>
						<div class="clinical-row-title">
							<h3>{prescription.pharmacy_name || 'Apotek belum dipilih'}</h3>
							<span class="health-status" class:status-active={prescription.status === 'ready'}
								>{statuses[prescription.status] || prescription.status}</span
							>
						</div>
						<details>
							<summary>Lihat obat & aturan pakai</summary
							>{#each prescription.items as item, index (index)}<div class="medication-item">
									<h3>{item.medication_name}</h3>
									<p>{item.dosage} · Jumlah: {item.quantity}</p>
									<p>{item.instructions}</p>
								</div>{/each}
						</details>
					</article>{/each}
			</div>{/if}
		<p class="panel-footnote">Hingga 6 resep terbaru</p>
	</section>
</div>

<div class="overview-lower">
	<section class="health-panel" id="riwayat" aria-labelledby="timeline-title">
		<div class="health-panel-heading">
			<div class="section-title">
				<WorkspaceIcon name="timeline" />
				<h2 id="timeline-title">Aktivitas terbaru</h2>
			</div>
			<a class="panel-link" href={resolve('/dashboard/[role]?view=timeline', { role: 'patient' })}
				>Lihat semua<WorkspaceIcon name="arrow" /></a
			>
		</div>
		<div class="timeline-filters" aria-label="Filter riwayat">
			{#each [['all', 'Semua'], ...Object.entries(kinds)] as [value, label] (value)}<button
					class:filter-active={filter === value}
					aria-pressed={filter === value}
					onclick={() => (filter = value)}>{label}</button
				>{/each}
		</div>
		{#if timeline.loading}<p class="resource-message" role="status">
				Memuat riwayat perawatan…
			</p>{:else if timeline.error}<div class="resource-message resource-error" role="alert">
				<p>{timeline.error}</p>
				<button class="text-button" onclick={refresh}>Coba lagi</button>
			</div>{:else if !filteredEvents.length}<div class="compact-empty">
				<h3>
					{timeline.items.length
						? 'Belum ada aktivitas pada kategori ini'
						: 'Belum ada aktivitas perawatan'}
				</h3>
				<p>
					{timeline.items.length
						? 'Pilih kategori lain untuk melihat riwayat yang tersedia.'
						: 'Perawatan dan keputusan persetujuan akan tercatat di sini.'}
				</p>
			</div>{:else}<ol class="patient-timeline">
				{#each filteredEvents as event (event.seq)}<li>
						<span class="timeline-marker" aria-hidden="true"></span>
						<div>
							<span class="timeline-kind">{kinds[event.kind] || event.kind}</span>
							<p>{event.message}</p>
							<time datetime={event.occurred_at}>{date(event.occurred_at)} WIB</time>
						</div>
					</li>{/each}
			</ol>{/if}
		<p class="panel-footnote">Filter pada 8 kejadian terbaru, bukan status terkini.</p>
	</section>
	<div class="overview-side">
		<section class="health-panel" id="perlu-tindakan" aria-labelledby="pending-title">
			<div class="health-panel-heading">
				<div class="section-title">
					<WorkspaceIcon name="consents" />
					<h2 id="pending-title">Perlu tindakan Anda</h2>
				</div>
			</div>
			{#if episodes.loading || labs.loading || prescriptions.loading}<p
					class="resource-message"
					role="status"
				>
					Memeriksa persetujuan…
				</p>{:else if pendingUnavailable}<div class="compact-empty">
					<h3>Ringkasan belum lengkap</h3>
					<p>Perbarui data atau buka menu Perawatan, Pemeriksaan, dan Resep & apotek.</p>
				</div>{:else if !pendingItems.length}<div class="compact-empty pending-empty">
					<span class="pending-check"><WorkspaceIcon name="consents" /></span>
					<h3>Tidak ada persetujuan tertunda</h3>
					<p>
						Dari daftar terbaru yang dimuat. Periksa juga permintaan di <a
							href={resolve('/dashboard/[role]?view=consents', { role: 'patient' })}>Izin dokter</a
						>.
					</p>
				</div>{:else}<div class="pending-list">
					{#each pendingItems as item (`${item.view}-${item.id}`)}<a
							href={resolve(`/dashboard/[role]?view=${item.view}`, { role: 'patient' })}
							><WorkspaceIcon name={item.icon} />
							<div>
								<h3>{item.title}</h3>
								<p>{item.text}</p>
							</div>
							<WorkspaceIcon name="arrow" /></a
						>{/each}
				</div>{/if}
		</section>
		<section class="health-panel patient-access" aria-labelledby="patient-code-title">
			<div class="section-title">
				<WorkspaceIcon name="audit" />
				<h2 id="patient-code-title">Identitas & akses</h2>
			</div>
			<div class="patient-code">
				<span>Kode pasien Anda</span>{#if user.patient_ref}<code>{user.patient_ref}</code>{:else}<p>
						Kode belum tersedia. Hubungi pengelola.
					</p>{/if}
			</div>
			<p>
				Berikan kode kepada dokter untuk menerima undangan. Akses data mengikuti perawatan dan
				persetujuan Anda.
			</p>
			<a class="panel-link" href={resolve('/dashboard/[role]?view=audit', { role: 'patient' })}
				>Periksa riwayat akses<WorkspaceIcon name="arrow" /></a
			>
		</section>
	</div>
</div>
