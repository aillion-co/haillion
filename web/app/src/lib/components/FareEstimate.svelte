<script lang="ts">
	import { estimateFare, type FareEstimate } from '$lib/api/billing';
	import { ApiError } from '$lib/api/client';

	let {
		pickup = '',
		destination = '',
		riders = 0,
		drivers = 0
	}: {
		pickup?: string;
		destination?: string;
		riders?: number;
		drivers?: number;
	} = $props();

	let estimate = $state<FareEstimate | null>(null);
	let loading = $state(false);
	let errorText = $state<string | null>(null);

	function formatPence(pence: number): string {
		return '£' + (pence / 100).toFixed(2);
	}

	function formatDistance(miles: number): string {
		return miles.toFixed(1);
	}

	$effect(() => {
		const p = pickup.trim();
		const d = destination.trim();

		if (!p || !d) {
			estimate = null;
			loading = false;
			errorText = null;
			return;
		}

		loading = true;
		errorText = null;

		const timeout = setTimeout(async () => {
			try {
				const res = await estimateFare({
					from: p,
					to: d,
					riders,
					drivers
				});
				estimate = res;
			} catch (err) {
				if (err instanceof ApiError) {
					errorText = err.message;
				} else {
					errorText = err instanceof Error ? err.message : 'Unknown error';
				}
				estimate = null;
			} finally {
				loading = false;
			}
		}, 400);

		return () => {
			clearTimeout(timeout);
		};
	});
</script>

{#if pickup.trim() && destination.trim()}
	<div
		class="rounded-2xl border border-gray-200 bg-white p-6 shadow-md"
		data-testid="fare-estimate-card"
	>
		<h3 class="mb-3 text-sm font-semibold text-gray-900">Fare Estimate</h3>
		{#if loading}
			<p class="text-sm text-gray-500" data-testid="fare-loading">Calculating…</p>
		{:else if errorText}
			<p class="text-sm text-gray-500" data-testid="fare-error">{errorText}</p>
		{:else if estimate}
			<div class="space-y-2 text-sm text-gray-600" data-testid="fare-breakdown">
				<div data-testid="fare-distance">
					Distance: {formatDistance(estimate.distance_miles)} mi
				</div>
				<div data-testid="fare-base">Base: {formatPence(estimate.base_pence)}</div>
				{#if estimate.surge_multiplier !== 1}
					<div data-testid="fare-surge">Surge: ×{estimate.surge_multiplier.toFixed(2)}</div>
				{/if}
				<div
					class="border-t border-gray-100 pt-2 font-semibold text-gray-900"
					data-testid="fare-total"
				>
					Estimated total: {formatPence(estimate.total_pence)}{estimate.minimum_applied
						? ' (minimum fare)'
						: ''}
				</div>
			</div>
		{/if}
	</div>
{/if}
