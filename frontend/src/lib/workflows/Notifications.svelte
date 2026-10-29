<script lang="ts">
	import { onMount } from 'svelte';
	import { subscribeRealtime } from './realtime.js';
	import PagedList from './PagedList.svelte';
	import { api, dateText, type Run } from './api.js';
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
	onMount(() =>
		subscribeRealtime(
			() => updates++,
			(status) => (connection = status)
		)
	);
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
