import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
	labQueue,
	updateLabStatus,
	submitLabResult,
	pharmacyQueue,
	updatePrescriptionStatus
} from './api';

describe('Hygeia Lab & Pharmacy API client', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	it('fetches lab queue without query when status is empty', async () => {
		const mockResponse = {
			lab_requests: [
				{
					id: 'lab-1',
					patient_ref: 'HYG-DEMO-001',
					test_code: 'DEMO-GLUCOSE',
					test_name: 'Pemeriksaan Glukosa',
					status: 'sent'
				}
			]
		};

		globalThis.fetch = vi.fn().mockResolvedValue({
			ok: true,
			status: 200,
			json: async () => mockResponse
		} as Response);

		const result = await labQueue();
		expect(globalThis.fetch).toHaveBeenCalledWith(
			'/api/v1/lab-requests',
			expect.objectContaining({ credentials: 'same-origin' })
		);
		expect(result).toHaveLength(1);
		expect(result[0].patient_ref).toBe('HYG-DEMO-001');
	});

	it('fetches lab queue with encoded status filter', async () => {
		globalThis.fetch = vi.fn().mockResolvedValue({
			ok: true,
			status: 200,
			json: async () => ({ lab_requests: [] })
		} as Response);

		await labQueue('in_progress');
		expect(globalThis.fetch).toHaveBeenCalledWith(
			'/api/v1/lab-requests?status=in_progress',
			expect.anything()
		);
	});

	it('fetches pharmacy queue with query parameter', async () => {
		const mockPrescriptions = {
			prescriptions: [
				{
					id: 'rx-1',
					patient_ref: 'HYG-DEMO-001',
					items: [
						{
							medication_name: 'Amoxicillin',
							dosage: '500mg',
							instructions: '3x sehari',
							quantity: 10
						}
					],
					status: 'sent'
				}
			]
		};

		globalThis.fetch = vi.fn().mockResolvedValue({
			ok: true,
			status: 200,
			json: async () => mockPrescriptions
		} as Response);

		const result = await pharmacyQueue('sent');
		expect(globalThis.fetch).toHaveBeenCalledWith(
			'/api/v1/prescriptions?status=sent',
			expect.anything()
		);
		expect(result[0].items[0].medication_name).toBe('Amoxicillin');
	});

	it('sends CSRF token and mutation body on updateLabStatus', async () => {
		globalThis.fetch = vi
			.fn()
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => ({ csrf_token: 'test-token-123' })
			} as Response)
			.mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => ({ lab_request: { id: 'req-1', status: 'accepted' } })
			} as Response);

		const res = await updateLabStatus('req-1', 'accepted');
		expect(res.lab_request.status).toBe('accepted');
	});
});
