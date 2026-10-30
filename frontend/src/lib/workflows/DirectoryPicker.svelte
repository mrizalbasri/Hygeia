<script lang="ts">
	import { api, errorText, type Entry, type Payload } from './api.js';
	let {
		path,
		listKey,
		label,
		organizationType,
		disabled = false,
		valueKey = 'id',
		value = $bindable('')
	}: {
		path: string;
		listKey: string;
		label: string;
		organizationType?: string;
		disabled?: boolean;
		value?: string;
		valueKey?: 'id' | 'result_id';
	} = $props();
	const describe = (item: Entry) =>
		item.name ||
		item.test_name ||
		item.items?.map((drug) => drug.medication_name).join(', ') ||
		`${item.patient_ref || ''} · ${item.doctor_name || item.organization_name || item.id}`;
	let entries = $state<Entry[]>([]),
		offset = $state(0),
		error = $state(''),
		loading = $state(true),
		retry = $state(0),
		chosen = $state<Entry | null>(null);
	let visibleEntries = $derived(
		organizationType ? entries.filter((item) => item.type === organizationType) : entries
	);
	$effect(() => {
		const page = offset,
			refresh = retry;
		void refresh;
		const controller = new AbortController();
		loading = true;
		error = '';
		entries = [];
		void api<Payload>(
			`${path}${path.includes('?') ? '&' : '?'}limit=20&offset=${page}`,
			'GET',
			undefined,
			controller.signal
		)
			.then((data) => {
				if (!controller.signal.aborted) {
					if (!Array.isArray(data[listKey])) throw new Error();
					entries = data[listKey] as Entry[];
				}
			})
			.catch((reason) => {
				if (!controller.signal.aborted) error = errorText(reason);
			})
			.finally(() => {
				if (!controller.signal.aborted) loading = false;
			});
		return () => controller.abort();
	});
</script>

<div class="wf-picker">
	<label
		>{label}<select
			aria-label={label}
			required
			bind:value
			disabled={disabled || loading || !!error}
			onchange={() => (chosen = visibleEntries.find((item) => item[valueKey] === value) || null)}
			><option value="">Pilih {label.toLowerCase()}</option
			>{#if value && chosen && !visibleEntries.some((item) => item.id === chosen?.id)}<option
					value={chosen[valueKey] || ''}>{describe(chosen)}</option
				>{/if}{#each visibleEntries as item (item.id)}<option
					value={item[valueKey] || ''}
					disabled={!item[valueKey]}>{describe(item)}</option
				>{/each}</select
		></label
	>{#if loading}<small role="status">Memuat pilihan…</small>{:else if error}<p
			class="wf-error"
			role="alert"
		>
			{error}<button type="button" class="text-button" onclick={() => retry++}>Coba lagi</button>
		</p>{:else if !visibleEntries.length}<small
			>Tidak ada pilihan yang sesuai pada halaman ini. Periksa halaman berikutnya jika tersedia.</small
		>{/if}
	<div class="wf-picker-pages">
		<button
			type="button"
			disabled={disabled || loading || offset === 0}
			onclick={() => (offset -= 20)}>Pilihan sebelumnya</button
		><button
			type="button"
			disabled={disabled || loading || !!error || entries.length < 20}
			onclick={() => (offset += 20)}>Pilihan berikutnya</button
		>
	</div>
</div>
