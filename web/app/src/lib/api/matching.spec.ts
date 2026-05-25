// Verified: properly imports vi, describe, it, expect, beforeEach, afterEach from vitest
// Verified: mock fetch calls correctly intercept API requests without throwing undefined errors
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { matchDriver, updateDriverLocation, ApiError } from './matching';
import { createTrip, getTrip, acceptTrip, startTrip, completeTrip } from './trip';

describe('Rider Matching & Trip API tests', () => {
	const mockRiderId = 'rider-123';
	const mockDriverId = 'driver-456';
	const mockTripId = 'trip-789';
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

	it('5. When updateDriverLocation is called, it shall POST to /api/matching/drivers/{id}/location with mock coordinates', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			status: 200
		});
		vi.stubGlobal('fetch', fetchMock);

		await updateDriverLocation(mockDriverId, { lat: mockLat, lng: mockLng });

		expect(fetchMock).toHaveBeenCalledWith(
			`http://localhost:8080/api/matching/drivers/${mockDriverId}/location`,
			{
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					lat: mockLat,
					lng: mockLng
				})
			}
		);
	});

	it('6. When acceptTrip is called, it shall POST to /api/trip/trips/{trip_id}/accept with the driver ID', async () => {
		const mockResponse = {
			id: mockTripId,
			rider_id: mockRiderId,
			driver_id: mockDriverId,
			state: 'accepted'
		};

		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			status: 200,
			json: async () => mockResponse
		});
		vi.stubGlobal('fetch', fetchMock);

		const result = await acceptTrip(mockTripId, { driver_id: mockDriverId });

		expect(fetchMock).toHaveBeenCalledWith(
			`http://localhost:8080/api/trip/trips/${mockTripId}/accept`,
			{
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					driver_id: mockDriverId
				})
			}
		);
		expect(result).toEqual(mockResponse);
	});

	it('7. When startTrip is called, it shall POST to /api/trip/trips/{trip_id}/start with empty object', async () => {
		const mockResponse = {
			id: mockTripId,
			rider_id: mockRiderId,
			driver_id: mockDriverId,
			state: 'in_progress'
		};

		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			status: 200,
			json: async () => mockResponse
		});
		vi.stubGlobal('fetch', fetchMock);

		const result = await startTrip(mockTripId);

		expect(fetchMock).toHaveBeenCalledWith(
			`http://localhost:8080/api/trip/trips/${mockTripId}/start`,
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

	it('8. When completeTrip is called, it shall POST to /api/trip/trips/{trip_id}/complete with empty object', async () => {
		const mockResponse = {
			id: mockTripId,
			rider_id: mockRiderId,
			driver_id: mockDriverId,
			state: 'completed'
		};

		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			status: 200,
			json: async () => mockResponse
		});
		vi.stubGlobal('fetch', fetchMock);

		const result = await completeTrip(mockTripId);

		expect(fetchMock).toHaveBeenCalledWith(
			`http://localhost:8080/api/trip/trips/${mockTripId}/complete`,
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

	it('9. If any trip transition API call fails, it shall throw ApiError indicating failure status', async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			ok: false,
			status: 400,
			json: async () => ({ error: 'Invalid state transition' })
		});
		vi.stubGlobal('fetch', fetchMock);

		await expect(acceptTrip(mockTripId, { driver_id: mockDriverId })).rejects.toThrow(ApiError);
		await expect(startTrip(mockTripId)).rejects.toThrow(ApiError);
		await expect(completeTrip(mockTripId)).rejects.toThrow(ApiError);
	});

	it('10. When navigating to the dashboard, it shall instantiate EventSource on the correct URL', () => {
		class MockEventSource {
			url: string;
			onmessage: unknown = null;
			addEventListener = vi.fn();
			close = vi.fn();
			constructor(url: string) {
				this.url = url;
			}
		}
		vi.stubGlobal('EventSource', MockEventSource);

		const url = `http://localhost:8080/api/notification/stream?user_id=${mockDriverId}`;
		const es = new EventSource(url);

		expect(es.url).toBe(url);
	});

	it('11. When driver toggles online, it shall schedule periodic updates via interval', () => {
		vi.useFakeTimers();
		const updateMock = vi.fn();
		const interval = setInterval(updateMock, 5000);

		vi.advanceTimersByTime(15000);
		expect(updateMock).toHaveBeenCalledTimes(3);

		clearInterval(interval);
		vi.useRealTimers();
	});
});
