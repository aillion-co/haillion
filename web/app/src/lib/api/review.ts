import { ApiError } from './client';

const API_BASE_URL = 'http://localhost:8080/api';

export interface CreateReviewRequest {
	reviewer_id: string;
	reviewee_id: string;
	trip_id: string;
	rating: number;
	comment?: string;
}

export interface ReviewResponse {
	id: string;
	reviewer_id: string;
	reviewee_id: string;
	trip_id: string;
	rating: number;
	comment?: string;
}

export async function createReview(request: CreateReviewRequest): Promise<ReviewResponse> {
	const response = await fetch(`${API_BASE_URL}/review/reviews`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify(request)
	});

	if (!response.ok) {
		throw new ApiError(response.status, `Review submission failed with status ${response.status}`);
	}

	return response.json();
}
