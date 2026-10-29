<script lang="ts">
	import DirectoryPicker from './DirectoryPicker.svelte';
	import type { Run } from './api.js';
	let { id, run, busy }: { id: string; run: Run; busy: boolean } = $props();
	let role = $state('doctor'),
		organization = $state('');
	const kind = $derived(
		({ doctor: 'clinic', lab: 'laboratory', pharmacy: 'pharmacy' } as Record<string, string>)[role]
	);
</script>

<details>
	<summary>Tetapkan peran staf</summary>
	<p>
		Hanya akun Clerk baru tanpa riwayat perawatan/audit yang dapat dialihkan. Profil pasien kosong
		akan dihapus; akun dengan data pasien tetap dipertahankan. Staf harus masuk ulang dengan MFA.
	</p>
	<form
		onsubmit={async (event) => {
			event.preventDefault();
			if (
				confirm(
					'Alihkan akun kosong ini ke peran staf yang dipilih? Akses baru akan mengikuti peran dan organisasi tersebut.'
				)
			)
				await run(`/admin/users/${id}/staff-role`, {
					role,
					organization_id: role === 'admin' ? '' : organization
				});
		}}
	>
		<label
			>Peran staf<select bind:value={role} onchange={() => (organization = '')} disabled={busy}
				><option value="doctor">Dokter</option><option value="lab">Laboratorium</option><option
					value="pharmacy">Apotek</option
				><option value="admin">Administrator</option></select
			></label
		>
		{#if role !== 'admin'}{#key role}<DirectoryPicker
					path="/admin/organizations"
					listKey="organizations"
					organizationType={kind}
					label="Organisasi staf"
					bind:value={organization}
				/>{/key}{/if}
		<button class="button secondary" disabled={busy || (role !== 'admin' && !organization)}
			>Konfirmasi peran staf</button
		>
	</form>
</details>
