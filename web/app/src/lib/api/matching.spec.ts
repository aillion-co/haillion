import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { matchDriver, ApiError } from './matching';
import { createTrip, getTrip } from './trip';

describe('Rider Matching & Trip API tests', () => {
	const mockRiderId = 'rider-123';
	const mockLat = 37.7749;
	const mockLng = -122.4194;

	beforeEach(() => {
		vi.stubGlobal('fetch', vi.fn());
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('1. When matchDriver is called, it shall POST to /api/matching/match with rider ID and coordinates', async () => {
		const mockResponse = { driver_id: 'driver-456', eta_seconds: 300 };

		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			status: 200,
			json: async () => mockResponse
		});
		vi.stubGlobal('fetch', fetchMock);

		const result = await matchDriver({
			rider_id: mockRiderId,
			lat: mockLat,
			lng: mockLng
		});

		expect(fetchMock).toHaveBeenCalledWith('http://localhost:8080/api/matching/match', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify({
				rider_id: mockRiderId,
				lat: mockLat,
				lng: mockLng
			})
		});
		expect(result).toEqual(mockResponse);
	});

	it('2. When matchDriver succeeds and createTrip is called, it shall POST to /api/trip/trips to create the trip', async () => {
		const mockResponse = {
			id: 'trip-789',
			rider_id: mockRiderId,
			driver_id: null,
			state: 'requested'
		};

		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			status: 201,
			json: async () => mockResponse
		});
		vi.stubGlobal('fetch', fetchMock);

		const result = await createTrip({
			rider_id: mockRiderId,
			lat: mockLat,
			lng: mockLng
		});

		expect(fetchMock).toHaveBeenCalledWith('http://localhost:8080/api/trip/trips', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify({
				rider_id: mockRiderId,
				lat: mockLat,
				lng: mockLng
			})
		});
		expect(result).toEqual(mockResponse);
	});

	it('3. While trip is active, getTrip shall GET /api/trip/trips/{id} to poll state', async () => {
		const tripId = 'trip-789';
		const mockResponse = {
			id: tripId,
			rider_id: mockRiderId,
			driver_id: 'driver-456',
			state: 'accepted'
		};

		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			status: 200,
			json: async () => mockResponse
		});
		vi.stubGlobal('fetch', fetchMock);

		const result = await getTrip(tripId);

		expect(fetchMock).toHaveBeenCalledWith(`http://localhost:8080/api/trip/trips/${tripId}`, {
			method: 'GET',
			headers: {
				'Content-Type': 'application/json'
			}
		});
		expect(result).toEqual(mockResponse);
	});

	it('4. If matchDriver API fails, it shall throw ApiError indicating matching failed', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: false,
			status: 500,
			json: async () => ({ error: 'No drivers available' })
		});
		vi.stubGlobal('fetch', fetchMock);

		await expect(
			matchDriver({
				rider_id: mockRiderId,
				lat: mockLat,
				lng: mockLng
			})
		).rejects.toThrow(ApiError);
	});
});
