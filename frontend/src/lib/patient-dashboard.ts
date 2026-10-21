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

export async function readPatientList<T>(path: string, key: string): Promise<Resource<T>> {
	try {
		const response = await fetch(`/api/v1/${path}`, {
			cache: 'no-store',
			signal: AbortSignal.timeout(12000)
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
