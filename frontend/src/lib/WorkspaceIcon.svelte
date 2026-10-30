<script lang="ts">
	let { name }: { name: string } = $props();
</script>

<svg
	width="18"
	height="18"
	viewBox="0 0 24 24"
	fill="none"
	stroke="currentColor"
	stroke-width="1.7"
	stroke-linecap="round"
	stroke-linejoin="round"
	aria-hidden="true"
>
	{#if name === 'menu'}
		<path d="M4 6h16M4 12h16M4 18h16" />
	{:else if name === 'close'}
		<path d="m6 6 12 12M18 6 6 18" />
	{:else if name === 'overview'}
		<rect x="3" y="3" width="7" height="7" rx="1.5" /><rect
			x="14"
			y="3"
			width="7"
			height="7"
			rx="1.5"
		/><rect x="3" y="14" width="7" height="7" rx="1.5" /><rect
			x="14"
			y="14"
			width="7"
			height="7"
			rx="1.5"
		/>
	{:else if name === 'episodes' || name === 'users'}
		<circle cx="9" cy="8" r="3" /><path
			d="M3 21v-3a6 6 0 0 1 12 0v3m1-16a3 3 0 0 1 0 6m3 10v-3a6 6 0 0 0-3-5"
		/>
	{:else if name === 'profile'}
		<circle cx="12" cy="8" r="4" /><path d="M4 21v-2a8 8 0 0 1 16 0v2" />
	{:else if name === 'labs' || name === 'corrections' || name === 'queue' || name === 'receipts'}
		<path d="M9 3h6m-5 0v7l-5 8a2 2 0 0 0 2 3h10a2 2 0 0 0 2-3l-5-8V3M8 15h8" />
	{:else if name === 'prescriptions'}
		<path d="m5 12 7-7a5 5 0 0 1 7 7l-7 7a5 5 0 0 1-7-7Zm4-4 7 7" />
	{:else if name === 'consents' || name === 'audit'}
		<path d="m12 3 8 3v6c0 5-8 9-8 9s-8-4-8-9V6l8-3Z" /><path d="m8 12 3 3 5-6" />
	{:else if name === 'sharing'}
		<rect x="3" y="3" width="6" height="6" rx="1" /><rect
			x="15"
			y="3"
			width="6"
			height="6"
			rx="1"
		/><rect x="3" y="15" width="6" height="6" rx="1" /><path
			d="M15 15h3v3h3m-6 3h3m3-9v3M3 12h3m6-9v3m0 6v6"
		/>
	{:else if name === 'emergency' || name === 'breakglass'}
		<path d="M10.3 4a2 2 0 0 1 3.4 0l8 14a2 2 0 0 1-1.7 3H4a2 2 0 0 1-1.7-3Z" /><path
			d="M12 9v4m0 4h.01"
		/>
	{:else if name === 'notifications'}
		<path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9m-11 12a2 2 0 0 0 4 0" />
	{:else if name === 'organizations'}
		<rect x="5" y="3" width="14" height="18" rx="2" /><path
			d="M9 7h1m4 0h1m-6 4h1m4 0h1m-5 10v-6h4v6"
		/>
	{:else if name === 'refresh'}
		<path d="M20 7v5h-5M4 17v-5h5m-4-5a8 8 0 0 1 14-2l1 2M4 17l1 2a8 8 0 0 0 14-2" />
	{:else if name === 'arrow'}
		<path d="M5 12h14m-5-5 5 5-5 5" />
	{:else if name === 'logout'}
		<path d="M9 4H5a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h4m5-13 5 5-5 5m-6-5h11" />
	{:else}
		<circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" />
	{/if}
</svg>
