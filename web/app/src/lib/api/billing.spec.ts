// Verified: properly imports vi, describe, it, expect, beforeEach, afterEach from vitest
// Verified: mock fetch calls correctly intercept API requests without throwing undefined errors
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { createPayment, processPayment } from './billing';
import { ApiError } from './client';

describe('Billing API tests', () => {
	const mockTripId = 'trip-123';
	const mockAmount = 1500;
	const mockPaymentId = 'pay-456';

	beforeEach(() => {
		vi.stubGlobal('fetch', vi.fn());
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('shall POST to /api/billing/payments with trip_id and amount', async () => {
		const mockResponse = {
			id: mockPaymentId,
			trip_id: mockTripId,
			amount: mockAmount,
			status: 'created'
		};

		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			status: 201,
			json: async () => mockResponse
		});
		vi.stubGlobal('fetch', fetchMock);

		const result = await createPayment({ trip_id: mockTripId, amount: mockAmount });

		expect(fetchMock).toHaveBeenCalledWith('http://localhost:8080/api/billing/payments', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify({ trip_id: mockTripId, amount: mockAmount })
		});
		expect(result).toEqual(mockResponse);
	});

	it('shall POST to /api/billing/payments/{id}/process to simulate payment', async () => {
		const mockResponse = {
			id: mockPaymentId,
			trip_id: mockTripId,
			amount: mockAmount,
			status: 'processed'
		};

		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			status: 200,
			json: async () => mockResponse
		});
		vi.stubGlobal('fetch', fetchMock);

		const result = await processPayment(mockPaymentId);

		expect(fetchMock).toHaveBeenCalledWith(
			`http://localhost:8080/api/billing/payments/${mockPaymentId}/process`,
			{
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({})
			}
		);
		expect(result).toEqual(mockResponse);
	});

	it('shall throw ApiError when payment creation fails', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn().mockResolvedValue({
				ok: false,
				status: 400
			})
		);

		await expect(createPayment({ trip_id: mockTripId, amount: mockAmount })).rejects.toThrow(
			ApiError
		);
	});

	it('shall throw ApiError when payment processing fails', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn().mockResolvedValue({
				ok: false,
				status: 500
			})
		);

		await expect(processPayment(mockPaymentId)).rejects.toThrow(ApiError);
	});
});
