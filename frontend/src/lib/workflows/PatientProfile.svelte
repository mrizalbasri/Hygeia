<script lang="ts">
	import { onMount } from 'svelte';
	import { api, ApiError, errorText } from './api.js';

	type Editable = {
		full_name: string;
		birth_date: string;
		sex: string;
		phone: string;
		address: string;
		emergency_contact_name: string;
		emergency_contact_phone: string;
	};
	type Profile = Omit<Editable, 'birth_date'> & {
		birth_date: string | null;
		birth_year: number | null;
		email: string;
		patient_ref: string;
	};
	let { onSaved }: { onSaved: (name: string) => void } = $props();
	const editable = (profile?: Profile): Editable => ({
		full_name: profile?.full_name ?? '',
		birth_date: profile?.birth_date ?? '',
		sex: profile?.sex ?? 'unknown',
		phone: profile?.phone ?? '',
		address: profile?.address ?? '',
		emergency_contact_name: profile?.emergency_contact_name ?? '',
		emergency_contact_phone: profile?.emergency_contact_phone ?? ''
	});
	let profile = $state<Profile | null>(null);
	let values = $state<Editable>(editable());
	let loading = $state(true);
	let saving = $state(false);
	let error = $state('');
	let success = $state('');
	let controller: AbortController | undefined;
	let dirty = $derived(
		profile !== null && JSON.stringify(values) !== JSON.stringify(editable(profile))
	);
	const today = new Date().toISOString().slice(0, 10);

	async function load() {
		controller?.abort();
		const current = new AbortController();
		controller = current;
		loading = true;
		error = '';
		try {
			const result = await api<{ profile: Profile }>(
				'/me/profile',
				'GET',
				undefined,
				current.signal
			);
			if (current.signal.aborted) return;
			profile = result.profile;
			values = editable(result.profile);
		} catch (reason) {
			if (!current.signal.aborted) error = errorText(reason);
		} finally {
			if (!current.signal.aborted) loading = false;
		}
	}
	onMount(() => {
		void load();
		return () => controller?.abort();
	});

	function reset() {
		if (!profile || saving) return;
		values = editable(profile);
		error = '';
		success = '';
	}
	async function save(event: SubmitEvent) {
		event.preventDefault();
		if (saving || !profile || !dirty) return;
		error = '';
		success = '';
		for (const phone of [values.phone.trim(), values.emergency_contact_phone.trim()]) {
			const digits = phone.replace(/\D/g, '').length;
			if (phone && (!/^\+?[\d ().-]+$/.test(phone) || digits < 7 || digits > 15)) {
				error =
					'Nomor telepon harus berisi 7–15 digit. Tanda + di awal, spasi, (), titik, dan - diperbolehkan.';
				return;
			}
		}
		saving = true;
		try {
			const result = await api<{ profile: Profile }>('/me/profile', 'PATCH', { ...values });
			profile = result.profile;
			values = editable(result.profile);
			onSaved(result.profile.full_name);
			success = 'Profil Anda berhasil disimpan.';
		} catch (reason) {
			error = errorText(reason);
			if (
				reason instanceof ApiError &&
				(reason.status === 401 || (reason.status === 403 && reason.code !== 'csrf_invalid'))
			) {
				profile = null;
				values = editable();
			}
		} finally {
			saving = false;
		}
	}
</script>

<div class="profile-heading">
	<h1>Profil saya</h1>
	<p class="muted">
		Kelola data diri dan kontak Anda. Email dan nomor pasien tidak dapat diubah di sini.
	</p>
</div>

{#if error}<p class="wf-error" role="alert">{error}</p>{/if}
{#if success}<p class="wf-success" role="status">{success}</p>{/if}
{#if loading}
	<p class="wf-state" role="status">Memuat profil Anda…</p>
{:else if !profile}
	<div class="wf-state">
		<p>Profil belum dapat ditampilkan.</p>
		<button class="button secondary" onclick={load}>Coba lagi</button>
	</div>
{:else}
	<section class="wf-form-panel profile-panel" aria-label="Profil pasien">
		<div class="profile-identity">
			<label>Nomor pasien<input value={profile.patient_ref} readonly /></label>
			<label>Email<input type="email" value={profile.email} readonly /></label>
		</div>
		<form onsubmit={save} aria-busy={saving}>
			<fieldset disabled={saving}>
				<legend>Data diri</legend>
				<label
					>Nama lengkap<input
						name="full_name"
						autocomplete="name"
						bind:value={values.full_name}
						required
						maxlength="200"
					/></label
				>
				<div class="profile-grid">
					<div>
						<label
							>Tanggal lahir <span class="muted">(opsional)</span><input
								type="date"
								name="birth_date"
								autocomplete="bday"
								min="1900-01-01"
								max={today}
								bind:value={values.birth_date}
								aria-describedby="birth-date-help"
							/></label
						>
						<p id="birth-date-help" class="profile-help">
							{#if profile.birth_year}
								Tahun lahir tercatat: {profile.birth_year}. Mengosongkan tanggal tetap menyimpan
								tahun ini.
							{:else}
								Isi jika Anda mengetahui tanggal lahir lengkap.
							{/if}
						</p>
					</div>
					<div>
						<label for="profile-sex">Jenis kelamin</label>
						<select id="profile-sex" name="sex" bind:value={values.sex}>
							<option value="unknown">Tidak disebutkan</option>
							<option value="female">Perempuan</option>
							<option value="male">Laki-laki</option>
							<option value="other">Lainnya</option>
						</select>
					</div>
				</div>
				<label
					>Nomor telepon <span class="muted">(opsional)</span><input
						type="tel"
						name="phone"
						autocomplete="tel"
						bind:value={values.phone}
						maxlength="32"
						aria-describedby="phone-help"
					/></label
				>
				<p id="phone-help" class="profile-help">Gunakan 7–15 digit, misalnya 081234567890.</p>
				<label
					>Alamat <span class="muted">(opsional)</span><textarea
						name="address"
						autocomplete="street-address"
						bind:value={values.address}
						maxlength="1000"
						rows="3"></textarea></label
				>
			</fieldset>
			<fieldset disabled={saving}>
				<legend>Kontak darurat <span class="muted">(opsional)</span></legend>
				<p>Isi nama dan nomor telepon bersama, atau kosongkan keduanya.</p>
				<div class="profile-grid">
					<label
						>Nama kontak darurat<input
							name="emergency_contact_name"
							bind:value={values.emergency_contact_name}
							maxlength="200"
							required={values.emergency_contact_phone.trim() !== ''}
						/></label
					>
					<label
						>Telepon kontak darurat<input
							type="tel"
							name="emergency_contact_phone"
							bind:value={values.emergency_contact_phone}
							maxlength="32"
							required={values.emergency_contact_name.trim() !== ''}
						/></label
					>
				</div>
			</fieldset>
			<div class="wf-actions">
				<button class="button" type="submit" disabled={saving || !dirty}
					>{saving ? 'Menyimpan profil…' : 'Simpan profil'}</button
				>
				<button class="button secondary" type="button" onclick={reset} disabled={saving || !dirty}
					>Batal perubahan</button
				>
				{#if dirty && !saving}<span class="muted profile-help">Ada perubahan belum disimpan.</span
					>{/if}
			</div>
		</form>
	</section>
{/if}

<style>
	.profile-heading {
		margin-bottom: 24px;
	}
	.profile-heading p {
		max-width: 65ch;
		margin-top: 12px;
	}
	.profile-panel {
		max-width: 800px;
	}
	.profile-identity,
	.profile-grid {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 0 24px;
	}
	.profile-identity {
		padding-bottom: 8px;
		border-bottom: 1px solid var(--border);
	}
	.profile-identity input {
		color: var(--muted);
		background: var(--bg);
	}
	.profile-panel form {
		max-width: none;
	}
	.profile-panel form fieldset {
		border: 0;
		border-radius: 0;
		padding: 0;
		margin: 28px 0;
	}
	.profile-panel legend {
		padding: 0;
		font-size: 18px;
	}
	.profile-panel .profile-help {
		font-size: 12px;
		line-height: 1.7;
	}
	@media (max-width: 600px) {
		.profile-identity,
		.profile-grid {
			grid-template-columns: minmax(0, 1fr);
			gap: 0;
		}
	}
</style>
