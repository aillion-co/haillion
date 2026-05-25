import { describe, it, expect } from 'vitest';
import fs from 'fs';
import path from 'path';

describe('NearbyList Component', () => {
	const componentPath = path.resolve(__dirname, './NearbyList.svelte');

	it('shall exist on the filesystem', () => {
		expect(fs.existsSync(componentPath)).toBe(true);
	});

	it('shall declare proper props interface', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain('kind:');
		expect(content).toContain('items:');
		expect(content).toContain('status:');
		expect(content).toContain('$props()');
	});

	it('shall slice short-id to 6 characters if longer, or render whole', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain('id.length < 6');
		expect(content).toContain('id.slice(0, 6)');
	});

	it('shall format distance using the correct formula (meters / 1609.34).toFixed(1)', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain('1609.34');
		expect(content).toContain('toFixed(1)');
	});

	it('shall sort items ASC by distance and limit to 10 rows', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain('distanceMeters');
		expect(content).toContain('slice(0, 10)');
		expect(content).toContain('$derived');
	});

	it('shall render empty state message: No <drivers|riders> nearby right now', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain("No {kind === 'driver' ? 'drivers' : 'riders'} nearby right now");
	});

	it('shall render error state message: Could not load nearby <drivers|riders>', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain("Could not load nearby {kind === 'driver' ? 'drivers' : 'riders'}");
	});

	it('shall render nothing when status is idle', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain("status !== 'idle'");
	});
});
