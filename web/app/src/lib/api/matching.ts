export interface MatchRequest {
	rider_id: string;
	lat: number;
	lng: number;
}

export interface MatchResponse {
	driver_id: string;
	eta_seconds: number;
}

export interface DriverLocationRequest {
	lat: number;
	lng: number;
}

import { API_BASE_URL } from './config';

export class ApiError extends Error {
	constructor(
		public status: number,
		message: string
	) {
		super(message);
		this.name = 'ApiError';
	}
}

export async function matchDriver(request: MatchRequest): Promise<MatchResponse> {
	const response = await fetch(`${API_BASE_URL}/matching/match`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify(request)
	});

	if (!response.ok) {
		throw new ApiError(response.status, `Matching failed with status ${response.status}`);
	}

	return response.json();
}

export async function updateDriverLocation(
	id: string,
	request: DriverLocationRequest
): Promise<void> {
	const response = await fetch(`${API_BASE_URL}/matching/drivers/${id}/location`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify(request)
	});

	if (!response.ok) {
		throw new ApiError(response.status, `Updating location failed with status ${response.status}`);
	}
}
