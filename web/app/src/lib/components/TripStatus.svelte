<script lang="ts">
	interface Props {
		driverId?: string | null;
		eta?: number | null;
		state?: string | null;
		errorMessage?: string | null;
	}

	let { driverId = null, eta = null, state = null, errorMessage = null }: Props = $props();

	// Helper to format ETA seconds to minutes
	function formatEta(seconds: number | null): string {
		if (seconds === null || seconds === undefined) return '';
		if (seconds < 60) return `${seconds} seconds`;
		const mins = Math.round(seconds / 60);
		return `${mins} min${mins > 1 ? 's' : ''}`;
	}

	// Helper to display clean status text
	function getStatusText(status: string | null): string {
		if (!status) return '';
		switch (status) {
			case 'requested':
				return 'Waiting for driver...';
			case 'accepted':
				return 'Driver accepted! On the way';
			case 'in_progress':
				return 'Trip in progress...';
			case 'completed':
				return 'Trip completed!';
			default:
				return status;
		}
	}
</script>

<div
	class="mt-6 rounded-2xl border border-gray-200 bg-white p-6 shadow-md"
	data-testid="trip-status-card"
>
	<h3 class="text-lg font-bold text-gray-900">Trip Status</h3>

	{#if errorMessage}
		<div
			class="mt-4 rounded-lg bg-red-50 p-4 text-sm text-red-700"
			role="alert"
			data-testid="error-container"
		>
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
	{:else if state}
		<div class="mt-4 space-y-4" data-testid="status-details">
			<div class="flex items-center justify-between border-b border-gray-100 pb-3">
				<span class="text-sm font-medium text-gray-500">Status</span>
				<span
					class="rounded-full bg-indigo-50 px-3 py-1 text-xs font-semibold tracking-wider text-indigo-700 uppercase"
					data-testid="trip-state"
				>
					{state}
				</span>
			</div>

			<div class="flex items-center justify-between border-b border-gray-100 pb-3">
				<span class="text-sm font-medium text-gray-500">Progress</span>
				<span class="text-sm font-semibold text-gray-800" data-testid="status-text">
					{getStatusText(state)}
				</span>
			</div>

			{#if driverId}
				<div class="flex items-center justify-between border-b border-gray-100 pb-3">
					<span class="text-sm font-medium text-gray-500">Driver ID</span>
					<span class="font-mono text-sm text-gray-700" data-testid="driver-id">{driverId}</span>
				</div>
			{/if}

			{#if eta !== null}
				<div class="flex items-center justify-between">
					<span class="text-sm font-medium text-gray-500">ETA</span>
					<span class="text-sm font-semibold text-gray-800" data-testid="eta-value"
						>{formatEta(eta)}</span
					>
				</div>
			{/if}
		</div>
	{:else}
		<p class="mt-2 text-sm text-gray-500" data-testid="no-active-trip">No active trip requests.</p>
	{/if}
</div>
