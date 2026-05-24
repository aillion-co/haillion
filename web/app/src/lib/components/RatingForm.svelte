<script lang="ts">
	import { createReview } from '$lib/api/review';

	interface Props {
		tripId: string;
		reviewerId: string;
		revieweeId: string;
		onSuccess?: () => void;
	}

	let { tripId, reviewerId, revieweeId, onSuccess }: Props = $props();

	let rating = $state(5);
	let comment = $state('');
	let status = $state<'idle' | 'submitting' | 'success' | 'failed'>('idle');
	let errorMessage = $state<string | null>(null);

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		status = 'submitting';
		errorMessage = null;

		try {
			await createReview({
				reviewer_id: reviewerId,
				reviewee_id: revieweeId,
				trip_id: tripId,
				rating,
				comment: comment.trim() || undefined
			});
			status = 'success';
			if (onSuccess) onSuccess();
		} catch {
			status = 'failed';
			errorMessage = 'Failed to submit review. Please try again.';
		}
	}
</script>

<div
	class="rounded-2xl border border-gray-200 bg-white p-6 shadow-md"
	data-testid="rating-form-container"
>
	<h2 class="text-xl font-bold text-gray-900">Rate your experience</h2>
	<p class="mt-1 text-sm text-gray-500">Your feedback helps us improve the network.</p>

	{#if status === 'success'}
		<div class="mt-6 text-center" data-testid="rating-success-message">
			<svg
				class="mx-auto h-12 w-12 text-green-500"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
			>
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
			</svg>
			<h3 class="mt-4 text-lg font-bold text-gray-900">Thank you!</h3>
			<p class="mt-2 text-sm text-gray-600">Your rating and review have been submitted.</p>
		</div>
	{:else}
		<form onsubmit={handleSubmit} class="mt-6 space-y-4">
			<div>
				<span class="block text-sm font-medium text-gray-700">Rating</span>
				<div class="mt-2 flex items-center space-x-2">
					{#each [1, 2, 3, 4, 5] as star (star)}
						<button
							type="button"
							onclick={() => (rating = star)}
							data-testid="star-btn-{star}"
							class="text-3xl focus:outline-none"
							aria-label="Rate {star} stars"
						>
							{#if star <= rating}
								<span class="text-yellow-400">★</span>
							{:else}
								<span class="text-gray-300">★</span>
							{/if}
						</button>
					{/each}
				</div>
			</div>

			<div>
				<label for="review-comment" class="block text-sm font-medium text-gray-700"
					>Comment (optional)</label
				>
				<textarea
					id="review-comment"
					bind:value={comment}
					data-testid="comment-textarea"
					rows={3}
					class="mt-1 block w-full rounded-xl border border-gray-300 px-4 py-2 text-sm text-gray-900 shadow-sm focus:border-indigo-500 focus:ring-indigo-500"
					placeholder="Tell us about your trip..."
				></textarea>
			</div>

			{#if errorMessage}
				<div
					class="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-800"
					data-testid="rating-error"
				>
					<p>{errorMessage}</p>
				</div>
			{/if}

			<button
				type="submit"
				disabled={status === 'submitting'}
				data-testid="submit-rating-btn"
				class="w-full rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus:ring-2 focus:ring-indigo-500 focus:outline-none disabled:bg-indigo-400"
			>
				{status === 'submitting' ? 'Submitting...' : 'Submit Rating'}
			</button>
		</form>
	{/if}
</div>
