import { describe, it, expect } from 'vitest';
import fs from 'fs';
import path from 'path';

describe('NearbyMap Component', () => {
	const componentPath = path.resolve(__dirname, './NearbyMap.svelte');

	it('shall exist on the filesystem', () => {
		expect(fs.existsSync(componentPath)).toBe(true);
	});

	it('shall declare the required MapMarker and Props interfaces with proper Svelte 5 runes', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');

		// Interface contracts
		expect(content).toContain('interface MapMarker');
		expect(content).toContain('id: string;');
		expect(content).toContain('lat: number;');
		expect(content).toContain('lng: number;');

		expect(content).toContain('interface Props');
		expect(content).toContain('center: { lat: number; lng: number };');
		expect(content).toContain('you: MapMarker;');
		expect(content).toContain('others: MapMarker[];');
		expect(content).toContain('zoom?: number;');

		// Svelte 5 Props rune
		expect(content).toContain('let { center, you, others, zoom = 13 }: Props = $props();');
	});

	it('shall be SSR-safe: dynamic import of leaflet inside onMount', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain('onMount(');
		expect(content).toContain("await import('leaflet')");
	});

	it('shall initialize the map and layers exactly once in onMount', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');

		// Constructs Map
		expect(content).toContain('L.map(container)');

		// Adds tile layer with OSM URL and attribution
		expect(content).toContain('L.tileLayer(');
		expect(content).toContain('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png');
		expect(content).toContain('OpenStreetMap');

		// Creates and positions the 'you' marker with distinct styling
		expect(content).toContain('L.marker([you.lat, you.lng]');
		expect(content).toContain('L.divIcon(');
		expect(content).toContain('youMarker =');
	});

	it('shall clear and repopulate the layer group when others updates (no map recreation)', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');

		// Setup the layer group
		expect(content).toContain('othersLayerGroup = L.layerGroup()');

		// Clear and repopulate the others layer group dynamically using $effect
		expect(content).toContain('othersLayerGroup.clearLayers()');
		expect(content).toContain('others.forEach');
		expect(content).toContain('marker.addTo(othersLayerGroup');
	});

	it('shall apply the Leaflet default-icon path fix-up snippet once', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain('delete (L.Icon.Default.prototype as unknown as { _getIconUrl?: unknown })._getIconUrl');
		expect(content).toContain('L.Icon.Default.mergeOptions');
	});

	it('shall import leaflet css file within the component file, not globally', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain("import 'leaflet/dist/leaflet.css';");
	});

	it('shall render a div container with fixed height of 320px via inline style', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');
		expect(content).toContain('bind:this={container}');
		expect(content).toContain('style="height: 320px;"');
	});
});
