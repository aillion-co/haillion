export interface CreateTripRequest {
	rider_id: string;
	lat: number;
	lng: number;
}

export interface TripResponse {
	id: string;
	rider_id: string;
	driver_id?: string | null;
	state: 'requested' | 'accepted' | 'in_progress' | 'completed';
}

const API_BASE_URL = 'http://localhost:8080/api';

import { ApiError } from './matching';

export async function createTrip(request: CreateTripRequest): Promise<TripResponse> {
	const response = await fetch(`${API_BASE_URL}/trip/trips`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify(request)
	});

	if (!response.ok) {
		throw new ApiError(response.status, `Trip creation failed with status ${response.status}`);
	}

	return response.json();
}

export async function getTrip(id: string): Promise<TripResponse> {
	const response = await fetch(`${API_BASE_URL}/trip/trips/${id}`, {
		method: 'GET',
		headers: {
			'Content-Type': 'application/json'
		}
	});

	if (!response.ok) {
		throw new ApiError(response.status, `Failed to fetch trip with status ${response.status}`);
	}

	return response.json();
}
