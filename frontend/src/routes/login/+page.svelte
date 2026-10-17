<script lang="ts">
	import { goto } from '$app/navigation';
	import { login } from '../../lib/api';

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);

	async function submit() {
		error = '';
		loading = true;
		try {
			await login(email.trim(), password);
			await goto('/');
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Login gagal.';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Login · Hygeia</title>
</svelte:head>

<main class="login-shell">
	<section class="login-card" aria-labelledby="login-title">
		<div class="brand-mark" aria-hidden="true">H+</div>
		<p class="eyebrow">Hygeia</p>
		<h1 id="login-title">Masuk ke ruang kerja Anda</h1>
		<p class="muted">Kelola permintaan laboratorium dan resep dengan akses yang tercatat.</p>

		<form onsubmit={(event) => { event.preventDefault(); submit(); }}>
			<label>
				Email
				<input type="email" bind:value={email} autocomplete="email" required />
			</label>
			<label>
				Password
				<input type="password" bind:value={password} autocomplete="current-password" required />
			</label>
			{#if error}<p class="error" role="alert">{error}</p>{/if}
			<button class="primary" type="submit" disabled={loading}>
				{loading ? 'Memeriksa...' : 'Masuk'}
			</button>

			<div class="quick-accounts">
				<span class="quick-label">Isi Cepat Akun Demo:</span>
				<div class="quick-buttons">
					<button
						type="button"
						class="chip-btn"
						onclick={() => {
							email = 'lab@demo.hygeia.test';
						}}
					>
						🔬 Staf Lab
					</button>
					<button
						type="button"
						class="chip-btn"
						onclick={() => {
							email = 'apotek@demo.hygeia.test';
						}}
					>
						💊 Staf Apotek
					</button>
				</div>
			</div>
		</form>
	</section>
</main>

<style>
	:global(body) { margin: 0; }
	.login-shell { min-height: 100vh; display: grid; place-items: center; padding: 24px; background: #0b101c; color: #f7f9fe; }
	.login-card { width: min(100%, 420px); padding: 40px; border: 1px solid #2b3345; border-radius: 24px; background: #111827; box-shadow: 0 8px 24px rgb(0 0 0 / 18%); }
	.brand-mark { width: 48px; height: 48px; display: grid; place-items: center; border-radius: 14px; background: linear-gradient(90deg, #146afe, #00abef, #1aebfd); color: white; font-weight: 700; }
	.eyebrow { margin: 20px 0 8px; color: #6fa3ff; font-weight: 700; letter-spacing: .08em; text-transform: uppercase; }
	h1 { margin: 0; font-size: 28px; line-height: 1.25; }
	.muted { color: #9aa6bd; line-height: 1.5; }
	form { display: grid; gap: 16px; margin-top: 28px; }
	label { display: grid; gap: 8px; color: #f7f9fe; font-size: 14px; font-weight: 600; }
	input { min-height: 44px; box-sizing: border-box; padding: 0 12px; border: 1px solid #2b3345; border-radius: 12px; background: #161d2e; color: #f7f9fe; font: inherit; }
	input:focus { outline: 2px solid #1aebfd; outline-offset: 2px; }
	.primary { min-height: 44px; border: 0; border-radius: 12px; background: #146afe; color: white; font-weight: 700; cursor: pointer; }
	.primary:disabled { opacity: .6; cursor: wait; }
	.error { margin: 0; color: #f87171; font-size: 14px; }
	.quick-accounts { margin-top: 10px; padding-top: 16px; border-top: 1px solid #2b3345; display: flex; flex-direction: column; gap: 8px; }
	.quick-label { font-size: 12px; color: #9aa6bd; font-weight: 600; text-transform: uppercase; letter-spacing: 0.04em; }
	.quick-buttons { display: flex; gap: 8px; }
	.chip-btn { flex: 1; min-height: 36px; padding: 0 10px; border: 1px solid #2b3345; border-radius: 10px; background: #161d2e; color: #f7f9fe; font-size: 12px; font-weight: 600; cursor: pointer; transition: all 0.15s ease; }
	.chip-btn:hover { border-color: #1aebfd; background: #1f293d; }
</style>
