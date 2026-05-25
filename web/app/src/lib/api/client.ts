export interface RegisterRequest {
	id: string;
	email: string;
	role: 'rider' | 'driver';
}

export interface RegisterResponse {
	id: string;
	email: string;
	role: 'rider' | 'driver';
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

export async function registerUser(request: RegisterRequest): Promise<RegisterResponse> {
	const response = await fetch(`${API_BASE_URL}/identity/users/register`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify(request)
	});

	if (!response.ok) {
		throw new ApiError(response.status, `Registration failed with status ${response.status}`);
	}

	return response.json();
}
