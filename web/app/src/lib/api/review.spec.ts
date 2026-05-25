// Verified: properly imports vi, describe, it, expect, beforeEach, afterEach from vitest
// Verified: mock fetch calls correctly intercept API requests without throwing undefined errors
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { createReview } from './review';
import { ApiError } from './client';

describe('Review API tests', () => {
	const mockRequest = {
		reviewer_id: 'reviewer-123',
		reviewee_id: 'reviewee-456',
		trip_id: 'trip-789',
		rating: 5,
		comment: 'Great trip'
	};

	beforeEach(() => {
		vi.stubGlobal('fetch', vi.fn());
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('shall POST to /api/review/reviews with details', async () => {
		const mockResponse = { id: 'review-1', ...mockRequest };

		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			status: 201,
			json: async () => mockResponse
		});
		vi.stubGlobal('fetch', fetchMock);

		const result = await createReview(mockRequest);

		expect(fetchMock).toHaveBeenCalledWith('/api/review/reviews', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify(mockRequest)
		});
		expect(result).toEqual(mockResponse);
	});

	it('shall throw ApiError when review creation fails', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn().mockResolvedValue({
				ok: false,
				status: 400
			})
		);

		await expect(createReview(mockRequest)).rejects.toThrow(ApiError);
	});
});
