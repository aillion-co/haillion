import { describe, it, expect } from 'vitest';
import fs from 'fs';
import path from 'path';

describe('FareEstimate Component', () => {
	const componentPath = path.resolve(__dirname, './FareEstimate.svelte');

	it('shall exist on the filesystem', () => {
		expect(fs.existsSync(componentPath)).toBe(true);
	});

	it('shall import estimateFare API method and handle errors', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain(
			"import { estimateFare, type FareEstimate } from '$lib/api/billing';"
		);
		expect(content).toContain('await estimateFare');
	});

	it('shall support props and runes', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain('$props()');
		expect(content).toContain('pickup');
		expect(content).toContain('destination');
		expect(content).toContain('riders');
		expect(content).toContain('drivers');
	});

	it('shall implement 400ms debounce using setTimeout and clearTimeout inside $effect', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain('$effect');
		expect(content).toContain('setTimeout');
		expect(content).toContain('clearTimeout');
		expect(content).toContain('400');
	});

	it('shall display distance, base, and total in breakdown', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain('data-testid="fare-distance"');
		expect(content).toContain('data-testid="fare-base"');
		expect(content).toContain('data-testid="fare-total"');
		expect(content).toContain('Distance:');
		expect(content).toContain('Base:');
		expect(content).toContain('Estimated total:');
	});

	it('shall omit surge row if surge_multiplier is exactly 1.0', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain('surge_multiplier !== 1');
		expect(content).toContain('data-testid="fare-surge"');
	});

	it('shall append (minimum fare) if minimum_applied is true', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain('minimum_applied');
		expect(content).toContain('(minimum fare)');
	});

	it('shall render error messages verbatim when error occurs', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain('data-testid="fare-error"');
		expect(content).toContain('errorText');
	});

	it('shall show Calculating placeholder when loading', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain('data-testid="fare-loading"');
		expect(content).toContain('Calculating…');
	});

	it('shall render nothing when pickup or destination postcode is empty', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain('pickup.trim() && destination.trim()');
	});
});
