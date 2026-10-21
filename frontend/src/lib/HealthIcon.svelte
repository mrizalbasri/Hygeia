<script lang="ts">
	let { name }: { name: 'care' | 'lab' | 'medicine' | 'history' | 'shield' } = $props();
</script>

<svg
	width="22"
	height="22"
	viewBox="0 0 24 24"
	fill="none"
	stroke="currentColor"
	stroke-width="1.7"
	stroke-linecap="round"
	stroke-linejoin="round"
	aria-hidden="true"
>
	{#if name === 'care'}
		<path d="M20 13c3-5-3-11-8-6C7 2 1 8 4 13l8 8 8-8Z" /><path d="M9 12h6m-3-3v6" />
	{:else if name === 'lab'}
		<path d="M9 3h6m-5 0v7l-5 8a2 2 0 0 0 2 3h10a2 2 0 0 0 2-3l-5-8V3M8 15h8" />
	{:else if name === 'medicine'}
		<path d="m5 12 7-7a5 5 0 0 1 7 7l-7 7a5 5 0 0 1-7-7Zm4-4 7 7" />
	{:else if name === 'history'}
		<path d="M4 9a9 9 0 1 1 0 7M4 4v5h5m3-2v5l3 2" />
	{:else}
		<path d="m12 3 8 3v6c0 5-8 9-8 9s-8-4-8-9V6l8-3Z" /><path d="m8 12 3 3 5-6" />
	{/if}
</svg>
