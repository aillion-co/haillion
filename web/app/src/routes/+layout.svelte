<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { resolve } from '$app/paths';

	let { children } = $props();

	function resolvePath(path: string): string {
		return resolve(path as Parameters<typeof resolve>[0]);
	}
</script>

<svelte:head>
	<title>Aillion - Ride Sharing</title>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="flex min-h-screen flex-col bg-gray-50 font-sans text-gray-900">
	<!-- Navigation Header -->
	<header class="sticky top-0 z-50 border-b border-gray-200 bg-white shadow-sm">
		<div class="mx-auto flex max-w-7xl items-center justify-between px-4 py-4 sm:px-6 lg:px-8">
			<div class="flex items-center space-x-8">
				<!-- Logo -->
				<a
					href={resolvePath('/')}
					class="flex items-center space-x-2 text-xl font-bold text-indigo-600 transition-colors hover:text-indigo-500"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-6 w-6"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
					</svg>
					<span>Aillion</span>
				</a>

				<!-- Nav Links -->
				<nav class="hidden items-center space-x-6 md:flex">
					<a
						href={resolvePath('/')}
						class="text-sm font-medium text-gray-700 transition-colors hover:text-indigo-600"
						>Dashboard</a
					>
					<a
						href={resolvePath('/rider')}
						class="text-sm font-medium text-gray-500 transition-colors hover:text-indigo-600"
						>Rider</a
					>
					<a
						href={resolvePath('/driver')}
						class="text-sm font-medium text-gray-500 transition-colors hover:text-indigo-600"
						>Driver</a
					>
					<a
						href={resolvePath('/payments')}
						class="text-sm font-medium text-gray-500 transition-colors hover:text-indigo-600"
						>Payments</a
					>
				</nav>
			</div>

			<!-- Right Side -->
			<div class="flex items-center space-x-4">
				<span
					class="inline-flex items-center rounded-full bg-green-50 px-2.5 py-0.5 text-xs font-medium text-green-700 ring-1 ring-green-600/20 ring-inset"
				>
					<span class="mr-1.5 h-1.5 w-1.5 rounded-full bg-green-500"></span>
					Gateway Online
				</span>
				<div class="text-sm font-medium text-gray-700">Guest</div>
			</div>
		</div>
	</header>

	<!-- Main Content Area -->
	<main class="flex-grow">
		<div class="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
			{@render children()}
		</div>
	</main>

	<!-- Footer -->
	<footer class="border-t border-gray-200 bg-white py-6">
		<div class="mx-auto max-w-7xl px-4 text-center text-sm text-gray-500 sm:px-6 lg:px-8">
			&copy; 2026 Aillion Inc. All rights reserved.
		</div>
	</footer>
</div>
