export type Role = 'patient' | 'doctor' | 'lab' | 'pharmacy' | 'admin';

export type User = {
	id: string;
	email: string;
	full_name: string;
	role: Role;
	organization: { id: string; name: string; type: string } | null;
};

export const roles: Record<
	Role,
	{ label: string; title: string; description: string; items: string[] }
> = {
	patient: {
		label: 'Pasien',
		title: 'Perjalanan Saya',
		description: 'Satu tempat untuk mengikuti perawatan dan mengendalikan akses data Anda.',
		items: ['Perjalanan Saya', 'Persetujuan', 'Izin', 'Riwayat Akses']
	},
	doctor: {
		label: 'Dokter',
		title: 'Ruang Perawatan',
		description: 'Hubungkan pemeriksaan, hasil, dan tindak lanjut dalam satu perjalanan pasien.',
		items: ['Pasien', 'Permintaan Lab', 'Resep', 'Minta Akses', 'Akses Darurat']
	},
	lab: {
		label: 'Laboratorium',
		title: 'Permintaan Masuk',
		description: 'Proses pemeriksaan yang ditujukan ke laboratorium Anda.',
		items: ['Permintaan Masuk', 'Hasil Pemeriksaan']
	},
	pharmacy: {
		label: 'Apotek',
		title: 'Antrean Resep',
		description: 'Siapkan resep yang telah dibagikan pasien ke apotek Anda.',
		items: ['Antrean Resep', 'Resep Selesai']
	},
	admin: {
		label: 'Admin',
		title: 'Kelola Platform',
		description: 'Kelola akun dan organisasi serta pantau akuntabilitas akses.',
		items: ['Organisasi', 'Akun', 'Status Audit', 'Tinjauan Akses Darurat']
	}
};

export async function authMutation(path: 'login' | 'logout', body: unknown = {}) {
	const csrf = await fetch('/api/v1/auth/csrf', { cache: 'no-store' });
	if (!csrf.ok) throw new Error('Layanan belum dapat dihubungi. Coba lagi sebentar.');
	const { csrf_token } = (await csrf.json()) as { csrf_token: string };
	return fetch(`/api/v1/auth/${path}`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf_token },
		body: JSON.stringify(body)
	});
}
