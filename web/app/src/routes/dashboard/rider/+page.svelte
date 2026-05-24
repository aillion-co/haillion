<script lang="ts">
	import { onDestroy } from 'svelte';
	import { authStore } from '$lib/stores/auth';
	import { matchDriver } from '$lib/api/matching';
	import { createTrip, getTrip } from '$lib/api/trip';
	import TripStatus from '$lib/components/TripStatus.svelte';

	// Mock coordinates
	let lat = $state(37.7749);
	let lng = $state(-122.4194);

	let loading = $state(false);
	let activeTripId = $state<string | null>(null);
	let driverId = $state<string | null>(null);
	let eta = $state<number | null>(null);
	let tripState = $state<string | null>(null);
	let errorMessage = $state<string | null>(null);

	let pollInterval: ReturnType<typeof setInterval> | null = null;

	function stopPolling() {
		if (pollInterval) {
			clearInterval(pollInterval);
			pollInterval = null;
		}
	}

	onDestroy(() => {
		stopPolling();
	});

	async function handleRequestRide() {
		if (!$authStore) {
			errorMessage = 'You must be logged in to request a ride.';
			return;
		}

		loading = true;
		errorMessage = null;
		activeTripId = null;
		driverId = null;
		eta = null;
		tripState = null;
		stopPolling();

		try {
			// 1. POST to matching API
			const matchResponse = await matchDriver({
				rider_id: $authStore.id,
				lat,
				lng
			});

			driverId = matchResponse.driver_id;
			eta = matchResponse.eta_seconds;

			// 2. POST to trip creation API
			const tripResponse = await createTrip({
				rider_id: $authStore.id,
				lat,
				lng
			});

			activeTripId = tripResponse.id;
			tripState = tripResponse.state;

			// 3. Start polling trip status
			pollInterval = setInterval(async () => {
				if (!activeTripId) return;
				try {
					const currentTrip = await getTrip(activeTripId);
					tripState = currentTrip.state;
					if (currentTrip.driver_id) {
						driverId = currentTrip.driver_id;
					}

					if (currentTrip.state === 'completed') {
						stopPolling();
					}
				} catch (err) {
					console.error('Error polling trip status:', err);
				}
			}, 3000);
		} catch (err) {
			// "If the match API call fails, then the component shall display a "No drivers available" error."
			errorMessage = 'No drivers available';
			console.error(err);
		} finally {
			loading = false;
		}
	}
</script>

<div class="mx-auto max-w-2xl space-y-6 px-4 py-8">
	<div class="flex items-center justify-between border-b border-gray-200 pb-4">
		<div>
			<h1 class="text-3xl font-extrabold tracking-tight text-gray-900">Rider Dashboard</h1>
			<p class="mt-1 text-sm text-gray-500">Request a ride and track its progress in real-time.</p>
		</div>
		<a
			href="/"
			class="inline-flex items-center rounded-xl border border-gray-300 bg-white px-4 py-2 text-sm font-semibold text-gray-700 shadow-sm hover:bg-gray-50"
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
			<h2 class="mt-4 text-lg font-bold text-yellow-800">Authentication Required</h2>
			<p class="mt-2 text-sm text-yellow-700">
				Please register or log in first to use the Rider dashboard.
			</p>
			<div class="mt-4">
				<a
					href="/register"
					class="inline-flex items-center rounded-xl bg-yellow-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-yellow-500"
				>
					Register
				</a>
			</div>
		</div>
	{:else}
		<div class="grid gap-6 md:grid-cols-2">
			<!-- Controls Card -->
			<div class="rounded-2xl border border-gray-200 bg-white p-6 shadow-md">
				<h2 class="text-xl font-bold text-gray-900">Request a Ride</h2>
				<p class="mt-1 text-xs text-gray-500">
					Set coordinates or use defaults to match with local drivers.
				</p>

				<div class="mt-6 space-y-4">
					<div>
						<label
							for="latitude"
							class="block text-xs font-semibold tracking-wider text-gray-500 uppercase"
							>Latitude</label
						>
						<input
							type="number"
							id="latitude"
							step="0.0001"
							bind:value={lat}
							disabled={loading || !!activeTripId}
							class="mt-1 block w-full rounded-xl border border-gray-300 px-4 py-2.5 text-sm text-gray-900 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 disabled:bg-gray-50 disabled:text-gray-500"
						/>
					</div>

					<div>
						<label
							for="longitude"
							class="block text-xs font-semibold tracking-wider text-gray-500 uppercase"
							>Longitude</label
						>
						<input
							type="number"
							id="longitude"
							step="0.0001"
							bind:value={lng}
							disabled={loading || !!activeTripId}
							class="mt-1 block w-full rounded-xl border border-gray-300 px-4 py-2.5 text-sm text-gray-900 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 disabled:bg-gray-50 disabled:text-gray-500"
						/>
					</div>

					<button
						onclick={handleRequestRide}
						disabled={loading || (!!activeTripId && tripState !== 'completed')}
						data-testid="request-ride-btn"
						class="flex w-full items-center justify-center rounded-xl bg-indigo-600 px-5 py-3 text-base font-semibold text-white shadow-sm transition-colors hover:bg-indigo-500 focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 focus:outline-none disabled:cursor-not-allowed disabled:bg-indigo-400"
					>
						{#if loading}
							<svg
								class="mr-3 -ml-1 h-5 w-5 animate-spin text-white"
								xmlns="http://www.w3.org/2000/svg"
								fill="none"
								viewBox="0 0 24 24"
							>
								<circle
									class="opacity-25"
									cx="12"
									cy="12"
									r="10"
									stroke="currentColor"
									stroke-width="4"
								></circle>
								<path
									class="opacity-75"
									fill="currentColor"
									d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
								></path>
							</svg>
							Matching Driver...
						{:else}
							Request Ride
						{/if}
					</button>
				</div>
			</div>

			<!-- Status Column -->
			<div>
				<TripStatus {driverId} {eta} state={tripState} {errorMessage} />
			</div>
		</div>
	{/if}
</div>
