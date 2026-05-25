export interface MatchRequest {
	rider_id: string;
	lat?: number;
	lng?: number;
	postcode?: string;
}

export interface MatchResponse {
	driver_id: string;
	eta_seconds: number;
}

export interface DriverLocationRequest {
	lat?: number;
	lng?: number;
	postcode?: string;
}

export interface PostcodeLocationRequest {
	postcode: string;
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
		const text = await response.text();
		throw new ApiError(
			response.status,
			text.trim() || `Matching failed with status ${response.status}`
		);
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
		const text = await response.text();
		throw new ApiError(
			response.status,
			text.trim() || `Updating location failed with status ${response.status}`
		);
	}
}

export async function requestRide(riderId: string, req: PostcodeLocationRequest): Promise<void> {
	const response = await fetch(`${API_BASE_URL}/matching/riders/${riderId}/request`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify(req)
	});

	if (!response.ok) {
		const text = await response.text();
		throw new ApiError(
			response.status,
			text.trim() || `Requesting ride failed with status ${response.status}`
		);
	}
}

export async function cancelRide(riderId: string): Promise<void> {
	const response = await fetch(`${API_BASE_URL}/matching/riders/${riderId}/cancel`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		}
	});

	if (!response.ok) {
		const text = await response.text();
		throw new ApiError(
			response.status,
			text.trim() || `Canceling ride failed with status ${response.status}`
		);
	}
}
