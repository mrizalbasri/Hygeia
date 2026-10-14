<script lang="ts">
	import PagedList from './PagedList.svelte';
	import { api, errorText, dateText } from './api.js';
	let { audit, revision }: { audit: boolean; revision: number } = $props();
	let kind = $state(''),
		result = $state<{
			events: number;
			intact: boolean;
			first_mismatch_seq?: string | number;
		} | null>(null),
		pending = $state(false),
		error = $state('');
	async function verify() {
		if (pending) return;
		pending = true;
		result = null;
		error = '';
		try {
			result = await api('/audit/verify');
		} catch (reason) {
			error = errorText(reason);
		} finally {
			pending = false;
		}
	}
</script>

{#if audit}<p>
		Riwayat akses berbeda dari timeline perawatan. Pemeriksaan integritas dijalankan hanya ketika
		Anda menekan tombol di bawah.
	</p>
	<button class="button primary" disabled={pending} onclick={verify}
		>{pending ? 'Memverifikasi…' : 'Verifikasi integritas audit'}</button
	>{#if error}<p class="wf-error" role="alert">{error}</p>{/if}{#if result}<p
			class:wf-error={!result.intact}
			role="status"
		>
			{result.intact
				? `Rantai audit utuh untuk ${result.events} kejadian yang diperiksa.`
				: `Ketidakcocokan ditemukan pada nomor ${result.first_mismatch_seq}. Hubungi pengelola.`}
		</p>{/if}{:else}<p>
		Timeline merekam kejadian saat itu; status terkini ada pada bagian perawatan, pemeriksaan, dan
		resep.
	</p>
	<label
		>Jenis kejadian<select bind:value={kind}
			><option value="">Semua jenis</option><option value="episode">Perawatan</option><option
				value="laboratory">Pemeriksaan</option
			><option value="prescription">Resep</option><option value="consent">Persetujuan</option
			></select
		></label
	>{/if}
{#key `${audit}-${kind}`}<PagedList
		path={audit ? '/patient/access-summary' : `/patient/timeline${kind ? `?kind=${kind}` : ''}`}
		listKey="events"
		cursorKey={audit ? '' : 'before_seq'}
		nextKey={audit ? '' : 'next_before_seq'}
		{revision}
		empty="Belum ada kejadian pada halaman ini."
		>{#snippet children(item)}<p>{item.message}</p>
			<time datetime={item.occurred_at}>{dateText(item.occurred_at)}</time>{/snippet}</PagedList
	>{/key}
