<script lang="ts">
	import { createPayment, processPayment } from '$lib/api/billing';

	interface Props {
		tripId: string;
		amount?: number;
		onPaymentSuccess: () => void;
	}

	let { tripId, amount = 1500, onPaymentSuccess }: Props = $props();

	let status = $state<'idle' | 'creating' | 'processing' | 'success' | 'failed'>('idle');
	let errorMessage = $state<string | null>(null);
	let paymentId = $state<string | null>(null);

	async function handlePayment() {
		status = 'creating';
		errorMessage = null;

		try {
			if (!paymentId) {
				const res = await createPayment({ trip_id: tripId, amount });
				paymentId = res.id;
			}
			status = 'processing';
			await processPayment(paymentId);
			status = 'success';
			onPaymentSuccess();
		} catch (err) {
			status = 'failed';
			errorMessage = err instanceof Error ? err.message : 'Payment processing failed';
		}
	}
</script>

<div class="rounded-2xl border border-gray-200 bg-white p-6 shadow-md" data-testid="payment-flow">
	<h2 class="text-xl font-bold text-gray-900">Trip Completed</h2>
	<p class="mt-1 text-sm text-gray-500">Please process your payment for this trip.</p>

	<div class="mt-6 space-y-4">
		<div class="flex items-center justify-between border-b border-gray-100 pb-4">
			<span class="text-sm font-medium text-gray-500">Amount Due</span>
			<span class="text-lg font-bold text-gray-900" data-testid="payment-amount">
				${(amount / 100).toFixed(2)}
			</span>
		</div>

		{#if errorMessage}
			<div
				class="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-800"
				data-testid="payment-error"
			>
				<p>{errorMessage}</p>
			</div>
		{/if}

		<div class="flex flex-col items-center justify-center">
			{#if status === 'idle' || status === 'failed'}
				<button
					onclick={handlePayment}
					data-testid="pay-btn"
					class="w-full rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500"
				>
					{status === 'failed' ? 'Retry Payment' : 'Pay Now'}
				</button>
			{:else if status === 'creating'}
				<p class="text-sm font-medium text-gray-600" data-testid="payment-status">
					Creating payment...
				</p>
			{:else if status === 'processing'}
				<p class="text-sm font-medium text-gray-600" data-testid="payment-status">
					Processing payment...
				</p>
			{:else if status === 'success'}
				<p class="text-sm font-semibold text-green-600" data-testid="payment-status">
					Payment processed successfully!
				</p>
			{/if}
		</div>
	</div>
</div>
