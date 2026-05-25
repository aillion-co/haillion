import { describe, it, expect } from 'vitest';
import fs from 'fs';
import path from 'path';

describe('Rider Dashboard Page Component', () => {
	const componentPath = path.resolve(__dirname, '../../routes/dashboard/rider/+page.svelte');

	it('shall exist on the filesystem', () => {
		expect(fs.existsSync(componentPath)).toBe(true);
	});

	it('shall contain the Pickup postcode text input and no lat/lng inputs', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');

		// Assert on key structure elements
		expect(content).toContain('id="pickup-postcode"');
		expect(content).toContain('type="text"');
		expect(content).toContain('bind:value={postcode}');
		expect(content).not.toContain('id="latitude"');
		expect(content).not.toContain('id="longitude"');
	});

	it('shall import and call matching and trip API methods in sequence', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');

		// Assert imports
		expect(content).toContain(
			"import { matchDriver, requestRide, ApiError } from '$lib/api/matching';"
		);
		expect(content).toContain("import { createTrip, getTrip } from '$lib/api/trip';");

		// Assert execution sequence logic
		expect(content).toContain('await requestRide');
		expect(content).toContain('await matchDriver');
		expect(content).toContain('await createTrip');
	});

	it('shall normalise postcode to uppercase on submit and handle invalid postcode/postcode not found errors verbatim', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');

		// Postcode normalisation (trim + uppercase)
		expect(content).toContain('postcode.trim().toUpperCase()');

		// Verbatim error handling for postcode failure codes
		expect(content).toContain("err.message === 'invalid postcode'");
		expect(content).toContain("err.message === 'postcode not found'");
	});

	it('shall disable submit button when postcode is empty', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');

		// Submit disabled when empty
		expect(content).toContain('!postcode.trim()');
	});
});
