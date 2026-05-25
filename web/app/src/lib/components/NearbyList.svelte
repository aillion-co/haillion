<script lang="ts">
	interface Props {
		kind: 'driver' | 'rider';
		items: { id: string; distanceMeters: number }[];
		status: 'loading' | 'ok' | 'empty' | 'error' | 'idle';
	}

	let { kind, items = [], status }: Props = $props();

	function formatId(id: string): string {
		return id.length < 6 ? id : id.slice(0, 6);
	}

	function formatDistance(meters: number): string {
		return (meters / 1609.34).toFixed(1);
	}

	let sortedItems = $derived(
		[...items].sort((a, b) => a.distanceMeters - b.distanceMeters).slice(0, 10)
	);
</script>

{#if status !== 'idle'}
	<div
		class="rounded-2xl border border-gray-200 bg-white p-6 shadow-md"
		data-testid="nearby-list-container"
	>
		<h3 class="mb-3 text-sm font-semibold text-gray-900">
			Nearby {kind === 'driver' ? 'Drivers' : 'Riders'}
		</h3>

		{#if status === 'loading'}
			<p class="text-sm text-gray-500" data-testid="nearby-loading">
				Loading nearby {kind === 'driver' ? 'drivers' : 'riders'}…
			</p>
		{:else if status === 'error'}
			<p class="text-sm text-red-500" data-testid="nearby-error">
				Could not load nearby {kind === 'driver' ? 'drivers' : 'riders'}
			</p>
		{:else if status === 'empty' || sortedItems.length === 0}
			<p class="text-sm text-gray-400 italic" data-testid="nearby-empty">
				No {kind === 'driver' ? 'drivers' : 'riders'} nearby right now
			</p>
		{:else}
			<ul class="space-y-2 text-sm text-gray-600" data-testid="nearby-list">
				{#each sortedItems as item (item.id)}
					<li
						class="flex justify-between border-b border-gray-50 py-1 last:border-0"
						data-testid="nearby-item"
					>
						<span>
							{kind === 'driver' ? 'Driver' : 'Rider'}
							{formatId(item.id)} — {formatDistance(item.distanceMeters)} mi
						</span>
					</li>
				{/each}
			</ul>
		{/if}
	</div>
{/if}
