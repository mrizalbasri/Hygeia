import { goto } from '$app/navigation';
import { resolve } from '$app/paths';

export type Medication = {
	medication_name: string;
	dosage: string;
	instructions: string;
	quantity: number;
};
export type Entry = {
	corrections?: (Entry & { reason: string; corrected_at: string })[];
	id: string;
	status?: string;
	patient_ref?: string;
	doctor_name?: string;
	doctor_user_id?: string;
	organization_name?: string;
	organization_id?: string | null;
	episode_id?: string;
	test_name?: string;
	test_code?: string;
	lab_name?: string;
	result_id?: string | null;
	pharmacy_name?: string | null;
	items?: Medication[];
	grantee_name?: string;
	grantee_org_name?: string;
	purpose?: string;
	resource_type?: string;
	resource_id?: string;
	expires_at?: string;
	revoked_at?: string | null;
	created_at?: string;
	occurred_at?: string;
	starts_at?: string;
	submitted_at?: string;
	read_at?: string | null;
	message?: string;
	seq?: string | number;
	reason_code?: string;
	reason_text?: string;
	review_decision?: string;
	value?: string;
	unit?: string;
	reference_range?: string;
	flag?: string;
	note?: string;
	name?: string;
	full_name?: string;
	email?: string;
	role?: string;
	type?: string;
	disabled_at?: string | null;
	chain_id?: string;
	events?: number;
	review_note?: string;
	action?: string;
	actor_id?: string;
};
export type Payload = { [key: string]: unknown };
export type Run = (path: string, body: Payload, method?: 'POST' | 'PATCH') => Promise<boolean>;
export class ApiError extends Error {
	constructor(
		message: string,
		public status: number,
		public requestId = ''
	) {
		super(message);
	}
}

// Cursor and sequence values can exceed JavaScript's safe integer range.
export function parseJSON(source: string): unknown {
	let output = '',
		quoted = false,
		escaped = false;
	for (let i = 0; i < source.length; i++) {
		const char = source[i];
		if (quoted) {
			output += char;
			if (escaped) escaped = false;
			else if (char === '\\') escaped = true;
			else if (char === '"') quoted = false;
			continue;
		}
		if (char === '"') {
			quoted = true;
			output += char;
			continue;
		}
		if (/[0-9-]/.test(char)) {
			let number = char;
			while (i + 1 < source.length && /[0-9eE+.-]/.test(source[i + 1])) number += source[++i];
			output +=
				/^-?\d+$/.test(number) && !Number.isSafeInteger(Number(number))
					? JSON.stringify(number)
					: number;
		} else output += char;
	}
	return JSON.parse(output);
}

export async function api<T>(
	path: string,
	method: 'GET' | 'POST' | 'PATCH' = 'GET',
	body?: Payload,
	signal?: AbortSignal
): Promise<T> {
	const headers: Record<string, string> = {};
	if (method !== 'GET') {
		const csrf = await api<{ csrf_token: string }>('/auth/csrf');
		headers['X-CSRF-Token'] = csrf.csrf_token;
		headers['Content-Type'] = 'application/json';
	}
	let response: Response;
	try {
		response = await fetch(`/api/v1${path}`, {
			method,
			credentials: 'same-origin',
			cache: 'no-store',
			headers,
			body: body ? JSON.stringify(body) : undefined,
			signal: signal
				? AbortSignal.any([signal, AbortSignal.timeout(15000)])
				: AbortSignal.timeout(15000)
		});
	} catch (error) {
		if (signal?.aborted) throw error;
		throw new ApiError('Layanan belum dapat dihubungi. Coba lagi.', 0);
	}
	const data = response.status === 204 ? {} : (parseJSON(await response.text()) as Payload);
	if (!response.ok) {
		if (response.status === 401) {
			await goto(resolve('/login'), { replace: true, refreshAll: true });
		}
		const detail = data.error as { message?: string; request_id?: string } | undefined;
		throw new ApiError(
			detail?.message || 'Permintaan tidak dapat diproses.',
			response.status,
			detail?.request_id || response.headers.get('X-Request-ID') || ''
		);
	}
	return data as T;
}
export const errorText = (error: unknown) =>
	error instanceof ApiError
		? `${error.message}${error.requestId ? ` (ID: ${error.requestId})` : ''}`
		: 'Data belum dapat dimuat. Coba lagi.';
export const dateText = (value?: string | null) =>
	value && Number.isFinite(Date.parse(value))
		? new Intl.DateTimeFormat('id-ID', { dateStyle: 'medium', timeStyle: 'short' }).format(
				new Date(value)
			)
		: '—';
export const statusText: Record<string, string> = {
	pending: 'Menunggu persetujuan',
	active: 'Aktif',
	rejected: 'Ditolak',
	ended: 'Berakhir',
	pending_patient_approval: 'Menunggu persetujuan pasien',
	sent: 'Terkirim',
	accepted: 'Diterima',
	in_progress: 'Sedang diperiksa',
	completed: 'Selesai',
	cancelled: 'Dibatalkan',
	preparing: 'Sedang disiapkan',
	ready: 'Siap diambil',
	denied: 'Ditolak',
	revoked: 'Dicabut',
	expired: 'Kedaluwarsa',
	appropriate: 'Sesuai prosedur',
	needs_followup: 'Perlu tindak lanjut',
	unconscious: 'Pasien tidak sadar',
	unreachable_patient: 'Pasien tidak dapat dihubungi',
	life_threatening: 'Kondisi mengancam nyawa'
};
export const formText = (form: FormData, name: string) => String(form.get(name) || '').trim();
export const uuidPattern =
	'[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}';
export type LabResultInput = {
	value: string;
	unit?: string;
	reference_range?: string;
	flag: 'normal' | 'high' | 'low' | 'abnormal' | 'unknown';
	note?: string;
};
export type ResultReceipt = {
	id: string;
	lab_request_id: string;
	status: 'completed';
	submitted_at: string;
};
export const menus = {
	patient: [
		['overview', 'Ringkasan'],
		['episodes', 'Perawatan'],
		['labs', 'Pemeriksaan'],
		['prescriptions', 'Resep & apotek'],
		['consents', 'Izin dokter'],
		['sharing', 'CarePass'],
		['timeline', 'Perjalanan'],
		['audit', 'Riwayat akses'],
		['emergency', 'Akses darurat'],
		['notifications', 'Notifikasi']
	],
	doctor: [
		['overview', 'Ringkasan dokter'],
		['episodes', 'Pasien & perawatan'],
		['labs', 'Pemeriksaan lab'],
		['prescriptions', 'Resep'],
		['consents', 'Permintaan akses'],
		['sharing', 'Tebus CarePass'],
		['emergency', 'Akses darurat'],
		['notifications', 'Notifikasi']
	],
	lab: [
		['overview', 'Ringkasan lab'],
		['queue', 'Permintaan masuk'],
		['corrections', 'Koreksi hasil'],
		['notifications', 'Notifikasi']
	],
	pharmacy: [
		['queue', 'Antrean resep'],
		['notifications', 'Notifikasi']
	],
	admin: [
		['organizations', 'Organisasi'],
		['users', 'Akun'],
		['audit', 'Status audit'],
		['breakglass', 'Tinjauan akses darurat'],
		['activity', 'Log aktivitas'],
		['notifications', 'Notifikasi']
	]
};
