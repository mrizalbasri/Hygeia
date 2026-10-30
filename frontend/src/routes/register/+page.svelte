<script lang="ts">
	import { onMount } from 'svelte';
	import ClerkPanel from '#lib/ClerkPanel.svelte';
	import '#lib/auth-layout.css';
	import type { PageProps } from './$types';
	let { data }: PageProps = $props();
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { authMutation, type User } from '#lib/auth.js';

	let fullName = $state(''),
		email = $state(''),
		password = $state(''),
		confirmation = $state('');
	let birthYear = $state<number | undefined>(),
		sex = $state('unknown');
	let pending = $state(false),
		ready = $state(false),
		message = $state('');
	const currentYear = new Date().getFullYear();
	onMount(() => {
		ready = true;
	});

	async function register(event: SubmitEvent) {
		event.preventDefault();
		if (pending) return;
		message = '';
		if (!fullName.trim()) {
			message = 'Nama lengkap wajib diisi.';
			return;
		}
		const passwordBytes = new TextEncoder().encode(password).length;
		if (passwordBytes < 12 || passwordBytes > 1024) {
			message =
				'Password harus berukuran 12–1024 byte. Huruf beraksen dan simbol dapat memakai lebih dari satu byte.';
			return;
		}
		if (password !== confirmation) {
			message = 'Konfirmasi password belum sama. Periksa kembali kedua kolom password.';
			return;
		}
		pending = true;
		try {
			const response = await authMutation('register', {
				email: email.trim(),
				password,
				full_name: fullName.trim(),
				birth_year: birthYear,
				sex
			});
			const body = (await response.json()) as { user?: User; error?: { message?: string } };
			if (!response.ok || body.user?.role !== 'patient') {
				message =
					body.error?.message || 'Pendaftaran belum berhasil. Periksa isian lalu coba lagi.';
				return;
			}
			password = '';
			confirmation = '';
			await goto(resolve('/dashboard/[role]', { role: 'patient' }), {
				replace: true,
				refreshAll: true
			});
		} catch {
			message =
				'Layanan belum dapat dihubungi. Coba lagi sebentar. Jika akun sudah dibuat, masuk dengan email dan password Anda.';
		} finally {
			pending = false;
		}
	}
</script>

<svelte:head
	><title>Daftar pasien · Hygeia</title><meta
		name="description"
		content="Buat akun pasien Hygeia untuk mengikuti perawatan dan mengatur izin akses data."
	/></svelte:head
>

<main class="login-shell register-shell auth-shell">
	<section class="login-story" aria-label="Tentang akun pasien">
		<a class="brand" href={resolve('/')} title="Kembali ke beranda Hygeia"
			><img src="/logo-light.webp" alt="Hygeia" width="44" height="44" /><span>Hygeia</span></a
		>
		<div class="story-body">
			<h1>Mulai perjalanan<br /> perawatan&nbsp;<span>Anda.</span></h1>
			<p class="story-copy">
				Buat akun pasien untuk mengikuti pemeriksaan, menerima resep, dan mengatur siapa yang dapat
				mengakses data Anda.
			</p>
			<div class="brand-showcase consultation-showcase" aria-hidden="true">
				<img src="/patient-care.webp" alt="" width="1536" height="1024" decoding="async" />
			</div>
		</div>
		<p class="story-footer">Akun dokter, laboratorium, dan apotek didaftarkan oleh pengelola.</p>
	</section>
	<section class="login-form-side" aria-labelledby="register-title">
		<div class="auth-top-action">
			<a class="back-link" href={resolve('/')} title="Kembali ke beranda">
				<svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true"
					><path
						d="m14 18-6-6 6-6"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					/></svg
				>
				<span>Kembali ke beranda</span>
			</a>
		</div>
		<div class="login-form-wrap register-form-wrap">
			{#if data.authProvider === 'clerk'}
				<h2 id="register-title" class="sr-only">Daftar sebagai pasien</h2>
			{:else}
				<h2 id="register-title">Daftar sebagai pasien</h2>
				<p class="muted intro">Setelah mendaftar, Anda langsung masuk ke ruang pasien.</p>
			{/if}
			{#if data.authProvider === 'clerk'}{#if data.clerkPublishableKey}<ClerkPanel
						publishableKey={data.clerkPublishableKey}
						mode="register"
					/>{:else}<p role="alert">Konfigurasi autentikasi belum tersedia.</p>{/if}{:else}<form
					method="POST"
					onsubmit={register}
					aria-busy={pending}
				>
					<label for="full-name">Nama lengkap</label>
					<input
						id="full-name"
						name="full_name"
						autocomplete="name"
						bind:value={fullName}
						maxlength="200"
						required
						disabled={pending}
					/>
					<label for="email">Email</label>
					<input
						id="email"
						name="email"
						type="email"
						autocomplete="username"
						bind:value={email}
						maxlength="254"
						required
						disabled={pending}
					/>
					<div class="registration-demographics">
						<div>
							<label for="birth-year">Tahun lahir</label><input
								id="birth-year"
								name="birth_year"
								type="number"
								inputmode="numeric"
								min="1900"
								max={currentYear}
								step="1"
								bind:value={birthYear}
								required
								disabled={pending}
							/>
						</div>
						<div>
							<label for="sex">Jenis kelamin</label><select
								id="sex"
								name="sex"
								bind:value={sex}
								disabled={pending}
								><option value="unknown">Tidak disebutkan</option><option value="female"
									>Perempuan</option
								><option value="male">Laki-laki</option><option value="other">Lainnya</option
								></select
							>
						</div>
					</div>
					<label for="password">Password</label>
					<input
						id="password"
						name="password"
						type="password"
						autocomplete="new-password"
						aria-describedby="password-hint"
						bind:value={password}
						maxlength="1024"
						required
						disabled={pending}
					/>
					<p id="password-hint" class="auth-hint">
						Gunakan minimal 12 karakter. Password tidak boleh lebih dari 1024 byte.
					</p>
					<label for="confirmation">Konfirmasi password</label>
					<input
						id="confirmation"
						name="confirmation"
						type="password"
						autocomplete="new-password"
						bind:value={confirmation}
						maxlength="1024"
						required
						disabled={pending}
					/>
					<div class="form-message" aria-live="polite">
						{#if message}<p role="alert">{message}</p>{/if}
					</div>
					<button class="button primary login-submit" type="submit" disabled={pending || !ready}
						>{pending ? 'Mendaftarkan akun…' : 'Buat akun pasien'}</button
					>
				</form>{/if}
			{#if data.authProvider !== 'clerk'}<p class="auth-switch">
					Sudah punya akun? <a href={resolve('/login')}>Masuk</a>
				</p>{/if}
		</div>
	</section>
</main>
