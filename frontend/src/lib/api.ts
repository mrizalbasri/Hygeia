export type Role = 'patient' | 'doctor' | 'lab' | 'pharmacy' | 'admin';

export type User = {
	id: string;
	email: string;
	full_name: string;
	role: Role;
	organization: { id: string; name: string; type: string } | null;
};

export type LabQueueRequest = {
	id: string;
	patient_ref: string;
	test_code: string;
	test_name: string;
	status: 'sent' | 'accepted' | 'in_progress';
};

export type LabResultInput = {
	value: string;
	unit: string;
	reference_range: string;
	flag: 'normal' | 'high' | 'low' | 'abnormal' | 'unknown';
	note: string;
};

export type PharmacyItem = {
	medication_name: string;
	dosage: string;
	instructions: string;
	quantity: number;
};

export type QueuePrescription = {
	id: string;
	patient_ref: string;
	items: PharmacyItem[];
	status: 'sent' | 'accepted' | 'preparing' | 'ready';
};

type ErrorPayload = { error?: { message?: string } };

let csrfToken: string | null = null;

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
	const response = await fetch(`/api/v1${path}`, {
		...init,
		credentials: 'same-origin',
		headers: {
			Accept: 'application/json',
			...(init.body ? { 'Content-Type': 'application/json' } : {}),
			...(init.headers ?? {})
		}
	});

	if (!response.ok) {
		const payload = (await response.json().catch(() => ({}))) as ErrorPayload;
		throw new Error(payload.error?.message ?? `Permintaan gagal (${response.status})`);
	}

	if (response.status === 204) return undefined as T;
	return (await response.json()) as T;
}

export async function csrf(): Promise<string> {
	const payload = await request<{ csrf_token: string }>('/auth/csrf');
	csrfToken = payload.csrf_token;
	return csrfToken;
}

export async function login(email: string, password: string): Promise<User> {
	const token = await csrf();
	const payload = await request<{ user: User }>('/auth/login', {
		method: 'POST',
		headers: { 'X-CSRF-Token': token },
		body: JSON.stringify({ email, password })
	});
	return payload.user;
}

export async function currentUser(): Promise<User> {
	const payload = await request<{ user: User }>('/me');
	return payload.user;
}

export async function logout(): Promise<void> {
	const token = csrfToken ?? (await csrf());
	await request<void>('/auth/logout', {
		method: 'POST',
		headers: { 'X-CSRF-Token': token },
		body: '{}'
	});
	csrfToken = null;
}

export async function mutate<T>(path: string, method: 'POST' | 'PATCH', body: unknown): Promise<T> {
	const token = csrfToken ?? (await csrf());
	try {
		return await request<T>(path, {
			method,
			headers: { 'X-CSRF-Token': token },
			body: JSON.stringify(body)
		});
	} catch (error) {
		if (error instanceof Error && error.message.includes('CSRF')) {
			csrfToken = await csrf();
			return request<T>(path, {
				method,
				headers: { 'X-CSRF-Token': csrfToken },
				body: JSON.stringify(body)
			});
		}
		throw error;
	}
}

export async function labQueue(status = ''): Promise<LabQueueRequest[]> {
	const query = status ? `?status=${encodeURIComponent(status)}` : '';
	const payload = await request<{ lab_requests: LabQueueRequest[] }>(`/lab-requests${query}`);
	return payload.lab_requests;
}

export async function updateLabStatus(id: string, status: 'accepted' | 'in_progress') {
	return mutate<{ lab_request: LabQueueRequest }>(`/lab-requests/${id}/status`, 'PATCH', { status });
}

export async function submitLabResult(id: string, result: LabResultInput) {
	return mutate<{ result: { id: string; status: 'completed'; submitted_at: string } }>(
		`/lab-requests/${id}/result`,
		'POST',
		result
	);
}

export async function pharmacyQueue(status = ''): Promise<QueuePrescription[]> {
	const query = status ? `?status=${encodeURIComponent(status)}` : '';
	const payload = await request<{ prescriptions: QueuePrescription[] }>(`/prescriptions${query}`);
	return payload.prescriptions;
}

export async function updatePrescriptionStatus(
	id: string,
	status: 'accepted' | 'preparing' | 'ready' | 'completed'
) {
	return mutate<{ prescription: QueuePrescription }>(`/prescriptions/${id}/status`, 'PATCH', {
		status
	});
}
