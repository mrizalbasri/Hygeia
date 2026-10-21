<script lang="ts">
	import { onMount } from 'svelte';
	import PagedList from './PagedList.svelte';
	import DirectoryPicker from './DirectoryPicker.svelte';
	import { api, errorText, dateText, type Entry, type Run } from './api.js';
	let {
		doctor,
		run,
		busy,
		revision,
		show,
		initialToken = '',
		consumed = () => {}
	}: {
		doctor: boolean;
		run: Run;
		busy: boolean;
		revision: number;
		show: (kind: string, id: string, expires?: string) => void;
		initialToken?: string;
		consumed?: () => void;
	} = $props();
	let kind = $state('lab_result'),
		resource = $state(''),
		token = $state(''),
		pending = $state(false),
		error = $state(''),
		created = $state<{
			carepass: Entry;
			token: string;
			share_url: string;
			qr_png_data_uri: string;
		} | null>(null),
		redeemed = $state<Entry | null>(null),
		localRevision = $state(0),
		feedback = $state('');
	onMount(() => {
		token = initialToken;
		consumed();
		const timer = setInterval(() => {
			if (created && Date.parse(created.carepass.expires_at || '') <= Date.now()) created = null;
			if (redeemed && Date.parse(redeemed.expires_at || '') <= Date.now()) redeemed = null;
		}, 1000);
		return () => {
			clearInterval(timer);
			token = '';
			created = null;
			redeemed = null;
		};
	});
	async function create(event: SubmitEvent) {
		event.preventDefault();
		if (pending) return;
		pending = true;
		error = '';
		created = null;
		const data = new FormData(event.currentTarget as HTMLFormElement);
		try {
			created = await api('/carepasses', 'POST', {
				resource_type: kind,
				resource_id: resource,
				ttl_minutes: Number(data.get('ttl_minutes'))
			});
			localRevision++;
		} catch (reason) {
			error = errorText(reason);
		} finally {
			pending = false;
		}
	}
	async function redeem(event: SubmitEvent) {
		event.preventDefault();
		if (pending) return;
		pending = true;
		error = '';
		redeemed = null;
		let value = token.trim();
		try {
			if (/^https?:\/\//.test(value))
				value = new URLSearchParams(new URL(value).hash.slice(1)).get('token') || '';
			const result = await api<{ carepass: Entry }>('/carepasses/redeem', 'POST', { token: value });
			redeemed = result.carepass;
		} catch (reason) {
			error = errorText(reason);
		} finally {
			token = '';
			pending = false;
		}
	}
	async function copy() {
		try {
			await navigator.clipboard.writeText(created?.share_url || '');
			feedback = 'Tautan disalin.';
		} catch {
			feedback = 'Penyalinan tidak tersedia. Pilih dan salin tautan secara manual.';
		}
	}
</script>

{#if doctor}<form class="wf-form-panel" onsubmit={redeem}>
		<h2>Tebus CarePass pasien</h2>
		<p>
			Pasien membagikan token atau tautan kepada dokter yang dituju. Penebusan pertama mengikat
			akses ke dokter dan klinik Anda.
		</p>
		<label
			>Token atau tautan CarePass<input
				required
				bind:value={token}
				maxlength="2048"
				autocomplete="off"
				spellcheck="false"
			/></label
		><button class="button primary" disabled={pending}
			>{pending ? 'Menebus…' : 'Tebus CarePass'}</button
		>
	</form>{:else}<form class="wf-form-panel" onsubmit={create}>
		<h2>Bagikan satu data dengan CarePass</h2>
		<p>
			QR dan tautan hanya ditampilkan saat dibuat. Bagikan langsung kepada dokter yang Anda tuju.
		</p>
		<label
			>Jenis data<select
				bind:value={kind}
				onchange={() => {
					resource = '';
					created = null;
				}}
				><option value="lab_result">Hasil laboratorium selesai</option><option value="prescription"
					>Resep</option
				></select
			></label
		>{#if kind === 'lab_result'}<DirectoryPicker
				path="/lab-requests?status=completed"
				listKey="lab_requests"
				label="Hasil pemeriksaan"
				valueKey="result_id"
				bind:value={resource}
			/>{:else}<DirectoryPicker
				path="/prescriptions"
				listKey="prescriptions"
				label="Resep yang dibagikan"
				bind:value={resource}
			/>{/if}<label
			>Masa berlaku<select name="ttl_minutes"
				><option value="15">15 menit</option><option value="60">60 menit</option></select
			></label
		><button class="button primary" disabled={pending || !resource}
			>{pending ? 'Membuat…' : 'Buat CarePass'}</button
		>
	</form>{/if}
{#if error}<p class="wf-error" role="alert">{error}</p>{/if}
{#if redeemed}<section class="wf-form-panel">
		<h2>CarePass berhasil ditebus</h2>
		<p>Berlaku sampai {dateText(redeemed.expires_at)}</p>
		<button
			class="button primary"
			onclick={() => show(redeemed!.resource_type!, redeemed!.resource_id!, redeemed!.expires_at)}
			>Buka data yang dibagikan</button
		>
	</section>{/if}
{#if created}<section class="wf-form-panel">
		<h2>CarePass siap dibagikan</h2>
		<p>
			Berlaku sampai {dateText(created.carepass.expires_at)}. Tautan memberi akses kepada dokter
			penebus pertama.
		</p>
		{#if created.qr_png_data_uri.startsWith('data:image/png;base64,')}<img
				class="wf-qr"
				src={created.qr_png_data_uri}
				width="256"
				height="256"
				alt="QR CarePass untuk satu data yang dipilih"
			/>{/if}<label
			>Tautan berbagi<textarea readonly value={created.share_url} aria-label="Tautan CarePass"
			></textarea></label
		>
		<div class="wf-actions">
			<button class="button secondary" onclick={copy}>Salin tautan</button><button
				class="button secondary"
				onclick={() => {
					created = null;
					feedback = '';
				}}>Sembunyikan QR & tautan</button
			>
		</div>
		<p role="status">{feedback}</p>
	</section>{/if}
{#if !doctor}<PagedList
		path="/carepasses"
		listKey="carepasses"
		revision={revision + localRevision}
		empty="Belum ada CarePass yang dibuat."
		>{#snippet children(item)}<div class="wf-row-heading">
				<h2>{item.resource_type === 'lab_result' ? 'Hasil lab' : 'Resep'}</h2>
				<span class="health-status"
					>{item.revoked_at
						? 'Dicabut'
						: Date.parse(item.expires_at || '') <= Date.now()
							? 'Kedaluwarsa'
							: 'Aktif'}</span
				>
			</div>
			<p>Berlaku sampai {dateText(item.expires_at)}</p>
			<code>{item.resource_id}</code
			>{#if !item.revoked_at && Date.parse(item.expires_at || '') > Date.now()}<button
					class="button secondary"
					disabled={busy}
					onclick={async () => {
						if (confirm('Cabut CarePass ini? Akses melalui tautan ini akan ditutup.')) {
							if (await run(`/carepasses/${item.id}`, { action: 'revoke' }, 'PATCH'))
								created = null;
						}
					}}>Cabut CarePass</button
				>{/if}{/snippet}</PagedList
	>{/if}
