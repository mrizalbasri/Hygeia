<script lang="ts">
	import { goto } from '$app/navigation';
	import {
		currentUser,
		logout,
		labQueue,
		pharmacyQueue,
		submitLabResult,
		updateLabStatus,
		updatePrescriptionStatus,
		type LabQueueRequest,
		type LabResultInput,
		type QueuePrescription,
		type User
	} from '../lib/api';
	import { onMount } from 'svelte';

	let user = $state<User | null>(null);
	let labs = $state<LabQueueRequest[]>([]);
	let prescriptions = $state<QueuePrescription[]>([]);
	let loading = $state(true);
	let actionLoading = $state<string | null>(null);
	let error = $state('');
	let toastMessage = $state<string | null>(null);
	let toastTimer: ReturnType<typeof setTimeout> | null = null;

	// Search & Filter state
	let searchQuery = $state('');
	let labStatusFilter = $state<string>('all');
	let pharmacyStatusFilter = $state<string>('all');

	// Active modal for structured lab result
	let activeLab = $state<LabQueueRequest | null>(null);
	let result = $state<LabResultInput>({
		value: '',
		unit: '',
		reference_range: '',
		flag: 'normal',
		note: 'Data sintetis, bukan hasil pemeriksaan nyata.'
	});

	function showToast(message: string) {
		toastMessage = message;
		if (toastTimer) clearTimeout(toastTimer);
		toastTimer = setTimeout(() => {
			toastMessage = null;
		}, 4000);
	}

	onMount(async () => {
		try {
			user = await currentUser();
			await refresh();
		} catch {
			await goto('/login');
		} finally {
			loading = false;
		}
	});

	async function refresh() {
		error = '';
		try {
			if (user?.role === 'lab') {
				labs = await labQueue();
			} else if (user?.role === 'pharmacy') {
				prescriptions = await pharmacyQueue();
			}
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Gagal memuat antrean data.';
		}
	}

	async function changeLabStatus(item: LabQueueRequest, status: 'accepted' | 'in_progress') {
		error = '';
		actionLoading = item.id;
		try {
			await updateLabStatus(item.id, status);
			const label = status === 'accepted' ? 'diterima' : 'mulai diproses';
			showToast(`Pemeriksaan ${item.test_code} berhasil ${label}.`);
			await refresh();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Status pemeriksaan tidak dapat diubah.';
		} finally {
			actionLoading = null;
		}
	}

	function openResult(item: LabQueueRequest) {
		activeLab = item;
		result = {
			value: '',
			unit: '',
			reference_range: '',
			flag: 'normal',
			note: 'Data sintetis, bukan hasil pemeriksaan nyata.'
		};
	}

	function closeResult() {
		activeLab = null;
	}

	async function sendResult() {
		if (!activeLab) return;
		error = '';
		actionLoading = activeLab.id;
		try {
			await submitLabResult(activeLab.id, result);
			showToast(`Hasil pemeriksaan untuk ${activeLab.patient_ref} berhasil dikirim.`);
			activeLab = null;
			await refresh();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Hasil pemeriksaan gagal dikirim.';
		} finally {
			actionLoading = null;
		}
	}

	async function changePrescriptionStatus(item: QueuePrescription) {
		const nextStatusMap: Record<string, 'accepted' | 'preparing' | 'ready' | 'completed'> = {
			sent: 'accepted',
			accepted: 'preparing',
			preparing: 'ready',
			ready: 'completed'
		};

		const next = nextStatusMap[item.status];
		if (!next) return;

		error = '';
		actionLoading = item.id;
		try {
			await updatePrescriptionStatus(item.id, next);
			const labelMap: Record<string, string> = {
				accepted: 'diterima',
				preparing: 'sedang disiapkan',
				ready: 'siap diserahkan ke pasien',
				completed: 'selesai diserahkan'
			};
			showToast(`Resep untuk ${item.patient_ref} ${labelMap[next] || 'diperbarui'}.`);
			await refresh();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Status resep tidak dapat diubah.';
		} finally {
			actionLoading = null;
		}
	}

	async function signOut() {
		try {
			await logout();
		} finally {
			await goto('/login');
		}
	}

	// Filtered lists using derived runes
	let filteredLabs = $derived(
		labs.filter((item) => {
			const matchStatus = labStatusFilter === 'all' || item.status === labStatusFilter;
			const q = searchQuery.toLowerCase().trim();
			const matchSearch =
				!q ||
				item.patient_ref.toLowerCase().includes(q) ||
				item.test_name.toLowerCase().includes(q) ||
				item.test_code.toLowerCase().includes(q);
			return matchStatus && matchSearch;
		})
	);

	let filteredPrescriptions = $derived(
		prescriptions.filter((item) => {
			const matchStatus = pharmacyStatusFilter === 'all' || item.status === pharmacyStatusFilter;
			const q = searchQuery.toLowerCase().trim();
			const matchSearch =
				!q ||
				item.patient_ref.toLowerCase().includes(q) ||
				item.id.toLowerCase().includes(q) ||
				item.items.some((m) => m.medication_name.toLowerCase().includes(q));
			return matchStatus && matchSearch;
		})
	);

	// Derived metrics
	let labStats = $derived({
		total: labs.length,
		sent: labs.filter((i) => i.status === 'sent').length,
		inProgress: labs.filter((i) => i.status === 'accepted' || i.status === 'in_progress').length
	});

	let pharmacyStats = $derived({
		total: prescriptions.length,
		sent: prescriptions.filter((i) => i.status === 'sent').length,
		preparing: prescriptions.filter((i) => i.status === 'accepted' || i.status === 'preparing').length,
		ready: prescriptions.filter((i) => i.status === 'ready').length
	});
</script>

<svelte:head>
	<title>
		{user
			? user.role === 'lab'
				? 'Ruang Kerja Laboratorium · Hygeia'
				: user.role === 'pharmacy'
					? 'Ruang Kerja Apotek · Hygeia'
					: 'Dashboard · Hygeia'
			: 'Dashboard · Hygeia'}
	</title>
</svelte:head>

{#if loading}
	<div class="center-loading">
		<div class="spinner"></div>
		<p>Menyiapkan ruang kerja Hygeia...</p>
	</div>
{:else if user}
	<div class="app-shell">
		<!-- Signature Hygeia ribbon border at very top -->
		<div class="brand-ribbon-bar"></div>

		<!-- Top Header Navigation -->
		<header class="topbar">
			<div class="brand-group">
				<div class="brand-logo" aria-hidden="true">H+</div>
				<div>
					<div class="title-with-badge">
						<h1 class="page-title">
							{user.role === 'lab'
								? 'Laboratorium'
								: user.role === 'pharmacy'
									? 'Apotek'
									: 'Ruang Kerja'}
						</h1>
						<span class="role-chip">
							{user.role === 'lab'
								? 'Staf Lab'
								: user.role === 'pharmacy'
									? 'Staf Farmasi'
									: user.role}
						</span>
					</div>
					<p class="subtitle">
						{user.organization?.name ?? 'Pusat Layanan Kesehatan'} · Terhubung & Terverifikasi
					</p>
				</div>
			</div>

			<div class="user-control">
				<div class="account-card">
					<div class="avatar" aria-hidden="true">
						{user.full_name
							.split(' ')
							.map((n) => n[0])
							.slice(0, 2)
							.join('')
							.toUpperCase()}
					</div>
					<div class="account-info">
						<span class="account-name">{user.full_name}</span>
						<span class="account-org">{user.organization?.name ?? user.email}</span>
					</div>
				</div>
				<button class="btn-ghost" onclick={signOut} title="Keluar dari akun">
					<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
						<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
						<polyline points="16 17 21 12 16 7" />
						<line x1="21" y1="12" x2="9" y2="12" />
					</svg>
					<span>Keluar</span>
				</button>
			</div>
		</header>

		<!-- Main Workspace Area -->
		<main class="main-content">
			<!-- Toast Alert Banner -->
			{#if toastMessage}
				<div class="toast-banner" role="status">
					<div class="toast-icon">
						<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
							<polyline points="20 6 9 17 4 12" />
						</svg>
					</div>
					<span>{toastMessage}</span>
					<button class="toast-close" onclick={() => (toastMessage = null)} aria-label="Tutup notifikasi">×</button>
				</div>
			{/if}

			{#if error}
				<div class="alert-error" role="alert">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<circle cx="12" cy="12" r="10" />
						<line x1="12" y1="8" x2="12" y2="12" />
						<line x1="12" y1="16" x2="12.01" y2="16" />
					</svg>
					<span>{error}</span>
				</div>
			{/if}

			<!-- ================================= LAB WORKSPACE ================================= -->
			{#if user.role === 'lab'}
				<!-- Metric Stats Header -->
				<section class="metrics-grid">
					<div class="metric-card">
						<span class="metric-label">Total Permintaan Masuk</span>
						<span class="metric-value">{labStats.total}</span>
						<span class="metric-hint">Semua antrean aktif lab</span>
					</div>
					<div class="metric-card metric-warning">
						<span class="metric-label">Perlu Diterima Segera</span>
						<span class="metric-value">{labStats.sent}</span>
						<span class="metric-hint">Status baru dari dokter</span>
					</div>
					<div class="metric-card metric-info">
						<span class="metric-label">Sedang Diproses</span>
						<span class="metric-value">{labStats.inProgress}</span>
						<span class="metric-hint">Diterima & dalam tahap uji</span>
					</div>
				</section>

				<!-- Toolbar: Filter Tabs & Search -->
				<div class="toolbar-panel">
					<div class="status-tabs" role="tablist">
						<button
							class="tab-btn {labStatusFilter === 'all' ? 'active' : ''}"
							onclick={() => (labStatusFilter = 'all')}
							role="tab"
						>
							Semua ({labs.length})
						</button>
						<button
							class="tab-btn {labStatusFilter === 'sent' ? 'active' : ''}"
							onclick={() => (labStatusFilter = 'sent')}
							role="tab"
						>
							Baru Masuk ({labStats.sent})
						</button>
						<button
							class="tab-btn {labStatusFilter === 'accepted' ? 'active' : ''}"
							onclick={() => (labStatusFilter = 'accepted')}
							role="tab"
						>
							Diterima ({labs.filter((i) => i.status === 'accepted').length})
						</button>
						<button
							class="tab-btn {labStatusFilter === 'in_progress' ? 'active' : ''}"
							onclick={() => (labStatusFilter = 'in_progress')}
							role="tab"
						>
							Dalam Uji ({labs.filter((i) => i.status === 'in_progress').length})
						</button>
					</div>

					<div class="search-actions">
						<div class="search-box">
							<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<circle cx="11" cy="11" r="8" />
								<line x1="21" y1="21" x2="16.65" y2="16.65" />
							</svg>
							<input
								type="text"
								placeholder="Cari pasien / nama tes / kode..."
								bind:value={searchQuery}
							/>
							{#if searchQuery}
								<button class="clear-btn" onclick={() => (searchQuery = '')} aria-label="Bersihkan pencarian">×</button>
							{/if}
						</div>
						<button class="btn-secondary" onclick={refresh} title="Perbarui data antrean">
							<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67" />
							</svg>
							<span>Segarkan</span>
						</button>
					</div>
				</div>

				<!-- Queue Cards List -->
				<div class="queue-list">
					{#each filteredLabs as item (item.id)}
						<article class="data-card">
							<div class="card-main">
								<div class="card-header">
									<div class="header-details">
										<div class="badge-status status-{item.status}">
											{#if item.status === 'sent'}
												<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
													<circle cx="12" cy="12" r="10" />
													<polyline points="12 6 12 12 16 14" />
												</svg>
												<span>Baru Masuk</span>
											{:else if item.status === 'accepted'}
												<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
													<path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
													<polyline points="22 4 12 14.01 9 11.01" />
												</svg>
												<span>Diterima Staf</span>
											{:else}
												<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
													<path d="M6 2v14a6 6 0 0 0 12 0V2" />
													<line x1="6" y1="6" x2="18" y2="6" />
													<line x1="6" y1="10" x2="18" y2="10" />
												</svg>
												<span>Sedang Diperiksa</span>
											{/if}
										</div>
										<code class="code-pill">{item.test_code}</code>
									</div>
									<div class="patient-tag">
										<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
											<path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
											<circle cx="12" cy="7" r="4" />
										</svg>
										<span>Pasien: <strong>{item.patient_ref}</strong></span>
									</div>
								</div>

								<div class="test-body">
									<h2 class="test-title">{item.test_name}</h2>
									<p class="test-instruction">
										{#if item.status === 'sent'}
											Permintaan baru dari dokter. Silakan konfirmasi terima untuk memulai persiapan sampel.
										{:else if item.status === 'accepted'}
											Permintaan diterima. Mulai pemeriksaan di laboratorium untuk menganalisis spesimen.
										{:else}
											Pemeriksaan sedang berlangsung. Setelah hasil valid, masukkan data hasil terstruktur di bawah.
										{/if}
									</p>
								</div>
							</div>

							<div class="card-footer">
								{#if item.status === 'sent'}
									<button
										class="btn-primary"
										disabled={actionLoading === item.id}
										onclick={() => changeLabStatus(item, 'accepted')}
									>
										{actionLoading === item.id ? 'Memproses...' : 'Terima Permintaan'}
									</button>
								{:else if item.status === 'accepted'}
									<button
										class="btn-primary"
										disabled={actionLoading === item.id}
										onclick={() => changeLabStatus(item, 'in_progress')}
									>
										{actionLoading === item.id ? 'Memproses...' : 'Mulai Pemeriksaan'}
									</button>
								{:else}
									<button class="btn-highlight" onclick={() => openResult(item)}>
										<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
											<path d="M12 20h9" />
											<path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z" />
										</svg>
										<span>Input Hasil Uji</span>
									</button>
								{/if}
							</div>
						</article>
					{:else}
						<div class="empty-state">
							<div class="empty-icon">🔬</div>
							<h3>Tidak ada permintaan laboratorium</h3>
							<p>
								{searchQuery
									? 'Tidak ditemukan permintaan yang cocok dengan kata kunci pencarian.'
									: 'Saat ini belum ada permintaan baru di antrean laboratorium Anda.'}
							</p>
						</div>
					{/each}
				</div>

			<!-- ================================= PHARMACY WORKSPACE ================================= -->
			{:else if user.role === 'pharmacy'}
				<!-- Metric Stats Header -->
				<section class="metrics-grid">
					<div class="metric-card">
						<span class="metric-label">Total Resep Ditugaskan</span>
						<span class="metric-value">{pharmacyStats.total}</span>
						<span class="metric-hint">Resep aktif yang dipilih pasien</span>
					</div>
					<div class="metric-card metric-warning">
						<span class="metric-label">Perlu Diterima</span>
						<span class="metric-value">{pharmacyStats.sent}</span>
						<span class="metric-hint">Status kirim baru dari pasien</span>
					</div>
					<div class="metric-card metric-info">
						<span class="metric-label">Sedang Disiapkan</span>
						<span class="metric-value">{pharmacyStats.preparing}</span>
						<span class="metric-hint">Obat sedang diracik staf</span>
					</div>
					<div class="metric-card metric-success">
						<span class="metric-label">Siap Diambil</span>
						<span class="metric-value">{pharmacyStats.ready}</span>
						<span class="metric-hint">Menunggu penyerahan pasien</span>
					</div>
				</section>

				<!-- Toolbar: Filter Tabs & Search -->
				<div class="toolbar-panel">
					<div class="status-tabs" role="tablist">
						<button
							class="tab-btn {pharmacyStatusFilter === 'all' ? 'active' : ''}"
							onclick={() => (pharmacyStatusFilter = 'all')}
							role="tab"
						>
							Semua ({prescriptions.length})
						</button>
						<button
							class="tab-btn {pharmacyStatusFilter === 'sent' ? 'active' : ''}"
							onclick={() => (pharmacyStatusFilter = 'sent')}
							role="tab"
						>
							Baru ({pharmacyStats.sent})
						</button>
						<button
							class="tab-btn {pharmacyStatusFilter === 'accepted' ? 'active' : ''}"
							onclick={() => (pharmacyStatusFilter = 'accepted')}
							role="tab"
						>
							Diterima ({prescriptions.filter((i) => i.status === 'accepted').length})
						</button>
						<button
							class="tab-btn {pharmacyStatusFilter === 'preparing' ? 'active' : ''}"
							onclick={() => (pharmacyStatusFilter = 'preparing')}
							role="tab"
						>
							Disiapkan ({prescriptions.filter((i) => i.status === 'preparing').length})
						</button>
						<button
							class="tab-btn {pharmacyStatusFilter === 'ready' ? 'active' : ''}"
							onclick={() => (pharmacyStatusFilter = 'ready')}
							role="tab"
						>
							Siap ({pharmacyStats.ready})
						</button>
					</div>

					<div class="search-actions">
						<div class="search-box">
							<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<circle cx="11" cy="11" r="8" />
								<line x1="21" y1="21" x2="16.65" y2="16.65" />
							</svg>
							<input
								type="text"
								placeholder="Cari kode pasien / nama obat..."
								bind:value={searchQuery}
							/>
							{#if searchQuery}
								<button class="clear-btn" onclick={() => (searchQuery = '')} aria-label="Bersihkan pencarian">×</button>
							{/if}
						</div>
						<button class="btn-secondary" onclick={refresh} title="Perbarui data antrean resep">
							<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67" />
							</svg>
							<span>Segarkan</span>
						</button>
					</div>
				</div>

				<!-- Queue Cards List -->
				<div class="queue-list">
					{#each filteredPrescriptions as item (item.id)}
						<article class="data-card">
							<div class="card-main">
								<div class="card-header">
									<div class="header-details">
										<div class="badge-status status-{item.status}">
											{#if item.status === 'sent'}
												<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
													<circle cx="12" cy="12" r="10" />
													<polyline points="12 6 12 12 16 14" />
												</svg>
												<span>Resep Masuk</span>
											{:else if item.status === 'accepted'}
												<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
													<path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
													<polyline points="22 4 12 14.01 9 11.01" />
												</svg>
												<span>Diterima Apotek</span>
											{:else if item.status === 'preparing'}
												<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
													<path d="M10.5 20.5 3 13l7.5-7.5" />
													<path d="m14 4 7 7-7 7" />
												</svg>
												<span>Sedang Dirack</span>
											{:else}
												<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
													<path d="M20 6 9 17l-5-5" />
												</svg>
												<span>Siap Diserahkan</span>
											{/if}
										</div>
										<code class="code-pill">ID: {item.id.slice(0, 8)}</code>
									</div>
									<div class="patient-tag">
										<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
											<path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
											<circle cx="12" cy="7" r="4" />
										</svg>
										<span>Pasien: <strong>{item.patient_ref}</strong></span>
									</div>
								</div>

								<div class="medicines-section">
									<h3 class="section-subtitle">Daftar Obat ({item.items.length} item)</h3>
									<div class="medicines-grid">
										{#each item.items as med}
											<div class="medicine-item">
												<div class="med-icon">💊</div>
												<div class="med-info">
													<div class="med-header">
														<span class="med-name">{med.medication_name}</span>
														<span class="med-qty">{med.quantity} unit</span>
													</div>
													<div class="med-details">
														<span>Dosis: <strong>{med.dosage}</strong></span>
														<span class="bullet">·</span>
														<span>Aturan: {med.instructions}</span>
													</div>
												</div>
											</div>
										{/each}
									</div>
								</div>
							</div>

							<div class="card-footer">
								{#if item.status === 'sent'}
									<button
										class="btn-primary"
										disabled={actionLoading === item.id}
										onclick={() => changePrescriptionStatus(item)}
									>
										{actionLoading === item.id ? 'Memproses...' : 'Terima Resep'}
									</button>
								{:else if item.status === 'accepted'}
									<button
										class="btn-primary"
										disabled={actionLoading === item.id}
										onclick={() => changePrescriptionStatus(item)}
									>
										{actionLoading === item.id ? 'Memproses...' : 'Mulai Siapkan Obat'}
									</button>
								{:else if item.status === 'preparing'}
									<button
										class="btn-success"
										disabled={actionLoading === item.id}
										onclick={() => changePrescriptionStatus(item)}
									>
										{actionLoading === item.id ? 'Memproses...' : 'Tandai Siap Diambil'}
									</button>
								{:else if item.status === 'ready'}
									<button
										class="btn-primary"
										disabled={actionLoading === item.id}
										onclick={() => changePrescriptionStatus(item)}
									>
										{actionLoading === item.id ? 'Memproses...' : 'Selesaikan Resep (Penyerahan)'}
									</button>
								{/if}
							</div>
						</article>
					{:else}
						<div class="empty-state">
							<div class="empty-icon">💊</div>
							<h3>Tidak ada antrean resep</h3>
							<p>
								{searchQuery
									? 'Tidak ditemukan resep yang cocok dengan kata kunci pencarian.'
									: 'Saat ini belum ada resep yang dialokasikan ke apotek Anda.'}
							</p>
						</div>
					{/each}
				</div>

			<!-- ================================= FALLBACK OTHER ROLES ================================= -->
			{:else}
				<div class="empty-panel">
					<div class="brand-logo big-logo" aria-hidden="true">H+</div>
					<h2>Ruang Kerja Khusus Staf Laboratorium & Apotek</h2>
					<p>
						Akun Anda saat ini masuk sebagai <strong>{user.role}</strong> ({user.full_name}).
						Halaman ini dikhususkan untuk pengujian dan pengelolaan antrean Laboratorium dan Farmasi/Apotek.
					</p>
					<div class="demo-hints">
						<p>Gunakan akun demo berikut untuk menguji:</p>
						<ul>
							<li>Laboratorium: <code>lab@demo.hygeia.test</code></li>
							<li>Apotek: <code>apotek@demo.hygeia.test</code></li>
						</ul>
					</div>
					<button class="btn-primary" onclick={signOut}>Ganti Akun</button>
				</div>
			{/if}
		</main>
	</div>

	<!-- ================================= MODAL INPUT HASIL LAB ================================= -->
	{#if activeLab}
		<div
			class="modal-backdrop"
			onclick={closeResult}
			onkeydown={(e) => {
				if (e.key === 'Escape') closeResult();
			}}
			role="button"
			tabindex="0"
			aria-label="Tutup jendela modal"
		>
			<!-- svelte-ignore a11y_click_events_have_key_events -->
			<div
				class="modal-dialog"
				onclick={(e) => e.stopPropagation()}
				role="dialog"
				aria-labelledby="modal-title"
				aria-modal="true"
				tabindex="-1"
			>
				<header class="modal-header">
					<div>
						<span class="modal-eyebrow">Pengiriman Hasil Terstruktur</span>
						<h2 id="modal-title">{activeLab.test_name}</h2>
						<p class="modal-meta">
							Pasien Ref: <code>{activeLab.patient_ref}</code> · Kode: <code>{activeLab.test_code}</code>
						</p>
					</div>
					<button class="btn-close" onclick={closeResult} aria-label="Tutup modal">×</button>
				</header>

				<form
					onsubmit={(e) => {
						e.preventDefault();
						sendResult();
					}}
				>
					<div class="modal-body">
						<div class="form-group">
							<label for="res-value">
								Nilai Hasil Pemeriksaan <span class="required">*</span>
							</label>
							<input
								id="res-value"
								type="text"
								bind:value={result.value}
								placeholder="Contoh: 110 atau Negatif"
								required
								maxlength="500"
							/>
						</div>

						<div class="grid-2">
							<div class="form-group">
								<label for="res-unit">Satuan Pengukuran</label>
								<input
									id="res-unit"
									type="text"
									bind:value={result.unit}
									placeholder="Contoh: mg/dL, g/dL, %"
									maxlength="100"
								/>
							</div>

							<div class="form-group">
								<label for="res-ref">Rentang Rujukan Standar</label>
								<input
									id="res-ref"
									type="text"
									bind:value={result.reference_range}
									placeholder="Contoh: 70 - 99 mg/dL"
									maxlength="200"
								/>
							</div>
						</div>

						<div class="form-group">
							<label for="res-flag">
								Penanda Klinis (Flag) <span class="required">*</span>
							</label>
							<select id="res-flag" bind:value={result.flag}>
								<option value="normal">Normal (Dalam batas normal)</option>
								<option value="high">Tinggi (High)</option>
								<option value="low">Rendah (Low)</option>
								<option value="abnormal">Abnormal (Penyimpangan hasil)</option>
								<option value="unknown">Tidak Diketahui / Lainnya</option>
							</select>
						</div>

						<div class="form-group">
							<label for="res-note">Catatan Tambahan Laboratorium</label>
							<textarea
								id="res-note"
								bind:value={result.note}
								rows="3"
								maxlength="2000"
								placeholder="Tambahkan catatan jika diperlukan..."
							></textarea>
						</div>

						<div class="modal-notice">
							<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<circle cx="12" cy="12" r="10" />
								<line x1="12" y1="16" x2="12" y2="12" />
								<line x1="12" y1="8" x2="12.01" y2="8" />
							</svg>
							<span>Hasil ini akan dicatat ke riwayat klinis pasien dan diaudit dengan HMAC. Pengiriman tidak dapat diubah setelah dikonfirmasi.</span>
						</div>
					</div>

					<footer class="modal-footer">
						<button type="button" class="btn-ghost" onclick={closeResult}>Batal</button>
						<button type="submit" class="btn-primary" disabled={actionLoading === activeLab.id}>
							{actionLoading === activeLab.id ? 'Mengirim Hasil...' : 'Kirim Hasil Resmi'}
						</button>
					</footer>
				</form>
			</div>
		</div>
	{/if}
{/if}

<style>
	/* App Shell & Layout */
	.app-shell {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
		background-color: var(--bg);
	}

	.brand-ribbon-bar {
		height: 4px;
		background: var(--ribbon);
		width: 100%;
	}

	.topbar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 24px;
		padding: 16px max(24px, calc((100vw - 1200px) / 2));
		background: var(--surface);
		border-bottom: 1px solid var(--border);
		position: sticky;
		top: 0;
		z-index: 20;
		box-shadow: 0 1px 3px rgba(11, 16, 28, 0.04);
	}

	.brand-group {
		display: flex;
		align-items: center;
		gap: 16px;
	}

	.brand-logo {
		width: 44px;
		height: 44px;
		display: grid;
		place-items: center;
		border-radius: 12px;
		background: #0b101c;
		color: #1aebfd;
		font-weight: 800;
		font-size: 18px;
		letter-spacing: -0.02em;
		box-shadow: 0 4px 12px rgba(11, 16, 28, 0.15);
		border: 1px solid #2b3345;
	}

	.title-with-badge {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	.page-title {
		margin: 0;
		font-size: 22px;
		font-weight: 700;
		color: var(--text);
		line-height: 1.2;
	}

	.role-chip {
		display: inline-flex;
		align-items: center;
		padding: 2px 10px;
		border-radius: 999px;
		background: #ebf1ff;
		color: var(--primary);
		font-size: 11px;
		font-weight: 700;
		letter-spacing: 0.04em;
		text-transform: uppercase;
	}

	.subtitle {
		margin: 3px 0 0;
		color: var(--text-muted);
		font-size: 13px;
	}

	.user-control {
		display: flex;
		align-items: center;
		gap: 14px;
	}

	.account-card {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 6px 12px;
		border-radius: 12px;
		background: #f8fafc;
		border: 1px solid var(--border);
	}

	.avatar {
		width: 34px;
		height: 34px;
		border-radius: 50%;
		background: #034bfd;
		color: white;
		display: grid;
		place-items: center;
		font-size: 12px;
		font-weight: 700;
	}

	.account-info {
		display: flex;
		flex-direction: column;
		text-align: left;
	}

	.account-name {
		font-size: 13px;
		font-weight: 700;
		color: var(--text);
	}

	.account-org {
		font-size: 11px;
		color: var(--text-muted);
	}

	/* Main Workspace */
	.main-content {
		max-width: 1200px;
		width: 100%;
		margin: 0 auto;
		padding: 32px 24px 64px;
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	/* Toast Notification */
	.toast-banner {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 14px 18px;
		border-radius: 14px;
		background: #e8f7ee;
		border: 1px solid #c2ebd1;
		color: #0b6b3a;
		font-size: 14px;
		font-weight: 600;
		box-shadow: 0 4px 12px rgba(11, 107, 58, 0.08);
		animation: slideDown 0.25s ease-out;
	}

	.toast-icon {
		display: grid;
		place-items: center;
	}

	.toast-close {
		margin-left: auto;
		background: transparent;
		border: none;
		color: inherit;
		font-size: 20px;
		line-height: 1;
		cursor: pointer;
		opacity: 0.6;
	}
	.toast-close:hover {
		opacity: 1;
	}

	.alert-error {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 14px 18px;
		border-radius: 14px;
		background: var(--status-error-bg);
		border: 1px solid #fecdd3;
		color: var(--status-error-text);
		font-size: 14px;
		font-weight: 600;
	}

	/* Metrics Grid */
	.metrics-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
		gap: 16px;
	}

	.metric-card {
		padding: 20px 22px;
		border-radius: 16px;
		background: var(--surface);
		border: 1px solid var(--border);
		box-shadow: 0 1px 3px rgba(11, 16, 28, 0.04);
		display: flex;
		flex-direction: column;
		gap: 4px;
		transition: transform 0.15s ease, box-shadow 0.15s ease;
	}
	.metric-card:hover {
		transform: translateY(-2px);
		box-shadow: 0 6px 16px rgba(11, 16, 28, 0.06);
	}

	.metric-label {
		font-size: 12px;
		font-weight: 700;
		color: var(--text-muted);
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.metric-value {
		font-size: 32px;
		font-weight: 800;
		color: var(--text);
		line-height: 1.1;
		margin: 4px 0;
	}

	.metric-hint {
		font-size: 12px;
		color: var(--text-muted);
	}

	.metric-warning .metric-value {
		color: #b45309;
	}
	.metric-info .metric-value {
		color: var(--primary);
	}
	.metric-success .metric-value {
		color: #0b6b3a;
	}

	/* Toolbar Panel */
	.toolbar-panel {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: center;
		gap: 16px;
		padding: 12px 16px;
		border-radius: 14px;
		background: var(--surface);
		border: 1px solid var(--border);
	}

	.status-tabs {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}

	.tab-btn {
		padding: 8px 14px;
		border-radius: 10px;
		border: 1px solid transparent;
		background: transparent;
		color: var(--text-muted);
		font-size: 13px;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.15s ease;
	}
	.tab-btn:hover {
		background: #f4f7fc;
		color: var(--text);
	}
	.tab-btn.active {
		background: #0b101c;
		color: #ffffff;
		border-color: #0b101c;
	}

	.search-actions {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	.search-box {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 0 12px;
		border-radius: 10px;
		background: #f8fafc;
		border: 1px solid var(--border);
		width: 260px;
	}
	.search-box:focus-within {
		border-color: var(--primary);
		background: white;
		outline: 2px solid rgba(3, 75, 253, 0.15);
	}
	.search-box input {
		width: 100%;
		border: none;
		background: transparent;
		padding: 9px 0;
		font: inherit;
		font-size: 13px;
		color: var(--text);
	}
	.search-box input:focus {
		outline: none;
	}
	.search-box svg {
		color: var(--text-muted);
	}

	.clear-btn {
		background: transparent;
		border: none;
		color: var(--text-muted);
		font-size: 16px;
		cursor: pointer;
		padding: 0 4px;
	}

	/* Queue Cards */
	.queue-list {
		display: grid;
		gap: 16px;
	}

	.data-card {
		padding: 22px 24px;
		border-radius: 16px;
		background: var(--surface);
		border: 1px solid var(--border);
		box-shadow: 0 1px 3px rgba(11, 16, 28, 0.04);
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 24px;
		transition: all 0.18s ease;
	}
	.data-card:hover {
		border-color: #b7c6df;
		box-shadow: 0 4px 14px rgba(11, 16, 28, 0.06);
	}

	.card-main {
		display: flex;
		flex-direction: column;
		gap: 12px;
		flex: 1;
	}

	.card-header {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
	}

	.header-details {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	/* Badges */
	.badge-status {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 4px 10px;
		border-radius: 999px;
		font-size: 12px;
		font-weight: 700;
	}

	.status-sent {
		background: var(--status-pending-bg);
		color: var(--status-pending-text);
		border: 1px solid #fed7aa;
	}
	.status-accepted {
		background: #eef4ff;
		color: #1d4ed8;
		border: 1px solid #bfdbfe;
	}
	.status-in_progress,
	.status-preparing {
		background: #f0fdf4;
		color: #15803d;
		border: 1px solid #bbf7d0;
	}
	.status-ready {
		background: #ecfdf5;
		color: #047857;
		border: 1px solid #a7f3d0;
	}

	.code-pill {
		padding: 4px 8px;
		border-radius: 8px;
		background: #f1f5f9;
		color: #475569;
		font-size: 11px;
		font-weight: 600;
		letter-spacing: 0.02em;
	}

	.patient-tag {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		color: var(--text-muted);
		font-size: 13px;
	}
	.patient-tag strong {
		color: var(--text);
		font-family: var(--font-mono);
	}

	.test-title {
		margin: 0;
		font-size: 18px;
		font-weight: 700;
		color: var(--text);
	}

	.test-instruction {
		margin: 4px 0 0;
		color: var(--text-muted);
		font-size: 13px;
		line-height: 1.5;
	}

	/* Medicines for Pharmacy */
	.medicines-section {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.section-subtitle {
		margin: 0;
		font-size: 13px;
		font-weight: 700;
		color: var(--text-muted);
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.medicines-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
		gap: 10px;
	}

	.medicine-item {
		display: flex;
		gap: 10px;
		padding: 10px 12px;
		border-radius: 12px;
		background: #f8fafc;
		border: 1px solid var(--border-subtle);
	}

	.med-icon {
		font-size: 16px;
		line-height: 1.4;
	}

	.med-info {
		display: flex;
		flex-direction: column;
		gap: 2px;
		flex: 1;
	}

	.med-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 8px;
	}

	.med-name {
		font-size: 14px;
		font-weight: 700;
		color: var(--text);
	}

	.med-qty {
		padding: 2px 7px;
		border-radius: 6px;
		background: #e2e8f0;
		color: #334155;
		font-size: 11px;
		font-weight: 700;
	}

	.med-details {
		font-size: 12px;
		color: var(--text-muted);
	}
	.bullet {
		margin: 0 4px;
	}

	/* Card Footer Buttons */
	.card-footer {
		display: flex;
		align-items: center;
	}

	/* Buttons */
	button {
		font-family: inherit;
		font-size: 13px;
		font-weight: 700;
		cursor: pointer;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		border-radius: 12px;
		min-height: 42px;
		padding: 0 18px;
		transition: all 0.15s ease;
	}
	button:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.btn-primary {
		border: 1px solid var(--primary);
		background: var(--primary);
		color: white;
		box-shadow: 0 2px 6px rgba(3, 75, 253, 0.2);
	}
	.btn-primary:hover:not(:disabled) {
		background: var(--primary-hover);
		transform: translateY(-1px);
	}

	.btn-highlight {
		border: 1px solid #034bfd;
		background: linear-gradient(135deg, #034bfd, #00abef);
		color: white;
		box-shadow: 0 3px 8px rgba(3, 75, 253, 0.25);
	}
	.btn-highlight:hover:not(:disabled) {
		filter: brightness(1.08);
		transform: translateY(-1px);
	}

	.btn-success {
		border: 1px solid #059669;
		background: #059669;
		color: white;
		box-shadow: 0 2px 6px rgba(5, 150, 105, 0.2);
	}
	.btn-success:hover:not(:disabled) {
		background: #047857;
	}

	.btn-secondary {
		border: 1px solid var(--border);
		background: white;
		color: var(--text);
	}
	.btn-secondary:hover {
		background: #f8fafc;
	}

	.btn-ghost {
		border: 1px solid transparent;
		background: transparent;
		color: var(--text-muted);
	}
	.btn-ghost:hover {
		background: #f1f5f9;
		color: var(--text);
	}

	/* Empty State */
	.empty-state {
		padding: 56px 24px;
		border: 2px dashed var(--border);
		border-radius: 18px;
		background: var(--surface);
		text-align: center;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
	}
	.empty-icon {
		font-size: 40px;
		margin-bottom: 8px;
	}
	.empty-state h3 {
		margin: 0;
		font-size: 18px;
		color: var(--text);
	}
	.empty-state p {
		margin: 0;
		color: var(--text-muted);
		font-size: 14px;
		max-width: 420px;
	}

	/* Modal Dialog */
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(11, 16, 28, 0.65);
		backdrop-filter: blur(4px);
		display: grid;
		place-items: center;
		padding: 24px;
		z-index: 50;
		animation: fadeIn 0.2s ease;
	}

	.modal-dialog {
		width: min(100%, 560px);
		background: var(--surface);
		border-radius: 20px;
		border: 1px solid var(--border);
		box-shadow: 0 16px 36px rgba(11, 16, 28, 0.22);
		overflow: hidden;
		animation: scaleUp 0.2s ease-out;
	}

	.modal-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		padding: 24px 28px 16px;
		border-bottom: 1px solid var(--border);
	}

	.modal-eyebrow {
		color: var(--primary);
		font-size: 11px;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.08em;
	}

	.modal-header h2 {
		margin: 4px 0 0;
		font-size: 20px;
		color: var(--text);
	}

	.modal-meta {
		margin: 4px 0 0;
		color: var(--text-muted);
		font-size: 13px;
	}
	.modal-meta code {
		color: var(--text);
	}

	.btn-close {
		width: 36px;
		height: 36px;
		padding: 0;
		border: none;
		background: #f1f5f9;
		border-radius: 50%;
		font-size: 20px;
		color: var(--text-muted);
		display: grid;
		place-items: center;
	}
	.btn-close:hover {
		background: #e2e8f0;
		color: var(--text);
	}

	.modal-body {
		padding: 24px 28px;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.form-group label {
		font-size: 13px;
		font-weight: 700;
		color: var(--text);
	}

	.required {
		color: var(--status-error-text);
	}

	.form-group input,
	.form-group select,
	.form-group textarea {
		width: 100%;
		padding: 10px 14px;
		border-radius: 10px;
		border: 1px solid var(--border);
		background: #f8fafc;
		font: inherit;
		font-size: 14px;
		color: var(--text);
		transition: border-color 0.15s ease;
	}

	.form-group input:focus,
	.form-group select:focus,
	.form-group textarea:focus {
		border-color: var(--primary);
		background: white;
		outline: 2px solid rgba(3, 75, 253, 0.15);
	}

	.grid-2 {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 14px;
	}

	.modal-notice {
		display: flex;
		align-items: flex-start;
		gap: 10px;
		padding: 12px 14px;
		border-radius: 12px;
		background: #f1f5f9;
		color: var(--text-muted);
		font-size: 12px;
		line-height: 1.5;
	}
	.modal-notice svg {
		color: var(--primary);
		flex-shrink: 0;
		margin-top: 2px;
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 12px;
		padding: 16px 28px 24px;
		border-top: 1px solid var(--border);
	}

	/* Fallback Screen for Other Roles */
	.empty-panel {
		max-width: 580px;
		margin: 40px auto;
		padding: 48px 36px;
		border-radius: 20px;
		background: var(--surface);
		border: 1px solid var(--border);
		text-align: center;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 16px;
		box-shadow: 0 4px 16px rgba(11, 16, 28, 0.06);
	}
	.big-logo {
		width: 64px;
		height: 64px;
		font-size: 26px;
		border-radius: 18px;
	}
	.empty-panel h2 {
		margin: 0;
		font-size: 22px;
	}
	.empty-panel p {
		margin: 0;
		color: var(--text-muted);
		line-height: 1.6;
		font-size: 14px;
	}
	.demo-hints {
		background: #f8fafc;
		border: 1px solid var(--border);
		border-radius: 12px;
		padding: 14px 20px;
		text-align: left;
		width: 100%;
	}
	.demo-hints p {
		font-weight: 700;
		margin-bottom: 6px;
	}
	.demo-hints ul {
		margin: 0;
		padding-left: 20px;
	}
	.demo-hints code {
		font-size: 13px;
		color: var(--primary);
	}

	/* Center Loading */
	.center-loading {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
		justify-content: center;
		align-items: center;
		gap: 16px;
		color: var(--text-muted);
		font-size: 14px;
	}

	.spinner {
		width: 36px;
		height: 36px;
		border: 3px solid #dce3f0;
		border-top-color: var(--primary);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	/* Keyframe Animations */
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
	@keyframes fadeIn {
		from {
			opacity: 0;
		}
		to {
			opacity: 1;
		}
	}
	@keyframes scaleUp {
		from {
			transform: scale(0.96);
			opacity: 0;
		}
		to {
			transform: scale(1);
			opacity: 1;
		}
	}
	@keyframes slideDown {
		from {
			transform: translateY(-8px);
			opacity: 0;
		}
		to {
			transform: translateY(0);
			opacity: 1;
		}
	}

	/* Responsive */
	@media (max-width: 768px) {
		.topbar {
			flex-direction: column;
			align-items: flex-start;
			gap: 16px;
		}
		.user-control {
			width: 100%;
			justify-content: space-between;
		}
		.data-card {
			flex-direction: column;
			align-items: stretch;
		}
		.card-footer {
			justify-content: stretch;
		}
		.card-footer button {
			width: 100%;
		}
		.grid-2 {
			grid-template-columns: 1fr;
		}
		.search-box {
			width: 100%;
		}
		.search-actions {
			width: 100%;
		}
	}
</style>
