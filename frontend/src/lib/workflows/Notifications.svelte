<script lang="ts">
	import { untrack } from 'svelte';
	import PagedList from './PagedList.svelte';
	import { dateText, type Run } from './api.js';
	import { useNotifications } from './notification-state.svelte.js';
	let { run, busy, revision }: { run: Run; busy: boolean; revision: number } = $props();
	const notifications = useNotifications();
	$effect(() => {
		const current = revision;
		void current;
		untrack(() => void notifications.refreshUnread());
	});
</script>

<p>
	{notifications.unread === null
		? 'Jumlah belum dibaca belum dapat dimuat.'
		: `${notifications.unread} notifikasi belum dibaca.`}
</p>
<p class="muted" role="status">{notifications.connection}</p>
<PagedList
	path="/notifications"
	listKey="notifications"
	cursorKey="before_id"
	nextKey="next_before_id"
	{revision}
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
