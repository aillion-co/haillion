<script lang="ts">
	import { goto } from '$app/navigation';
	import { registerUser, ApiError } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth';

	let email = $state('');
	let role = $state<'rider' | 'driver'>('rider');
	let loading = $state(false);
	let errorMessage = $state('');

	function generateUUID() {
		if (typeof crypto !== 'undefined' && crypto.randomUUID) {
			return crypto.randomUUID();
		}
		return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
			const r = (Math.random() * 16) | 0;
			const v = c === 'x' ? r : (r & 0x3) | 0x8;
			return v.toString(16);
		});
	}

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		loading = true;
		errorMessage = '';

		const id = generateUUID();

		try {
			const response = await registerUser({ id, email, role });
			authStore.set(response);
			await goto('/');
		} catch (err) {
			if (err instanceof ApiError && err.status === 409) {
				errorMessage = 'Email already registered';
			} else {
				errorMessage = 'An unexpected error occurred. Please try again.';
			}
		} finally {
			loading = false;
		}
	}
</script>

<div class="mx-auto max-w-md space-y-6 py-12">
	<div class="rounded-2xl border border-gray-200 bg-white p-8 shadow-md">
		<div class="text-center">
			<h1 class="text-2xl font-bold tracking-tight text-gray-900">Create an Account</h1>
			<p class="mt-2 text-sm text-gray-500">Sign up as a rider or driver to start using Aillion.</p>
		</div>

		<form onsubmit={handleSubmit} class="mt-8 space-y-6">
			{#if errorMessage}
				<div class="rounded-lg bg-red-50 p-4 text-sm text-red-700" role="alert">
					<div class="flex">
						<svg class="mr-2 h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
							<path
								fill-rule="evenodd"
								d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
								clip-rule="evenodd"
							/>
						</svg>
						<span class="font-medium" data-testid="error-message">{errorMessage}</span>
					</div>
				</div>
			{/if}

			<div class="space-y-1">
				<label for="email" class="block text-sm font-medium text-gray-700">Email Address</label>
				<input
					type="email"
					id="email"
					name="email"
					required
					bind:value={email}
					disabled={loading}
					placeholder="user@example.com"
					class="block w-full rounded-xl border border-gray-300 px-4 py-3 text-gray-900 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
				/>
			</div>

			<div class="space-y-2">
				<span class="block text-sm font-medium text-gray-700">Select Your Role</span>
				<div class="grid grid-cols-2 gap-4">
					<label
						class="flex cursor-pointer flex-col items-center justify-center rounded-xl border-2 p-4 transition-all hover:bg-gray-50 {role ===
						'rider'
							? 'border-indigo-600 bg-indigo-50/50 text-indigo-900'
							: 'border-gray-200 text-gray-700'}"
					>
						<input
							type="radio"
							name="role"
							value="rider"
							bind:group={role}
							disabled={loading}
							class="sr-only"
						/>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							class="mb-2 h-6 w-6"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
							/>
						</svg>
						<span class="text-sm font-semibold">Rider</span>
					</label>

					<label
						class="flex cursor-pointer flex-col items-center justify-center rounded-xl border-2 p-4 transition-all hover:bg-gray-50 {role ===
						'driver'
							? 'border-indigo-600 bg-indigo-50/50 text-indigo-900'
							: 'border-gray-200 text-gray-700'}"
					>
						<input
							type="radio"
							name="role"
							value="driver"
							bind:group={role}
							disabled={loading}
							class="sr-only"
						/>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							class="mb-2 h-6 w-6"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							stroke-width="2"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M9 17a2 2 0 11-4 0 2 2 0 014 0zM19 17a2 2 0 11-4 0 2 2 0 014 0z"
							/>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M13 16V6a1 1 0 00-1-1H4a1 1 0 00-1 1v10a1 1 0 001 1h1m8-1a1 1 0 01-1 1H9m4-1V8a1 1 0 011-1h2.586a1 1 0 01.707.293l2.414 2.414a1 1 0 01.293.707V16a1 1 0 01-1 1h-1m-6-1a1 1 0 001 1h1M5 17a2 2 0 104 0m-4 0a2 2 0 114 0m6 0a2 2 0 104 0m-4 0a2 2 0 114 0"
							/>
						</svg>
						<span class="text-sm font-semibold">Driver</span>
					</label>
				</div>
			</div>

			<button
				type="submit"
				disabled={loading}
				class="flex w-full items-center justify-center rounded-xl bg-indigo-600 px-5 py-3 text-base font-semibold text-white shadow-sm transition-colors hover:bg-indigo-500 focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 focus:outline-none disabled:cursor-not-allowed disabled:bg-indigo-400"
			>
				{#if loading}
					<svg
						class="mr-3 -ml-1 h-5 w-5 animate-spin text-white"
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
					>
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
						></circle>
						<path
							class="opacity-75"
							fill="currentColor"
							d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
						></path>
					</svg>
					Registering...
				{:else}
					Register
				{/if}
			</button>
		</form>
	</div>
</div>
