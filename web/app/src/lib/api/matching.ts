export interface MatchRequest {
	rider_id: string;
	lat: number;
	lng: number;
}

export interface MatchResponse {
	driver_id: string;
	eta_seconds: number;
}

const API_BASE_URL = 'http://localhost:8080/api';

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
