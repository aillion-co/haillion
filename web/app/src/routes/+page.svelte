<script lang="ts">
	import { resolve } from '$app/paths';

	function resolvePath(path: string): string {
		return resolve(path as Parameters<typeof resolve>[0]);
	}
</script>

<div class="space-y-8">
	<!-- Hero Section -->
	<div
		class="rounded-2xl bg-gradient-to-r from-indigo-600 to-violet-600 p-8 text-white shadow-lg sm:p-12"
	>
		<div class="max-w-2xl space-y-4">
			<h1 class="text-3xl font-extrabold tracking-tight sm:text-4xl">
				Welcome to Aillion Ride Sharing
			</h1>
			<p class="text-lg text-indigo-100">
				A next-generation, high-performance headless ride-sharing platform. Choose Rider to book a
				trip or Driver to manage your active duties.
			</p>
			<div class="flex flex-wrap gap-4 pt-4">
				<a
					href={resolvePath('/dashboard/rider')}
					class="inline-flex items-center justify-center rounded-xl bg-white px-5 py-3 text-base font-semibold text-indigo-600 shadow-sm transition-colors hover:bg-indigo-50"
				>
					Request a Trip
				</a>
				<a
					href={resolvePath('/dashboard/driver')}
					class="inline-flex items-center justify-center rounded-xl border border-white/30 bg-white/10 px-5 py-3 text-base font-semibold text-white shadow-sm transition-colors hover:bg-white/20"
				>
					Driver Dashboard
				</a>
			</div>
		</div>
	</div>

	<!-- Platform Features -->
	<div class="grid gap-6 md:grid-cols-3">
		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<div
				class="flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-50 text-indigo-600"
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					class="h-6 w-6"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
			</div>
			<h3 class="mt-4 text-lg font-semibold text-gray-900">Real-time Matching</h3>
			<p class="mt-2 text-sm text-gray-500">
				Instant matching between riders and nearest available drivers powered by optimized matching
				algorithms.
			</p>
		</div>

		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<div
				class="flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-50 text-indigo-600"
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					class="h-6 w-6"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z"
					/>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M15 11a3 3 0 11-6 0 3 3 0 016 0z"
					/>
				</svg>
			</div>
			<h3 class="mt-4 text-lg font-semibold text-gray-900">Live GPS Tracking</h3>
			<p class="mt-2 text-sm text-gray-500">
				Live tracking of active trips and driver location telemetry, ensuring precise pickups and
				safety.
			</p>
		</div>

		<div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
			<div
				class="flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-50 text-indigo-600"
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					class="h-6 w-6"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
					/>
				</svg>
			</div>
			<h3 class="mt-4 text-lg font-semibold text-gray-900">Secure Payments</h3>
			<p class="mt-2 text-sm text-gray-500">
				Seamless post-trip payments and standard rating mechanism to build driver-rider trust.
			</p>
		</div>
	</div>
</div>
