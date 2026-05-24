import { describe, it, expect } from 'vitest';
import fs from 'fs';
import path from 'path';

describe('PaymentFlow Component', () => {
	const componentPath = path.resolve(__dirname, './PaymentFlow.svelte');

	it('shall exist on the filesystem', () => {
		expect(fs.existsSync(componentPath)).toBe(true);
	});

	it('shall contain required elements and test ids', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');

		// Assert on key structure elements
		expect(content).toContain('data-testid="payment-flow"');
		expect(content).toContain('data-testid="payment-amount"');
		expect(content).toContain('data-testid="payment-error"');
		expect(content).toContain('data-testid="pay-btn"');
		expect(content).toContain('data-testid="payment-status"');
	});

	it('shall import and call API methods', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');

		// Assert API calls
		expect(content).toContain("import { createPayment, processPayment } from '$lib/api/billing';");
		expect(content).toContain('await createPayment');
		expect(content).toContain('await processPayment');
	});

	it('shall support props and runes', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');

		// Assert Svelte 5 runes and types
		expect(content).toContain('let { tripId, amount = 1500, onPaymentSuccess }: Props = $props();');
		expect(content).toContain('$state');
	});
});
