import { describe, it, expect } from 'vitest';
import fs from 'fs';
import path from 'path';

describe('RatingForm Component', () => {
	const componentPath = path.resolve(__dirname, './RatingForm.svelte');

	it('shall exist on the filesystem', () => {
		expect(fs.existsSync(componentPath)).toBe(true);
	});

	it('shall contain required elements and test ids', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');

		// Assert on key structure elements
		expect(content).toContain('data-testid="rating-form-container"');
		expect(content).toContain('data-testid="rating-success-message"');
		expect(content).toContain('data-testid="star-btn-');
		expect(content).toContain('data-testid="comment-textarea"');
		expect(content).toContain('data-testid="rating-error"');
		expect(content).toContain('data-testid="submit-rating-btn"');
	});

	it('shall import and call API methods', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');

		// Assert API calls
		expect(content).toContain("import { createReview } from '$lib/api/review';");
		expect(content).toContain('await createReview');
	});

	it('shall support props and runes', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');

		// Assert Svelte 5 runes and types
		expect(content).toContain(
			'let { tripId, reviewerId, revieweeId, onSuccess }: Props = $props();'
		);
		expect(content).toContain('$state');
	});
});
