<script lang="ts">
	import { onMount } from 'svelte';
	import type * as Leaflet from 'leaflet';
	import 'leaflet/dist/leaflet.css';

	interface MapMarker {
		id: string;
		lat: number;
		lng: number;
		label?: string;
	}

	interface Props {
		center: { lat: number; lng: number };
		you: MapMarker;
		others: MapMarker[];
		zoom?: number;
	}

	let { center, you, others, zoom = 13 }: Props = $props();

	let container = $state<HTMLDivElement>();
	let L = $state<typeof Leaflet>();
	let map: Leaflet.Map | undefined;
	let youMarker: Leaflet.Marker | undefined;
	let othersLayerGroup: Leaflet.LayerGroup | undefined;

	onMount(() => {
		let mapInstance: Leaflet.Map | undefined;

		async function initMap() {
			const leafletModule = await import('leaflet');
			L = (leafletModule.default || leafletModule) as unknown as typeof Leaflet;

			if (!container) return;

			// Apply Leaflet default-icon path fix-up snippet once
			delete (L.Icon.Default.prototype as unknown as { _getIconUrl?: unknown })._getIconUrl;
			L.Icon.Default.mergeOptions({
				iconRetinaUrl:
					'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-icon-2x.png',
				iconUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-icon.png',
				shadowUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-shadow.png'
			});

			// Create Map
			mapInstance = L.map(container).setView([center.lat, center.lng], zoom);
			map = mapInstance;

			// Add Tile Layer
			L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
				attribution:
					'&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
			}).addTo(mapInstance);

			// Custom "you" icon - styled distinctly
			const youIcon = L.divIcon({
				html: `<div style="background-color: #4f46e5; width: 16px; height: 16px; border-radius: 50%; border: 2px solid white; box-shadow: 0 2px 4px rgba(0,0,0,0.3);"></div>`,
				className: 'you-marker-icon',
				iconSize: [16, 16],
				iconAnchor: [8, 8]
			});

			// Add "you" marker
			youMarker = L.marker([you.lat, you.lng], { icon: youIcon });
			if (you.label) {
				youMarker.bindPopup(you.label);
			}
			youMarker.addTo(mapInstance);

			// Create layer group for others and add to map
			othersLayerGroup = L.layerGroup().addTo(mapInstance);

			// Populate others initially
			others.forEach((item) => {
				if (L && othersLayerGroup) {
					const marker = L.marker([item.lat, item.lng]);
					if (item.label) {
						marker.bindPopup(item.label);
					}
					marker.addTo(othersLayerGroup);
				}
			});
		}

		initMap();

		return () => {
			if (mapInstance) {
				mapInstance.remove();
			}
		};
	});

	// Reactive update for "others" layer group when props change
	$effect(() => {
		if (map && othersLayerGroup && L) {
			othersLayerGroup.clearLayers();
			others.forEach((item) => {
				if (L) {
					const marker = L.marker([item.lat, item.lng]);
					if (item.label) {
						marker.bindPopup(item.label);
					}
					marker.addTo(othersLayerGroup!);
				}
			});
		}
	});

	// Reactive update for "you" marker
	$effect(() => {
		if (youMarker && L) {
			youMarker.setLatLng([you.lat, you.lng]);
			if (you.label) {
				youMarker.bindPopup(you.label);
			}
		}
	});

	// Reactive update for map center
	$effect(() => {
		if (map) {
			map.setView([center.lat, center.lng], zoom);
		}
	});
</script>

<div
	bind:this={container}
	style="height: 320px;"
	class="w-full rounded-2xl border border-gray-200 bg-gray-50 shadow-md"
>
	<div class="hidden" aria-hidden="true">
		{you.id}
		{#each others as item (item.id)}
			{item.id}
		{/each}
	</div>
</div>
