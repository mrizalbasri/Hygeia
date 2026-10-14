<script lang="ts">
	import { onMount } from 'svelte';
	import HealthIcon from './HealthIcon.svelte';
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
	let filteredEvents = $derived(
		timeline.items.filter((event) => filter === 'all' || event.kind === filter)
	);
	const kinds: Record<string, string> = {
		episode: 'Perawatan',
		laboratory: 'Pemeriksaan',
		prescription: 'Resep',
		consent: 'Persetujuan'
	};
	const date = (value: string) =>
		new Intl.DateTimeFormat('id-ID', {
			day: 'numeric',
			month: 'short',
			year: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		}).format(new Date(value));
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
		void refresh();
	});
</script>

<div class="patient-heading">
	<div>
		<p class="muted greeting">Halo, {user.full_name}.</p>
		<h1>Perjalanan Saya</h1>
		<p class="muted dashboard-intro">Ikuti perawatan Anda, satu langkah pada satu waktu.</p>
	</div>
	<button class="button secondary" onclick={refresh} disabled={refreshing}
		>{refreshing ? 'Memperbarui…' : 'Perbarui data'}</button
	>
</div>

<section class="patient-hero" aria-labelledby="patient-hero-title">
	<div>
		<h2 id="patient-hero-title">Perawatan yang terhubung.</h2>
		<p>Ikuti pemeriksaan, resep, dan perkembangan perawatan Anda.</p>
		<a href="#perawatan">Lihat perawatan saya <span aria-hidden="true">→</span></a>
	</div>
</section>

<div class="patient-columns">
	<div class="patient-primary">
		<section class="health-panel" id="perawatan" aria-labelledby="care-title">
			<div class="health-panel-heading">
				<div class="section-title">
					<span class="health-icon"><HealthIcon name="care" /></span>
					<h2 id="care-title">Perawatan Anda</h2>
				</div>
				<span class="section-caption">Hingga 6 terbaru</span>
			</div>
			{#if episodes.loading}<p class="resource-message" role="status">Memuat perawatan Anda…</p>
			{:else if episodes.error}<div class="resource-message resource-error" role="alert">
					<p>{episodes.error}</p>
					<button class="text-button" onclick={refresh}>Coba lagi</button>
				</div>
			{:else if episodes.items.length === 0}<div class="care-empty">
					<span class="empty-symbol"><HealthIcon name="care" /></span>
					<h3>Perjalanan Anda dimulai di sini.</h3>
					<p>
						Undangan dari dokter akan muncul di bagian ini. Setelah disetujui, Anda dapat mengikuti
						perawatan yang terhubung.
					</p>
				</div>
			{:else}<div class="care-list">
					{#each episodes.items as episode (episode.id)}<article class="care-row">
							<div class="person-initial" aria-hidden="true">{episode.doctor_name.charAt(0)}</div>
							<div>
								<h3>{episode.doctor_name}</h3>
								<p>{episode.organization_name}</p>
							</div>
							<span class:status-active={episode.status === 'active'} class="health-status"
								>{statuses[episode.status] || episode.status}</span
							>
						</article>{/each}
				</div>{/if}
		</section>

		<div class="clinical-panels">
			<section class="health-panel" id="pemeriksaan" aria-labelledby="lab-title">
				<div class="health-panel-heading">
					<div class="section-title">
						<span class="health-icon"><HealthIcon name="lab" /></span>
						<h2 id="lab-title">Pemeriksaan</h2>
					</div>
				</div>
				{#if labs.loading}<p class="resource-message" role="status">
						Memuat pemeriksaan…
					</p>{:else if labs.error}<div class="resource-message resource-error" role="alert">
						<p>{labs.error}</p>
						<button class="text-button" onclick={refresh}>Coba lagi</button>
					</div>{:else if !labs.items.length}<div class="compact-empty">
						<h3>Belum ada pemeriksaan</h3>
						<p>Permintaan lab dari dokter dan statusnya akan tampil di sini.</p>
					</div>{:else}<div class="clinical-list">
						{#each labs.items as lab (lab.id)}<article>
								<h3>{lab.test_name}</h3>
								<p>{lab.lab_name}</p>
								<span class="health-status">{statuses[lab.status] || lab.status}</span>
							</article>{/each}
					</div>{/if}
				<p class="panel-footnote">Menampilkan hingga 6 permintaan terbaru.</p>
			</section>
			<section class="health-panel" id="obat" aria-labelledby="medicine-title">
				<div class="health-panel-heading">
					<div class="section-title">
						<span class="health-icon"><HealthIcon name="medicine" /></span>
						<h2 id="medicine-title">Resep & obat</h2>
					</div>
				</div>
				{#if prescriptions.loading}<p class="resource-message" role="status">
						Memuat resep…
					</p>{:else if prescriptions.error}<div
						class="resource-message resource-error"
						role="alert"
					>
						<p>{prescriptions.error}</p>
						<button class="text-button" onclick={refresh}>Coba lagi</button>
					</div>{:else if !prescriptions.items.length}<div class="compact-empty">
						<h3>Belum ada resep</h3>
						<p>Obat yang diresepkan dokter dan status penyiapannya akan tampil di sini.</p>
					</div>{:else}<div class="clinical-list">
						{#each prescriptions.items as prescription (prescription.id)}<article>
								<span class="health-status" class:status-active={prescription.status === 'ready'}
									>{statuses[prescription.status] || prescription.status}</span
								>
								<p>{prescription.pharmacy_name || 'Apotek belum dipilih'}</p>
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
				<p class="panel-footnote">Menampilkan hingga 6 resep terbaru.</p>
			</section>
		</div>

		<section class="health-panel" id="riwayat" aria-labelledby="timeline-title">
			<div class="health-panel-heading">
				<div class="section-title">
					<span class="health-icon"><HealthIcon name="history" /></span>
					<h2 id="timeline-title">Riwayat perawatan</h2>
				</div>
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
							: 'Perkembangan perawatan dan keputusan persetujuan akan tersusun di sini.'}
					</p>
				</div>{:else}<ol class="patient-timeline">
					{#each filteredEvents as event (event.seq)}<li>
							<span class="timeline-kind">{kinds[event.kind] || event.kind}</span>
							<p>{event.message}</p>
							<time datetime={event.occurred_at}>{date(event.occurred_at)}</time>
						</li>{/each}
				</ol>{/if}
			<p class="panel-footnote">
				Filter berlaku pada 8 aktivitas terbaru. Riwayat merekam kejadian saat itu, bukan status
				terkini.
			</p>
		</section>
	</div>
	<aside class="patient-secondary" aria-label="Profil dan informasi akses">
		<section class="health-panel patient-profile">
			<p class="eyebrow muted">PROFIL ANDA</p>
			<div class="profile-initial" aria-hidden="true">
				{user.full_name
					.split(' ')
					.filter(Boolean)
					.slice(0, 2)
					.map((name) => name.charAt(0))
					.join('')}
			</div>
			<h2>{user.full_name}</h2>
			<span class="role-chip">Pasien</span>
			<dl>
				<div>
					<dt>Email</dt>
					<dd>{user.email}</dd>
				</div>
				<div>
					<dt>Organisasi</dt>
					<dd>{user.organization?.name || 'Akun pasien pribadi'}</dd>
				</div>
			</dl>
			<div class="profile-access-note">
				<p>Akses dokter mengikuti izin dan episode perawatan Anda.</p>
				<a href="#riwayat">Lihat riwayat perawatan</a>
			</div>
		</section>
	</aside>
</div>
