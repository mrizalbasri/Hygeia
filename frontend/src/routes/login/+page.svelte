<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { authMutation, type User } from '#lib/auth.js';
	let email = $state('');
	let password = $state('');
	let pending = $state(false);
	let ready = $state(false);
	let message = $state('');
	onMount(() => {
		ready = true;
	});

	async function login(event: SubmitEvent) {
		event.preventDefault();
		if (pending) return;
		pending = true;
		message = '';
		try {
			const response = await authMutation('login', { email, password });
			const body = (await response.json()) as { user?: User; error?: { message: string } };
			if (!response.ok || !body.user) {
				message = body.error?.message || 'Tidak dapat masuk. Coba lagi.';
				return;
			}
			password = '';
			await goto(resolve('/dashboard/[role]', { role: body.user.role }), {
				replace: true,
				refreshAll: true
			});
		} catch {
			message = 'Layanan belum dapat dihubungi. Coba lagi sebentar.';
		} finally {
			pending = false;
		}
	}
</script>

<svelte:head
	><title>Masuk · Hygeia</title><meta
		name="description"
		content="Masuk ke Hygeia untuk perjalanan perawatan yang terhubung dan terkendali."
	/></svelte:head
>

<main class="login-shell">
	<section class="login-story" aria-label="Tentang Hygeia">
		<a class="brand" href={resolve('/login')}
			><img src="/logo-light.png" alt="" width="44" height="44" /><span>Hygeia</span></a
		>
		<div class="story-body">
			<p class="eyebrow">TERHUBUNG. ATAS IZIN ANDA.</p>
			<h1>Perawatan terhubung.<br /> Kendali tetap<br /> <span>di tangan Anda.</span></h1>
			<p class="story-copy">
				Pasien, dokter, laboratorium, dan apotek. Satu perjalanan dengan akses data yang jelas.
			</p>
			<div class="brand-showcase consultation-showcase" aria-hidden="true">
				<img src="/care-consultation.webp" alt="" width="1536" height="1024" decoding="async" />
			</div>
			<div class="care-path" aria-label="Alur pasien, dokter, laboratorium, dan apotek">
				{#each ['Pasien', 'Dokter', 'Lab', 'Apotek'] as step, index (step)}<div>
						<span class="path-node">0{index + 1}</span><span>{step}</span>
					</div>{/each}
			</div>
		</div>
		<p class="story-footer">Diizinkan pasien. Dibatasi seperlunya. Tercatat setiap aksesnya.</p>
	</section>
	<section class="login-form-side" aria-labelledby="login-title">
		<div class="login-form-wrap">
			<p class="eyebrow muted">SELAMAT DATANG DI HYGEIA</p>
			<h2 id="login-title">Masuk ke ruang Anda</h2>
			<p class="muted intro">Lanjutkan perjalanan perawatan dengan akun Anda.</p>
			<form method="POST" onsubmit={login} aria-busy={pending}>
				<label for="email">Email</label>
				<input
					id="email"
					name="email"
					type="email"
					autocomplete="username"
					placeholder="nama@email.com"
					bind:value={email}
					maxlength="254"
					required
					disabled={pending}
				/>
				<label for="password">Password</label>
				<input
					id="password"
					name="password"
					type="password"
					autocomplete="current-password"
					bind:value={password}
					maxlength="1024"
					required
					disabled={pending}
				/>
				<div class="form-message" aria-live="polite">
					{#if message}<p role="alert">{message}</p>{/if}
				</div>
				<button class="button primary login-submit" type="submit" disabled={pending || !ready}
					>{pending ? 'Sedang masuk…' : 'Masuk'}<span aria-hidden="true">→</span></button
				>
			</form>
			<p class="auth-switch">
				Belum punya akun pasien? <a href={resolve('/register')}>Daftar sebagai pasien</a>
			</p>
			<div class="demo-notice">
				<span class="notice-dot" aria-hidden="true"></span>
				<p>
					<strong>Lingkungan demo</strong><br />Gunakan akun demo yang diberikan pengelola. Seluruh
					data bersifat sintetis.
				</p>
			</div>
		</div>
		<a class="form-footer button secondary" href={resolve('/')}>Kembali ke beranda</a>
	</section>
</main>
