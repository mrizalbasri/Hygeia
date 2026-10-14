<script lang="ts">
	import type { Snippet } from 'svelte';
	import { SvelteURLSearchParams } from 'svelte/reactivity';
	import { api, errorText, statusText, type Entry, type Payload } from './api.js';
	let {
		path,
		listKey,
		revision = 0,
		statuses = [],
		cursorKey = '',
		nextKey = '',
		children,
		empty = 'Belum ada catatan pada daftar ini.'
	}: {
		path: string;
		listKey: string;
		revision?: number;
		statuses?: string[];
		cursorKey?: string;
		nextKey?: string;
		children: Snippet<[Entry]>;
		empty?: string;
	} = $props();
	let items = $state<Entry[]>([]),
		loading = $state(true),
		error = $state(''),
		status = $state(''),
		offset = $state(0),
		cursor = $state(''),
		next = $state(''),
		previous = $state<string[]>([]),
		retry = $state(0);
	$effect(() => {
		const currentPath = path,
			currentRevision = revision,
			query = new SvelteURLSearchParams({ limit: '20' });
		if (status) query.set('status', status);
		if (cursorKey) {
			if (cursor) query.set(cursorKey, cursor);
		} else query.set('offset', String(offset));
		const currentRetry = retry;
		void currentRevision;
		void currentRetry;
		const controller = new AbortController();
		items = [];
		error = '';
		loading = true;
		next = '';
		void api<Payload>(
			`${currentPath}${currentPath.includes('?') ? '&' : '?'}${query}`,
			'GET',
			undefined,
			controller.signal
		)
			.then((data) => {
				if (controller.signal.aborted) return;
				if (!Array.isArray(data[listKey])) throw new Error('Invalid list');
				items = data[listKey] as Entry[];
				next = String(data[nextKey] || '');
			})
			.catch((reason) => {
				if (!controller.signal.aborted) error = errorText(reason);
			})
			.finally(() => {
				if (!controller.signal.aborted) loading = false;
			});
		return () => controller.abort();
	});
	function reset() {
		offset = 0;
		cursor = '';
		previous = [];
	}
</script>

<div class="wf-list">
	<div class="wf-toolbar">
		{#if statuses.length}<label
				>Status<select aria-label="Status" bind:value={status} onchange={reset}
					><option value="">Semua status</option>{#each statuses as item (item)}<option value={item}
							>{statusText[item] || item}</option
						>{/each}</select
				></label
			>{/if}<button
			type="button"
			class="button secondary"
			disabled={loading}
			onclick={() => retry++}>Muat ulang daftar</button
		>
	</div>
	{#if loading}<p class="wf-state" role="status">Memuat data…</p>{:else if error}<div
			class="wf-error"
			role="alert"
		>
			{error}<button type="button" class="text-button" onclick={() => retry++}>Coba lagi</button>
		</div>{:else if !items.length}<p class="wf-state">{empty}</p>{:else}<div class="wf-rows">
			{#each items as item, index (`${item.id || item.seq || index}`)}<article class="wf-row">
					{@render children(item)}
				</article>{/each}
		</div>{/if}
	<div class="wf-pagination">
		<button
			type="button"
			class="button secondary"
			disabled={loading || (cursorKey ? previous.length === 0 : offset === 0)}
			onclick={() => {
				if (cursorKey) {
					cursor = previous[previous.length - 1];
					previous = previous.slice(0, -1);
				} else offset -= 20;
			}}>Sebelumnya</button
		><span
			>{cursorKey ? `Halaman ${previous.length + 1}` : `Halaman ${offset / 20 + 1}`} · hingga 20 catatan</span
		><button
			type="button"
			class="button secondary"
			disabled={loading || !!error || (cursorKey ? !next : items.length < 20)}
			onclick={() => {
				if (cursorKey) {
					previous = [...previous, cursor];
					cursor = next;
				} else offset += 20;
			}}>Berikutnya</button
		>
	</div>
</div>
