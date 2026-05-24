<script lang="ts">
	import { onMount } from 'svelte';
	import { updateDriverLocation } from '$lib/api/matching';
	import { acceptTrip, startTrip, completeTrip } from '$lib/api/trip';

	interface Props {
		driverId: string;
	}

	let { driverId }: Props = $props();

	let isOnline = $state(false);
	let lat = $state(37.7749);
	let lng = $state(-122.4194);
	let activeTripId = $state<string | null>(null);
	let activeTripState = $state<'requested' | 'accepted' | 'in_progress' | 'completed' | null>(null);
	let errorMessage = $state<string | null>(null);
	let statusMessage = $state<string | null>(null);
	let sseConnected = $state(false);
	let manualTripId = $state('');

	let eventSource: EventSource | null = null;
	let locationInterval: ReturnType<typeof setInterval> | null = null;

	onMount(() => {
		// Connect to EventSource
		const url = `http://localhost:8080/api/notification/stream?user_id=${driverId}`;
		eventSource = new EventSource(url);

		eventSource.onopen = () => {
			sseConnected = true;
		};

		eventSource.onerror = () => {
			sseConnected = false;
		};

		const handleEvent = (dataStr: string) => {
			try {
				const data = JSON.parse(dataStr);
				const incomingTripId =
					data.trip_id ||
					data.id ||
					(data.payload && (data.payload.trip_id || data.payload.id || data.payload.tripId));
				if (incomingTripId) {
					activeTripId = incomingTripId;
					activeTripState = data.state || data.status || 'requested';
					statusMessage = `New trip request received: ${incomingTripId}`;
					errorMessage = null;
				}
			} catch (err) {
				console.error('Failed to parse SSE event data:', err);
			}
		};

		eventSource.onmessage = (event) => {
			handleEvent(event.data);
		};

		eventSource.addEventListener('trip_created', (event) => {
			handleEvent(event.data);
		});

		eventSource.addEventListener('trip_requested', (event) => {
			handleEvent(event.data);
		});

		return () => {
			if (eventSource) {
				eventSource.close();
			}
			if (locationInterval) {
				clearInterval(locationInterval);
			}
		};
	});

	async function sendLocationUpdate() {
		try {
			await updateDriverLocation(driverId, { lat, lng });
			// Slight random drift for realistic simulation
			lat += (Math.random() - 0.5) * 0.0001;
			lng += (Math.random() - 0.5) * 0.0001;
			errorMessage = null;
		} catch (err) {
			const msg = err instanceof Error ? err.message : String(err);
			errorMessage = `Failed to send location update: ${msg}`;
		}
	}

	function handleOnlineToggle() {
		if (isOnline) {
			// Start periodic updates
			sendLocationUpdate();
			locationInterval = setInterval(sendLocationUpdate, 5000);
		} else {
			// Stop periodic updates
			if (locationInterval) {
				clearInterval(locationInterval);
				locationInterval = null;
			}
		}
	}

	async function handleAccept() {
		if (!activeTripId) return;
		try {
			errorMessage = null;
			const res = await acceptTrip(activeTripId, { driver_id: driverId });
			activeTripState = res.state;
			statusMessage = `Accepted trip ${activeTripId}`;
		} catch (err) {
			const msg = err instanceof Error ? err.message : String(err);
			errorMessage = `Failed to accept trip: ${msg}`;
		}
	}

	async function handleStart() {
		if (!activeTripId) return;
		try {
			errorMessage = null;
			const res = await startTrip(activeTripId);
			activeTripState = res.state;
			statusMessage = `Started trip ${activeTripId}`;
		} catch (err) {
			const msg = err instanceof Error ? err.message : String(err);
			errorMessage = `Failed to start trip: ${msg}`;
		}
	}

	async function handleComplete() {
		if (!activeTripId) return;
		try {
			errorMessage = null;
			const res = await completeTrip(activeTripId);
			activeTripState = res.state;
			statusMessage = `Completed trip ${activeTripId}`;
		} catch (err) {
			const msg = err instanceof Error ? err.message : String(err);
			errorMessage = `Failed to complete trip: ${msg}`;
		}
	}

	function setManualTrip() {
		if (manualTripId.trim()) {
			activeTripId = manualTripId.trim();
			activeTripState = 'requested';
			statusMessage = `Manually set active trip ID to ${activeTripId}`;
			errorMessage = null;
			manualTripId = '';
		}
	}

	function clearActiveTrip() {
		activeTripId = null;
		activeTripState = null;
		statusMessage = 'Active trip cleared';
		errorMessage = null;
	}
</script>

<div class="space-y-6">
	<!-- Online/Offline & Status -->
	<div class="rounded-2xl border border-gray-200 bg-white p-6 shadow-md">
		<div class="flex items-center justify-between">
			<div>
				<h2 class="text-xl font-bold text-gray-900">Driver Status</h2>
				<p class="mt-1 text-xs text-gray-500">
					Go online to receive matches and send location updates.
				</p>
			</div>
			<div class="flex items-center space-x-3">
				<span class="text-sm font-semibold {isOnline ? 'text-green-600' : 'text-gray-500'}">
					{isOnline ? 'Online' : 'Offline'}
				</span>
				<label class="relative inline-flex cursor-pointer items-center">
					<input
						type="checkbox"
						bind:checked={isOnline}
						onchange={handleOnlineToggle}
						data-testid="online-toggle"
						class="peer sr-only"
					/>
					<div
						class="peer h-6 w-11 rounded-full bg-gray-200 peer-checked:bg-green-600 peer-focus:ring-4 peer-focus:ring-green-300 peer-focus:outline-none after:absolute after:top-[2px] after:left-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-gray-300 after:bg-white after:transition-all after:content-[''] peer-checked:after:translate-x-full peer-checked:after:border-white dark:border-gray-600"
					></div>
				</label>
			</div>
		</div>

		<!-- Notification Connection Status -->
		<div
			class="mt-4 flex items-center space-x-2 border-t border-gray-100 pt-4 text-xs text-gray-500"
		>
			<span class="relative flex h-2 w-2">
				<span
					class="absolute inline-flex h-full w-full animate-ping rounded-full opacity-75 {sseConnected
						? 'bg-green-400'
						: 'bg-red-400'}"
				></span>
				<span
					class="relative inline-flex h-2 w-2 rounded-full {sseConnected
						? 'bg-green-500'
						: 'bg-red-500'}"
				></span>
			</span>
			<span data-testid="stream-status">
				Notification Stream: {sseConnected ? 'Connected' : 'Disconnected'}
			</span>
		</div>

		<!-- Coordinates inputs -->
		<div class="mt-4 grid grid-cols-2 gap-4">
			<div>
				<label
					for="driver-lat"
					class="block text-xs font-semibold tracking-wider text-gray-500 uppercase">Latitude</label
				>
				<input
					id="driver-lat"
					type="number"
					step="0.0001"
					bind:value={lat}
					class="mt-1 block w-full rounded-xl border border-gray-300 px-3 py-2 text-sm text-gray-900 shadow-sm focus:border-indigo-500 focus:ring-indigo-500"
				/>
			</div>
			<div>
				<label
					for="driver-lng"
					class="block text-xs font-semibold tracking-wider text-gray-500 uppercase"
					>Longitude</label
				>
				<input
					id="driver-lng"
					type="number"
					step="0.0001"
					bind:value={lng}
					class="mt-1 block w-full rounded-xl border border-gray-300 px-3 py-2 text-sm text-gray-900 shadow-sm focus:border-indigo-500 focus:ring-indigo-500"
				/>
			</div>
		</div>
	</div>

	<!-- Active Trip Management -->
	<div class="rounded-2xl border border-gray-200 bg-white p-6 shadow-md">
		<h2 class="text-xl font-bold text-gray-900">Trip Management</h2>
		<p class="mt-1 text-xs text-gray-500">Manage your currently assigned trip lifecycle.</p>

		{#if activeTripId}
			<div
				class="mt-4 rounded-xl border border-indigo-100 bg-indigo-50 p-4"
				data-testid="active-trip-card"
			>
				<div class="flex items-center justify-between">
					<div>
						<h3 class="text-sm font-bold text-indigo-900">Trip Assigned</h3>
						<p class="mt-1 text-xs text-indigo-700" data-testid="trip-id-display">
							ID: {activeTripId}
						</p>
					</div>
					<span
						class="inline-flex rounded-full px-2.5 py-0.5 text-xs font-semibold tracking-wide uppercase
						{activeTripState === 'requested' ? 'bg-yellow-100 text-yellow-800' : ''}
						{activeTripState === 'accepted' ? 'bg-blue-100 text-blue-800' : ''}
						{activeTripState === 'in_progress' ? 'bg-orange-100 text-orange-800' : ''}
						{activeTripState === 'completed' ? 'bg-green-100 text-green-800' : ''}"
						data-testid="trip-state-display"
					>
						{activeTripState}
					</span>
				</div>

				<div class="mt-4 flex gap-2">
					{#if activeTripState === 'requested'}
						<button
							onclick={handleAccept}
							data-testid="accept-btn"
							class="flex-1 rounded-xl bg-indigo-600 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500"
						>
							Accept Trip
						</button>
					{:else}
						<button
							disabled
							class="flex-1 cursor-not-allowed rounded-xl bg-gray-200 py-2 text-sm font-semibold text-gray-400"
						>
							Accept Trip
						</button>
					{/if}

					{#if activeTripState === 'accepted'}
						<button
							onclick={handleStart}
							data-testid="start-btn"
							class="flex-1 rounded-xl bg-indigo-600 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500"
						>
							Start Trip
						</button>
					{:else}
						<button
							disabled
							class="flex-1 cursor-not-allowed rounded-xl bg-gray-200 py-2 text-sm font-semibold text-gray-400"
						>
							Start Trip
						</button>
					{/if}

					{#if activeTripState === 'in_progress'}
						<button
							onclick={handleComplete}
							data-testid="complete-btn"
							class="flex-1 rounded-xl bg-indigo-600 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500"
						>
							Complete Trip
						</button>
					{:else}
						<button
							disabled
							class="flex-1 cursor-not-allowed rounded-xl bg-gray-200 py-2 text-sm font-semibold text-gray-400"
						>
							Complete Trip
						</button>
					{/if}
				</div>

				<div class="mt-4 flex justify-end">
					<button
						onclick={clearActiveTrip}
						class="text-xs font-semibold text-indigo-600 hover:text-indigo-500 hover:underline"
					>
						Clear Trip
					</button>
				</div>
			</div>
		{:else}
			<div class="mt-4 border border-dashed border-gray-200 py-8 text-center text-sm text-gray-500">
				Waiting for assigned trips...
			</div>
		{/if}

		<!-- Fallback manual input -->
		<div class="mt-6 border-t border-gray-100 pt-4">
			<label
				for="manual-trip"
				class="block text-xs font-semibold tracking-wider text-gray-500 uppercase"
				>Manual Trip ID Fallback</label
			>
			<div class="mt-2 flex gap-2">
				<input
					id="manual-trip"
					type="text"
					bind:value={manualTripId}
					placeholder="Enter trip_id to override/simulate"
					class="block w-full rounded-xl border border-gray-300 px-3 py-2 text-sm text-gray-900 shadow-sm focus:border-indigo-500 focus:ring-indigo-500"
				/>
				<button
					onclick={setManualTrip}
					data-testid="manual-trip-btn"
					class="rounded-xl bg-gray-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-gray-500"
				>
					Set
				</button>
			</div>
		</div>
	</div>

	<!-- Notifications / Alerts -->
	{#if errorMessage}
		<div
			class="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-800"
			data-testid="driver-error-message"
		>
			<div class="flex items-center space-x-2">
				<svg class="h-5 w-5 text-red-500" fill="currentColor" viewBox="0 0 20 20">
					<path
						fill-rule="evenodd"
						d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
						clip-rule="evenodd"
					/>
				</svg>
				<span>{errorMessage}</span>
			</div>
		</div>
	{/if}

	{#if statusMessage}
		<div class="rounded-xl border border-green-200 bg-green-50 p-4 text-sm text-green-800">
			<div class="flex items-center space-x-2">
				<svg class="h-5 w-5 text-green-500" fill="currentColor" viewBox="0 0 20 20">
					<path
						fill-rule="evenodd"
						d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
						clip-rule="evenodd"
					/>
				</svg>
				<span>{statusMessage}</span>
			</div>
		</div>
	{/if}
</div>
