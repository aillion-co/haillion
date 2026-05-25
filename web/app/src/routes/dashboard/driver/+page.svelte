<script lang="ts">
	import { onMount } from 'svelte';
	import { authStore } from '$lib/stores/auth';
	import DriverControls from '$lib/components/DriverControls.svelte';
	import RatingForm from '$lib/components/RatingForm.svelte';

	let completedTripId = $state<string | null>(null);
	let lastCompletedTripRiderId = $state<string | null>(null);

	onMount(() => {
		const originalFetch = window.fetch;
		window.fetch = async (input, init) => {
			const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
			const response = await originalFetch(input, init);

			if (url.includes('/trip/trips/') && url.includes('/complete') && init?.method === 'POST') {
				if (response.ok) {
					const parts = url.split('/');
					const completeIndex = parts.indexOf('complete');
					const tripId = completeIndex > 0 ? parts[completeIndex - 1] : null;
					if (tripId) {
						completedTripId = tripId;
						try {
							const tripRes = await originalFetch(`/api/trip/trips/${tripId}`);
							if (tripRes.ok) {
								const tripData = await tripRes.json();
								lastCompletedTripRiderId = tripData.rider_id;
							}
						} catch (err) {
							console.error('Failed to fetch completed trip details:', err);
						}
					}
				}
			}

			return response;
		};

		return () => {
			window.fetch = originalFetch;
		};
	});
</script>

<div class="mx-auto max-w-2xl space-y-6 px-4 py-8">
	<div class="flex items-center justify-between border-b border-gray-200 pb-4">
		<div>
			<h1 class="font-sans text-3xl font-extrabold tracking-tight text-gray-900">
				Driver Dashboard
			</h1>
			<p class="mt-1 font-sans text-sm text-gray-500">
				Toggle online to receive trips and manage your status.
			</p>
		</div>
		<a
			href="/"
			class="inline-flex items-center rounded-xl border border-gray-300 bg-white px-4 py-2 font-sans text-sm font-semibold text-gray-700 shadow-sm hover:bg-gray-50"
		>
			Back to Home
		</a>
	</div>

	{#if !$authStore}
		<div class="rounded-2xl border border-yellow-200 bg-yellow-50 p-6 text-center shadow-sm">
			<svg
				class="mx-auto h-12 w-12 text-yellow-500"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
				/>
			</svg>
			<h2 class="mt-4 font-sans text-lg font-bold text-yellow-800">Authentication Required</h2>
			<p class="mt-2 font-sans text-sm text-yellow-700">
				Please register or log in first to use the Driver dashboard.
			</p>
			<div class="mt-4">
				<a
					href="/register"
					class="inline-flex items-center rounded-xl bg-yellow-600 px-4 py-2 font-sans text-sm font-semibold text-white shadow-sm hover:bg-yellow-500"
				>
					Register / Login
				</a>
			</div>
		</div>
	{:else if $authStore.role !== 'driver'}
		<div class="rounded-2xl border border-red-200 bg-red-50 p-6 text-center shadow-sm">
			<svg
				class="mx-auto h-12 w-12 text-red-500"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
				/>
			</svg>
			<h2 class="mt-4 font-sans text-lg font-bold text-red-800">Access Denied</h2>
			<p class="mt-2 font-sans text-sm text-red-700">
				This dashboard is only accessible to drivers. Your current role is "{$authStore.role}".
			</p>
			<div class="mt-4">
				<a
					href="/dashboard/rider"
					class="inline-flex items-center rounded-xl bg-red-600 px-4 py-2 font-sans text-sm font-semibold text-white shadow-sm hover:bg-red-500"
				>
					Go to Rider Dashboard
				</a>
			</div>
		</div>
	{:else if completedTripId && lastCompletedTripRiderId}
		<div class="space-y-6">
			<RatingForm
				tripId={completedTripId}
				reviewerId={$authStore.id}
				revieweeId={lastCompletedTripRiderId}
			/>
			<div class="flex justify-end">
				<button
					onclick={() => {
						completedTripId = null;
						lastCompletedTripRiderId = null;
					}}
					data-testid="back-to-dashboard-btn"
					class="rounded-xl border border-gray-300 bg-white px-4 py-2 text-sm font-semibold text-gray-700 shadow-sm hover:bg-gray-50"
				>
					Back to Dashboard
				</button>
			</div>
		</div>
	{:else}
		<DriverControls driverId={$authStore.id} />
	{/if}
</div>
