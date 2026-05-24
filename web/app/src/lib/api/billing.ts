import { ApiError } from './client';

const API_BASE_URL = 'http://localhost:8080/api';

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
