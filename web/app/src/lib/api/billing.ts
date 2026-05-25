import { ApiError } from './client';
import { API_BASE_URL } from './config';

export interface CreatePaymentRequest {
	trip_id: string;
	amount: number;
}

export interface PaymentResponse {
	id: string;
	trip_id: string;
	amount: number;
	status: string;
}

export interface FareEstimate {
	distance_miles: number;
	base_pence: number;
	surge_multiplier: number;
	total_pence: number;
	minimum_applied: boolean;
}

export async function estimateFare(params: {
	from: string;
	to: string;
	riders: number;
	drivers: number;
}): Promise<FareEstimate> {
	const query = new URLSearchParams({
		from: params.from,
		to: params.to,
		riders: params.riders.toString(),
		drivers: params.drivers.toString()
	}).toString();

	const response = await fetch(`${API_BASE_URL}/billing/fare/estimate?${query}`, {
		method: 'GET',
		headers: {
			'Content-Type': 'application/json'
		}
	});

	if (!response.ok) {
		const errorMsg = await response.text();
		throw new ApiError(
			response.status,
			errorMsg || `Fare estimation failed with status ${response.status}`
		);
	}

	return response.json();
}

export async function createPayment(request: CreatePaymentRequest): Promise<PaymentResponse> {
	const response = await fetch(`${API_BASE_URL}/billing/payments`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify(request)
	});

	if (!response.ok) {
		throw new ApiError(response.status, `Payment creation failed with status ${response.status}`);
	}

	return response.json();
}

export async function processPayment(id: string): Promise<PaymentResponse> {
	const response = await fetch(`${API_BASE_URL}/billing/payments/${id}/process`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({})
	});

	if (!response.ok) {
		throw new ApiError(response.status, `Payment processing failed with status ${response.status}`);
	}

	return response.json();
}
