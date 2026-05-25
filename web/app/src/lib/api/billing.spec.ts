// Verified: properly imports vi, describe, it, expect, beforeEach, afterEach from vitest
// Verified: mock fetch calls correctly intercept API requests without throwing undefined errors
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { createPayment, processPayment, estimateFare } from './billing';
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

		expect(fetchMock).toHaveBeenCalledWith('/api/billing/payments', {
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

		expect(fetchMock).toHaveBeenCalledWith(`/api/billing/payments/${mockPaymentId}/process`, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify({})
		});
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

	describe('estimateFare', () => {
		it('shall GET /api/billing/fare/estimate with query parameters', async () => {
			const mockEstimate = {
				distance_miles: 5.5,
				base_pence: 500,
				surge_multiplier: 1.2,
				total_pence: 600,
				minimum_applied: false
			};

			const fetchMock = vi.fn().mockResolvedValue({
				ok: true,
				status: 200,
				json: async () => mockEstimate
			});
			vi.stubGlobal('fetch', fetchMock);

			const result = await estimateFare({
				from: 'SW1A 1AA',
				to: 'EC1A 1BB',
				riders: 2,
				drivers: 5
			});

			expect(fetchMock).toHaveBeenCalledWith(
				'/api/billing/fare/estimate?from=SW1A+1AA&to=EC1A+1BB&riders=2&drivers=5',
				{
					method: 'GET',
					headers: {
						'Content-Type': 'application/json'
					}
				}
			);
			expect(result).toEqual(mockEstimate);
		});

		it('shall throw ApiError with error response text when fare estimate fails', async () => {
			const fetchMock = vi.fn().mockResolvedValue({
				ok: false,
				status: 400,
				text: async () => 'postcode not found'
			});
			vi.stubGlobal('fetch', fetchMock);

			await expect(
				estimateFare({
					from: 'SW1A 1AA',
					to: 'EC1A 1BB',
					riders: 0,
					drivers: 0
				})
			).rejects.toThrowError('postcode not found');
		});
	});
});
