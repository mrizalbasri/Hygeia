export type Episode = {
	id: string;
	doctor_name: string;
	organization_name: string;
	status: string;
};

export type LabRequest = {
	id: string;
	test_name: string;
	lab_name: string;
	status: string;
};

export type Prescription = {
	id: string;
	status: string;
	pharmacy_name: string | null;
	items: { medication_name: string; dosage: string; instructions: string; quantity: number }[];
};

export type TimelineEvent = {
	seq: number;
	occurred_at: string;
	kind: string;
	message: string;
};

export type Resource<T> = { items: T[]; loading: boolean; error: string };

export const statuses: Record<string, string> = {
	pending: 'Menunggu jawaban Anda',
	active: 'Perawatan aktif',
	rejected: 'Ditolak',
	ended: 'Perawatan berakhir',
	pending_patient_approval: 'Menunggu persetujuan Anda',
	sent: 'Terkirim',
	accepted: 'Diterima',
	in_progress: 'Sedang diperiksa',
	preparing: 'Sedang disiapkan',
	ready: 'Siap diambil',
	completed: 'Selesai',
	cancelled: 'Dibatalkan'
};

export type WorkSummary = {
	active_episodes: number;
	labs_in_progress: number;
	prescriptions_ready: number;
	pending: { episodes: number; lab_requests: number; prescriptions: number; consents: number };
};
export type SummaryResource = { data: WorkSummary | null; loading: boolean; error: string };

export function pendingWork(summary: WorkSummary) {
	return [
		{
			id: 'episodes',
			count: summary.pending.episodes,
			title: 'Undangan perawatan',
			text: 'Tinjau hubungan perawatan dengan dokter.',
			view: 'episodes',
			icon: 'episodes'
		},
		{
			id: 'labs',
			count: summary.pending.lab_requests,
			title: 'Persetujuan pemeriksaan',
			text: 'Tinjau pengiriman pemeriksaan ke laboratorium.',
			view: 'labs',
			icon: 'labs'
		},
		{
			id: 'prescriptions',
			count: summary.pending.prescriptions,
			title: 'Persetujuan resep',
			text: 'Pilih apotek dan tinjau pengiriman resep.',
			view: 'prescriptions',
			icon: 'prescriptions'
		},
		{
			id: 'consents',
			count: summary.pending.consents,
			title: 'Permintaan izin dokter',
			text: 'Tinjau dokter, tujuan, dan durasi akses hasil.',
			view: 'consents',
			icon: 'consents'
		}
	].filter((item) => item.count > 0);
}

export async function readWorkSummary(signal?: AbortSignal): Promise<SummaryResource> {
	try {
		const response = await fetch('/api/v1/patient/work-summary', {
			cache: 'no-store',
			signal: signal
				? AbortSignal.any([signal, AbortSignal.timeout(12000)])
				: AbortSignal.timeout(12000)
		});
		if (!response.ok) throw new Error('Summary unavailable');
		const { summary } = await response.json();
		const counts = [
			summary?.active_episodes,
			summary?.labs_in_progress,
			summary?.prescriptions_ready,
			summary?.pending?.episodes,
			summary?.pending?.lab_requests,
			summary?.pending?.prescriptions,
			summary?.pending?.consents
		];
		if (!counts.every((value) => Number.isSafeInteger(value) && value >= 0))
			throw new Error('Invalid summary');
		return { data: summary, loading: false, error: '' };
	} catch {
		return {
			data: null,
			loading: false,
			error: 'Ringkasan pekerjaan belum dapat dimuat. Coba lagi.'
		};
	}
}

export async function readPatientList<T>(
	path: string,
	key: string,
	signal?: AbortSignal
): Promise<Resource<T>> {
	try {
		const response = await fetch(`/api/v1/${path}`, {
			cache: 'no-store',
			signal: signal
				? AbortSignal.any([signal, AbortSignal.timeout(12000)])
				: AbortSignal.timeout(12000)
		});
		if (!response.ok) {
			return {
				items: [],
				loading: false,
				error:
					response.status === 401
						? 'Sesi berakhir. Masuk kembali untuk melihat data.'
						: 'Data belum dapat dimuat. Coba lagi.'
			};
		}
		const body = await response.json();
		if (!Array.isArray(body[key])) throw new Error('Invalid list');
		return { items: body[key], loading: false, error: '' };
	} catch {
		return { items: [], loading: false, error: 'Layanan belum dapat dihubungi. Coba lagi.' };
	}
}
