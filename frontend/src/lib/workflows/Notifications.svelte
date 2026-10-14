<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import PagedList from './PagedList.svelte';
	import { api, dateText, parseJSON, type Run, type Entry } from './api.js';
	let { run, busy, revision }: { run: Run; busy: boolean; revision: number } = $props();
	let updates = $state(0),
		unread = $state<number | null>(null),
		connection = $state('Menghubungkan pembaruan…');
	async function count() {
		try {
			const response = await api<{ unread_count: number }>('/notifications/unread-count');
			unread = response.unread_count;
		} catch {
			unread = null;
		}
	}
	$effect(() => {
		const current = revision + updates;
		void current;
		void count();
	});
	onMount(() => {
		let source: EventSource | null = null,
			disposed = false;
		void api<{ notifications: Entry[] }>('/notifications?limit=1')
			.then((data) => {
				if (disposed) return;
				const last = data.notifications[0]?.id || '0';
				source = new EventSource(
					`/api/v1/notifications/stream?after_id=${encodeURIComponent(String(last))}`
				);
				source.onopen = () => (connection = 'Pembaruan langsung aktif');
				source.onerror = () =>
					(connection =
						'Koneksi terputus; mencoba menyambungkan kembali. Anda juga dapat memuat ulang daftar.');
				source.addEventListener('notification', (event) => {
					try {
						parseJSON((event as MessageEvent).data);
						updates++;
					} catch {
						connection = 'Pembaruan tidak dapat dibaca. Muat ulang daftar.';
					}
				});
				source.addEventListener(
					'unavailable',
					() => (connection = 'Layanan pembaruan terganggu; mencoba menyambungkan kembali.')
				);
				source.addEventListener('session-ended', () => {
					source?.close();
					void goto(resolve('/login'), { replace: true, refreshAll: true });
				});
			})
			.catch(() => (connection = 'Pembaruan langsung belum tersedia. Gunakan muat ulang daftar.'));
		return () => {
			disposed = true;
			source?.close();
		};
	});
</script>

<p>
	{unread === null
		? 'Jumlah belum dibaca belum dapat dimuat.'
		: `${unread} notifikasi belum dibaca.`}
</p>
<p class="muted" role="status">{connection}</p>
<PagedList
	path="/notifications"
	listKey="notifications"
	cursorKey="before_id"
	nextKey="next_before_id"
	revision={revision + updates}
	empty="Belum ada notifikasi untuk akun Anda."
	>{#snippet children(item)}<p>{item.message}</p>
		<p>{dateText(item.created_at)} · {item.read_at ? 'Sudah dibaca' : 'Belum dibaca'}</p>
		{#if !item.read_at}<button
				class="button secondary"
				disabled={busy}
				onclick={() => run(`/notifications/${item.id}`, { read: true }, 'PATCH')}
				>Tandai dibaca</button
			>{/if}{/snippet}</PagedList
>
