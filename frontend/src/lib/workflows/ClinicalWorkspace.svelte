<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import PatientDashboard from '../PatientDashboard.svelte';
	import DoctorOverview from './DoctorOverview.svelte';
	import type { User } from '../auth.js';
	import { api, ApiError, errorText, menus, type Payload } from './api.js';
	import Episodes from './Episodes.svelte';
	import LabRequests from './LabRequests.svelte';
	import Prescriptions from './Prescriptions.svelte';
	import Consents from './Consents.svelte';
	import Sharing from './Sharing.svelte';
	import Emergency from './Emergency.svelte';
	import History from './History.svelte';
	import Notifications from './Notifications.svelte';
	import ClinicalDetail from './ClinicalDetail.svelte';
	import './workflow.css';
	let { user }: { user: User } = $props();
	let doctor = $derived(user.role === 'doctor');
	let menu = $derived(doctor ? menus.doctor : menus.patient);
	let view = $derived(
		menu.some(([id]) => id === page.url.searchParams.get('view'))
			? page.url.searchParams.get('view')!
			: menu[0][0]
	);
	let busy = $state(false),
		ready = $state(false),
		error = $state(''),
		success = $state(''),
		revision = $state(0),
		detail = $state<{ kind: string; id: string; expires?: string } | null>(null);
	onMount(() => {
		ready = true;
	});
	$effect(() => {
		const currentView = view;
		void currentView;
		error = '';
		success = '';
		detail = null;
	});
	function show(kind: string, id: string, expires?: string) {
		detail = { kind, id, expires };
	}
	async function run(path: string, body: Payload, method: 'POST' | 'PATCH' = 'POST') {
		if (busy) return false;
		busy = true;
		error = '';
		success = '';
		detail = null;
		try {
			await api(path, method, body);
			revision++;
			success = 'Perubahan berhasil disimpan. Daftar diperbarui.';
			return true;
		} catch (reason) {
			error = errorText(reason);
			if (reason instanceof ApiError && reason.status === 409) {
				revision++;
				error += ' Status telah berubah. Periksa daftar terbaru sebelum mencoba kembali.';
			}
			return false;
		} finally {
			busy = false;
		}
	}
</script>

{#if !doctor && view !== 'overview'}<section class="wf-overview-note patient-reference-note">
		<h2>Kode pasien Anda</h2>
		{#if user.patient_ref}<code>{user.patient_ref}</code>
			<p>Berikan kode ini kepada dokter yang akan mengundang Anda dalam perawatan.</p>{:else}<p>
				Kode pasien belum tersedia dari layanan identitas. Minta kode Anda kepada pengelola.
			</p>{/if}
	</section>{/if}
{#if view === 'overview'}
	{#if doctor}<DoctorOverview {user} />{:else}<PatientDashboard {user} />{/if}
{:else}<div class="wf-heading">
		<p class="muted">{user.full_name}</p>
		<h1>{doctor ? 'Ruang Perawatan' : 'Perjalanan Saya'}</h1>
		<h2>{menu.find(([id]) => id === view)?.[1]}</h2>
	</div>{/if}
{#if error}<p class="wf-error" role="alert">{error}</p>{/if}{#if success}<p
		class="wf-success"
		role="status"
	>
		{success}
	</p>{/if}{#if busy}<p role="status">Menyimpan perubahan…</p>{/if}
{#key view}<fieldset class="wf-content" disabled={busy || !ready} aria-busy={busy || !ready}>
		{#if view === 'episodes'}<Episodes {doctor} {run} {busy} {revision} />
		{:else if view === 'labs'}<LabRequests {doctor} {run} {busy} {revision} {show} />
		{:else if view === 'prescriptions'}<Prescriptions {doctor} {run} {busy} {revision} {show} />
		{:else if view === 'consents'}<Consents {doctor} {run} {busy} {revision} {show} />
		{:else if view === 'sharing'}<Sharing {doctor} {run} {busy} {revision} {show} />
		{:else if view === 'emergency'}<Emergency {doctor} {run} {busy} {revision} {show} />
		{:else if view === 'timeline' || view === 'audit'}<History
				audit={view === 'audit'}
				{revision}
			/>
		{:else if view === 'notifications'}<Notifications {run} {busy} {revision} />{/if}
	</fieldset>{/key}
{#if detail}{#key `${detail.kind}-${detail.id}`}<ClinicalDetail
			kind={detail.kind}
			id={detail.id}
			expiresAt={detail.expires}
			close={() => (detail = null)}
		/>{/key}{/if}
