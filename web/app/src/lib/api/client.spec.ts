import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { registerUser, ApiError, type RegisterRequest, type RegisterResponse } from './client';
import { authStore } from '../stores/auth';

describe('API Client & Store Integration', () => {
	const mockRequest: RegisterRequest = {
		id: '123e4567-e89b-12d3-a456-426614174000',
		email: 'test@example.com',
		role: 'rider'
	};

	beforeEach(() => {
		authStore.set(null);
		vi.stubGlobal('fetch', vi.fn());
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('shall POST to register endpoint with ID, Email, and Role', async () => {
		const mockResponse = { id: mockRequest.id, email: mockRequest.email, role: mockRequest.role };

		const fetchMock = vi.fn().mockResolvedValue({
			ok: true,
			status: 200,
			json: async () => mockResponse
		});
		vi.stubGlobal('fetch', fetchMock);

		const result = await registerUser(mockRequest);

		expect(fetchMock).toHaveBeenCalledWith('http://localhost:8080/api/identity/users/register', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify(mockRequest)
		});
		expect(result).toEqual(mockResponse);
	});

	it('shall store the user profile in the authStore if registration is successful', async () => {
		const mockResponse = { id: mockRequest.id, email: mockRequest.email, role: mockRequest.role };

		vi.stubGlobal(
			'fetch',
			vi.fn().mockResolvedValue({
				ok: true,
				status: 200,
				json: async () => mockResponse
			})
		);

		const response = await registerUser(mockRequest);
		authStore.set(response);

		let currentUser: RegisterResponse | null = null;
		const unsubscribe = authStore.subscribe((val) => {
			currentUser = val;
		});
		unsubscribe();

		expect(currentUser).toEqual(mockResponse);
	});

	it('shall throw ApiError with 409 status when email is already registered', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn().mockResolvedValue({
				ok: false,
				status: 409,
				json: async () => ({ message: 'Conflict' })
			})
		);

		await expect(registerUser(mockRequest)).rejects.toThrow(ApiError);
		await expect(registerUser(mockRequest)).rejects.toHaveProperty('status', 409);
	});

	it('shall throw ApiError on other unexpected failure statuses', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn().mockResolvedValue({
				ok: false,
				status: 500,
				json: async () => ({ message: 'Internal Server Error' })
			})
		);

		await expect(registerUser(mockRequest)).rejects.toThrow(ApiError);
		await expect(registerUser(mockRequest)).rejects.toHaveProperty('status', 500);
	});
});
