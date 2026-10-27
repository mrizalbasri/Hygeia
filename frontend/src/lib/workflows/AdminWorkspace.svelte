<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { roles, type User } from '../auth.js';
	import {
		api,
		ApiError,
		dateText,
		errorText,
		formText,
		menus,
		statusText,
		type Entry,
		type Payload
	} from './api.js';
	import PagedList from './PagedList.svelte';
	import Notifications from './Notifications.svelte';
	import './workflow.css';

	let { user }: { user: User } = $props();

	let menu = $derived(menus.admin);
	let view = $derived(
		menu.some(([id]) => id === page.url.searchParams.get('view'))
			? page.url.searchParams.get('view')!
			: menu[0][0]
	);

	let busy = $state(false),
		ready = $state(false),
		error = $state(''),
		success = $state(''),
		revision = $state(0);

	// Organizations cache for the user creation form and organization names.
	let orgList = $state<{ id: string; name: string; type: string }[]>([]);
	let orgNames = $derived(new Map(orgList.map((org) => [org.id, org.name])));
	const orgLabel = (id?: string | null) => (id ? orgNames.get(id) || id : '');

	// Audit verification result
	let verifyResult = $state<{
		chain: string;
		intact: boolean;
		events: number;
		first_mismatch_seq?: number;
	} | null>(null);

	// Modals
	let renameOrgDialog: HTMLDialogElement | null = $state(null),
		selectedOrg = $state<Entry | null>(null),
		renameOrgValue = $state('');

	let renameUserDialog: HTMLDialogElement | null = $state(null),
		selectedUser = $state<Entry | null>(null),
		renameUserValue = $state('');

	let breakglassDialog: HTMLDialogElement | null = $state(null),
		selectedBreakglass = $state<Entry | null>(null),
		reviewDecision = $state<'appropriate' | 'needs_followup'>('appropriate'),
		reviewNote = $state('');

	// Form states
	let newUserRole = $state('doctor'),
		newUserOrgId = $state('');

	onMount(() => {
		ready = true;
		void loadOrganizations();
	});

	$effect(() => {
		const currentView = view;
		void currentView;
		error = '';
		success = '';
		verifyResult = null;
	});

	async function loadOrganizations() {
		try {
			const res = await api<{ organizations: { id: string; name: string; type: string }[] }>(
				'/admin/organizations?limit=100'
			);
			orgList = res.organizations || [];
		} catch {
			orgList = [];
		}
	}

	async function run(path: string, body: Payload, method: 'POST' | 'PATCH' = 'POST') {
		if (busy) return false;
		busy = true;
		error = '';
		success = '';
		try {
			await api(path, method, body);
			revision++;
			success = 'Tindakan administratif berhasil diproses.';
			void loadOrganizations();
			return true;
		} catch (reason) {
			error = errorText(reason);
			if (reason instanceof ApiError && reason.status === 409) {
				revision++;
				error += ' Terjadi konflik data atau status telah berubah.';
			}
			return false;
		} finally {
			busy = false;
		}
	}

	// Create Organization
	async function createOrganization(event: SubmitEvent) {
		event.preventDefault();
		const form = event.currentTarget as HTMLFormElement,
			data = new FormData(form);
		const name = formText(data, 'name');
		const type = formText(data, 'type');
		if (!name || !type) return;

		const ok = await run('/admin/organizations', { name, type }, 'POST');
		if (ok) {
			form.reset();
			success = `Organisasi ${name} berhasil dibuat.`;
		}
	}

	// Rename Organization
	function openRenameOrg(item: Entry) {
		selectedOrg = item;
		renameOrgValue = item.name || '';
		renameOrgDialog?.showModal();
	}

	async function submitRenameOrg(event: SubmitEvent) {
		event.preventDefault();
		if (!selectedOrg || busy) return;
		const name = renameOrgValue.trim();
		if (!name) return;
		const id = selectedOrg.id;
		renameOrgDialog?.close();
		const ok = await run(`/admin/organizations/${id}`, { name }, 'PATCH');
		if (ok) {
			selectedOrg = null;
			success = `Nama organisasi berhasil diperbarui menjadi ${name}.`;
		}
	}

	// Create User
	async function createUser(event: SubmitEvent) {
		event.preventDefault();
		const form = event.currentTarget as HTMLFormElement,
			data = new FormData(form);
		const email = formText(data, 'email').toLowerCase();
		const full_name = formText(data, 'full_name');
		const password = String(data.get('password') || '');
		const role = newUserRole;

		const body: Payload = { email, full_name, password, role };
		if (role === 'doctor' || role === 'lab' || role === 'pharmacy') {
			if (!newUserOrgId) {
				error = 'Organisasi wajib dipilih untuk peran dokter, lab, atau apotek.';
				return;
			}
			body.organization_id = newUserOrgId;
		}

		const ok = await run('/admin/users', body, 'POST');
		if (ok) {
			form.reset();
			newUserRole = 'doctor';
			newUserOrgId = '';
			success = `Akun ${email} (${roles[role as keyof typeof roles]?.label || role}) berhasil dibuat.`;
		}
	}

	// Rename User
	function openRenameUser(item: Entry) {
		selectedUser = item;
		renameUserValue = item.full_name || item.name || '';
		renameUserDialog?.showModal();
	}

	async function submitRenameUser(event: SubmitEvent) {
		event.preventDefault();
		if (!selectedUser || busy) return;
		const full_name = renameUserValue.trim();
		if (!full_name) return;
		const id = selectedUser.id;
		renameUserDialog?.close();
		const ok = await run(`/admin/users/${id}`, { full_name }, 'PATCH');
		if (ok) {
			selectedUser = null;
			success = `Nama akun berhasil diperbarui menjadi ${full_name}.`;
		}
	}

	// Toggle User Active/Disabled
	async function toggleUserDisabled(item: Entry) {
		const willDisable = !item.disabled_at;
		const actionName = willDisable ? 'dinonaktifkan' : 'diaktifkan kembali';
		const ok = await run(`/admin/users/${item.id}`, { disabled: willDisable }, 'PATCH');
		if (ok) {
			success = `Akun ${item.full_name || item.email} telah berhasil ${actionName}.`;
		}
	}

	// Verify Audit Chain
	async function verifyChain(chainId: string) {
		busy = true;
		error = '';
		verifyResult = null;
		try {
			const res = await api<{
				chain: string;
				events: number;
				intact: boolean;
				first_mismatch_seq?: number;
			}>(`/audit/verify?chain=${encodeURIComponent(chainId)}`);
			verifyResult = res;
		} catch (reason) {
			error = errorText(reason);
		} finally {
			busy = false;
		}
	}

	// Review Break-Glass
	function openBreakglassReview(item: Entry) {
		selectedBreakglass = item;
		reviewDecision = 'appropriate';
		reviewNote = '';
		breakglassDialog?.showModal();
	}

	async function submitBreakglassReview(event: SubmitEvent) {
		event.preventDefault();
		if (!selectedBreakglass || busy) return;
		const note = reviewNote.trim();
		if (note.length < 10 || note.length > 1000) {
			error = 'Catatan peninjauan wajib antara 10 hingga 1000 karakter.';
			return;
		}
		const id = selectedBreakglass.id;
		breakglassDialog?.close();
		const ok = await run(`/admin/break-glass/${id}`, { decision: reviewDecision, note }, 'PATCH');
		if (ok) {
			selectedBreakglass = null;
			success = 'Peninjauan akses darurat berhasil disimpan.';
		}
	}
</script>

<div class="wf-heading">
	<p class="muted">Admin Platform · {user.full_name}</p>
	<h1>Kelola Platform Hygeia</h1>
	<h2>{menu.find(([id]) => id === view)?.[1] || 'Organisasi'}</h2>
</div>

{#if error}<p class="wf-error" role="alert">{error}</p>{/if}
{#if success}<p class="wf-success" role="status">{success}</p>{/if}
{#if busy}<p role="status">Memproses tindakan administratif…</p>{/if}

{#key view}
	<fieldset class="wf-content" disabled={busy || !ready} aria-busy={busy || !ready}>
		{#if view === 'organizations'}
			<details class="wf-form-panel">
				<summary>Tambah organisasi baru</summary>
				<form onsubmit={createOrganization}>
					<label>
						Nama organisasi *
						<input
							name="name"
							required
							maxlength="200"
							placeholder="Contoh: Klinik Pratama Sehat, Lab Sentra, Apotek Kimia Farma"
							disabled={busy}
						/>
					</label>
					<label>
						Tipe organisasi *
						<select name="type" required disabled={busy}>
							<option value="clinic">Klinik (Clinic)</option>
							<option value="laboratory">Laboratorium (Laboratory)</option>
							<option value="pharmacy">Apotek (Pharmacy)</option>
						</select>
					</label>
					<div class="wf-actions">
						<button type="submit" class="button primary" disabled={busy}>
							Daftarkan organisasi
						</button>
					</div>
				</form>
			</details>

			<PagedList
				path="/admin/organizations"
				listKey="organizations"
				{revision}
				empty="Belum ada organisasi yang terdaftar."
			>
				{#snippet children(item)}
					<div class="wf-row-heading">
						<div>
							<h2>{item.name}</h2>
							<p>
								Tipe: <strong>{item.type}</strong> · ID: <code>{item.id}</code>
							</p>
						</div>
						<span class="health-status">
							{item.type === 'clinic'
								? 'Klinik'
								: item.type === 'laboratory'
									? 'Laboratorium'
									: 'Apotek'}
						</span>
					</div>
					<div class="wf-actions">
						<button
							type="button"
							class="button secondary"
							disabled={busy}
							onclick={() => openRenameOrg(item)}
						>
							Ubah nama organisasi
						</button>
					</div>
				{/snippet}
			</PagedList>
		{:else if view === 'users'}
			<details class="wf-form-panel">
				<summary>Buat akun pengguna baru</summary>
				<form onsubmit={createUser}>
					<label>
						Email pengguna *
						<input
							name="email"
							type="email"
							required
							maxlength="254"
							placeholder="nama@domain.test"
							disabled={busy}
						/>
					</label>
					<label>
						Nama lengkap *
						<input
							name="full_name"
							required
							maxlength="200"
							placeholder="Contoh: dr. Maya Putri, Sp.PD"
							disabled={busy}
						/>
					</label>
					<label>
						Kata sandi * (minimal 12 karakter)
						<input
							name="password"
							type="password"
							required
							minlength="12"
							maxlength="1024"
							placeholder="Minimal 12 karakter"
							disabled={busy}
						/>
					</label>
					<label>
						Peran (Role) *
						<select name="role" bind:value={newUserRole} required disabled={busy}>
							<option value="doctor">Dokter</option>
							<option value="patient">Pasien</option>
							<option value="lab">Petugas Laboratorium</option>
							<option value="pharmacy">Petugas Apotek</option>
							<option value="admin">Administrator Platform</option>
						</select>
					</label>

					{#if newUserRole === 'doctor' || newUserRole === 'lab' || newUserRole === 'pharmacy'}
						{@const expectedType =
							newUserRole === 'doctor'
								? 'clinic'
								: newUserRole === 'lab'
									? 'laboratory'
									: 'pharmacy'}
						{@const availableOrgs = orgList.filter((org) => org.type === expectedType)}
						<label>
							Organisasi {roles[newUserRole as keyof typeof roles]?.label} *
							<select name="organization_id" bind:value={newUserOrgId} required disabled={busy}>
								<option value="">Pilih {expectedType} yang sesuai</option>
								{#each availableOrgs as org (org.id)}
									<option value={org.id}>{org.name}</option>
								{/each}
							</select>
						</label>
						{#if availableOrgs.length === 0}
							<p class="wf-warning" role="alert">
								Belum ada organisasi bertipe {expectedType}. Daftarkan organisasi terlebih dahulu di
								menu Organisasi.
							</p>
						{/if}
					{/if}

					<div class="wf-actions">
						<button type="submit" class="button primary" disabled={busy}> Daftarkan akun </button>
					</div>
				</form>
			</details>

			<PagedList path="/admin/users" listKey="users" {revision} empty="Belum ada akun pengguna.">
				{#snippet children(item)}
					<div class="wf-row-heading">
						<div>
							<h2>{item.full_name || item.name}</h2>
							<p>
								Email: <strong>{item.email}</strong> · Peran:
								<strong>{roles[item.role as keyof typeof roles]?.label || item.role}</strong>
								{#if item.patient_ref}
									· Kode pasien: <code>{item.patient_ref}</code>
								{/if}
								{#if item.organization_id}
									· Organisasi: <em>{orgLabel(item.organization_id)}</em>
								{/if}
							</p>
						</div>
						<span class="health-status">
							{item.disabled_at ? 'Dinonaktifkan' : 'Aktif'}
						</span>
					</div>
					<div class="wf-actions">
						<button
							type="button"
							class="button secondary"
							disabled={busy}
							onclick={() => openRenameUser(item)}
						>
							Ubah nama
						</button>
						{#if item.id !== user.id}
							<button
								type="button"
								class="button secondary"
								disabled={busy}
								onclick={() => toggleUserDisabled(item)}
							>
								{item.disabled_at ? 'Aktifkan akun' : 'Nonaktifkan akun'}
							</button>
						{/if}
					</div>
				{/snippet}
			</PagedList>
		{:else if view === 'audit'}
			{#if verifyResult}
				<section class={verifyResult.intact ? 'wf-success' : 'wf-error'} role="status">
					<h3>Hasil Verifikasi Integritas Rantai Audit HMAC</h3>
					<p><strong>Rantai Pasien ID:</strong> <code>{verifyResult.chain}</code></p>
					<p><strong>Jumlah Kejadian (Events):</strong> {verifyResult.events}</p>
					{#if verifyResult.intact}
						<p>
							Status Rantai: <strong>UTUH & TERVERIFIKASI (HMAC VALID)</strong>. Tidak ditemukan
							manipulasi atau pemalsuan data pada seluruh riwayat audit pasien.
						</p>
					{:else}
						<p>
							Status Rantai: <strong>PERINGATAN: INTEGRITAS RUSAK!</strong> Terdapat ketidakcocokan
							tanda tangan HMAC pada urutan ke-<strong>{verifyResult.first_mismatch_seq}</strong>.
						</p>
					{/if}
					<button type="button" class="button secondary" onclick={() => (verifyResult = null)}>
						Tutup hasil verifikasi
					</button>
				</section>
			{/if}

			<PagedList
				path="/admin/audit-chains"
				listKey="chains"
				{revision}
				empty="Belum ada rantai audit pasien yang tercatat."
			>
				{#snippet children(item)}
					<div class="wf-row-heading">
						<div>
							<h2>Pasien: {item.patient_ref}</h2>
							<p>Chain ID: <code>{item.chain_id || item.id}</code></p>
							<p>Total peristiwa audit tercatat: <strong>{item.events} event</strong></p>
						</div>
						<span class="health-status">Rantai HMAC</span>
					</div>
					<div class="wf-actions">
						<button
							type="button"
							class="button primary"
							disabled={busy}
							onclick={() => verifyChain(item.chain_id || item.id)}
						>
							Verifikasi Integritas HMAC
						</button>
					</div>
				{/snippet}
			</PagedList>
		{:else if view === 'breakglass'}
			<PagedList
				path="/admin/break-glass"
				listKey="sessions"
				{revision}
				empty="Belum ada catatan pembukaan akses darurat."
			>
				{#snippet children(item)}
					<div class="wf-row-heading">
						<div>
							<h2>Akses Darurat: Pasien {item.patient_ref}</h2>
							<p>
								Dokter ID: <code>{item.doctor_user_id}</code>
								{#if item.organization_id}
									· Organisasi: <strong>{orgLabel(item.organization_id)}</strong>
								{/if}
							</p>
							<p>
								Alasan: <strong>{statusText[item.reason_code || ''] || item.reason_code}</strong>
							</p>
							<p>Keterangan dokter: <em>"{item.reason_text}"</em></p>
							<p class="muted">
								Waktu mulai: {dateText(item.starts_at)} · Berakhir: {dateText(item.expires_at)}
							</p>
						</div>
						<span class="health-status">
							{item.review_decision
								? statusText[item.review_decision] || item.review_decision
								: 'Menunggu tinjauan'}
						</span>
					</div>

					{#if item.review_decision}
						<div class="wf-form-panel">
							<p>
								<strong>Keputusan Tinjauan:</strong>
								{statusText[item.review_decision] || item.review_decision}
							</p>
							{#if item.review_note}
								<p><strong>Catatan Administrator:</strong> {item.review_note}</p>
							{/if}
						</div>
					{:else}
						<div class="wf-actions">
							<button
								type="button"
								class="button primary"
								disabled={busy}
								onclick={() => openBreakglassReview(item)}
							>
								Tinjau akses darurat ini
							</button>
						</div>
					{/if}
				{/snippet}
			</PagedList>
		{:else if view === 'activity'}
			<PagedList
				path="/admin/activity"
				listKey="activity"
				{revision}
				empty="Belum ada log aktivitas administratif."
			>
				{#snippet children(item)}
					<div class="wf-row-heading">
						<div>
							<h2>Aktivitas: <code>{item.action}</code></h2>
							<p>
								Pelaku ID: <code>{item.actor_id}</code>
								{#if item.resource_id}
									· Target Resource ID: <code>{item.resource_id}</code>
								{/if}
							</p>
						</div>
						<span class="health-status">Audit Admin</span>
					</div>
					<p class="muted">Waktu: {dateText(item.occurred_at)}</p>
				{/snippet}
			</PagedList>
		{:else if view === 'notifications'}
			<Notifications {run} {busy} {revision} />
		{/if}
	</fieldset>
{/key}

<!-- Modal: Ubah Nama Organisasi -->
<dialog
	bind:this={renameOrgDialog}
	class="wf-dialog"
	onclose={() => (selectedOrg = null)}
	aria-labelledby="rename-org-title"
>
	<header>
		<h2 id="rename-org-title">Ubah Nama Organisasi</h2>
		<button
			type="button"
			class="button secondary"
			disabled={busy}
			onclick={() => renameOrgDialog?.close()}
		>
			Batal
		</button>
	</header>
	{#if selectedOrg}
		<form onsubmit={submitRenameOrg}>
			<label>
				Nama baru organisasi
				<input name="name" required maxlength="200" bind:value={renameOrgValue} disabled={busy} />
			</label>
			<div class="wf-actions">
				<button type="submit" class="button primary" disabled={busy || !renameOrgValue.trim()}>
					{busy ? 'Menyimpan…' : 'Simpan nama baru'}
				</button>
				<button
					type="button"
					class="button secondary"
					disabled={busy}
					onclick={() => renameOrgDialog?.close()}
				>
					Batal
				</button>
			</div>
		</form>
	{/if}
</dialog>

<!-- Modal: Ubah Nama User -->
<dialog
	bind:this={renameUserDialog}
	class="wf-dialog"
	onclose={() => (selectedUser = null)}
	aria-labelledby="rename-user-title"
>
	<header>
		<h2 id="rename-user-title">Ubah Nama Akun</h2>
		<button
			type="button"
			class="button secondary"
			disabled={busy}
			onclick={() => renameUserDialog?.close()}
		>
			Batal
		</button>
	</header>
	{#if selectedUser}
		<form onsubmit={submitRenameUser}>
			<label>
				Nama lengkap baru
				<input
					name="full_name"
					required
					maxlength="200"
					bind:value={renameUserValue}
					disabled={busy}
				/>
			</label>
			<div class="wf-actions">
				<button type="submit" class="button primary" disabled={busy || !renameUserValue.trim()}>
					{busy ? 'Menyimpan…' : 'Simpan nama baru'}
				</button>
				<button
					type="button"
					class="button secondary"
					disabled={busy}
					onclick={() => renameUserDialog?.close()}
				>
					Batal
				</button>
			</div>
		</form>
	{/if}
</dialog>

<!-- Modal: Tinjau Akses Darurat -->
<dialog
	bind:this={breakglassDialog}
	class="wf-dialog"
	onclose={() => (selectedBreakglass = null)}
	aria-labelledby="review-breakglass-title"
>
	<header>
		<h2 id="review-breakglass-title">Tinjau Akses Darurat Dokter</h2>
		<button
			type="button"
			class="button secondary"
			disabled={busy}
			onclick={() => breakglassDialog?.close()}
		>
			Batal
		</button>
	</header>
	{#if selectedBreakglass}
		<p class="muted">
			Dokter ID: <code>{selectedBreakglass.doctor_user_id}</code>
			{#if selectedBreakglass.organization_id}
				· Organisasi: <strong>{orgLabel(selectedBreakglass.organization_id)}</strong>
			{/if}
			· Pasien: <strong>{selectedBreakglass.patient_ref}</strong>
		</p>
		<p>Alasan darurat: <em>"{selectedBreakglass.reason_text}"</em></p>
		<form onsubmit={submitBreakglassReview}>
			<label>
				Keputusan Peninjauan *
				<select name="decision" bind:value={reviewDecision} required disabled={busy}>
					<option value="appropriate">Sesuai Prosedur Darurat (Appropriate)</option>
					<option value="needs_followup">Perlu Tindak Lanjut / Investigasi (Needs Followup)</option>
				</select>
			</label>
			<label>
				Catatan Peninjauan Administrator * (10–1000 karakter)
				<textarea
					name="note"
					required
					minlength="10"
					maxlength="1000"
					placeholder="Catatan hasil peninjauan akuntabilitas pembukaan akses darurat"
					bind:value={reviewNote}
					disabled={busy}></textarea>
			</label>
			<div class="wf-actions">
				<button
					type="submit"
					class="button primary"
					disabled={busy || reviewNote.trim().length < 10}
				>
					{busy ? 'Menyimpan…' : 'Simpan Keputusan Peninjauan'}
				</button>
				<button
					type="button"
					class="button secondary"
					disabled={busy}
					onclick={() => breakglassDialog?.close()}
				>
					Batal
				</button>
			</div>
		</form>
	{/if}
</dialog>
