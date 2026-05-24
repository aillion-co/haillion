<script lang="ts">
	import { authStore } from '$lib/stores/auth';
	import DriverControls from '$lib/components/DriverControls.svelte';
</script>

<div class="mx-auto max-w-2xl space-y-6 px-4 py-8">
	<div class="flex items-center justify-between border-b border-gray-200 pb-4">
		<div>
			<h1 class="font-sans text-3xl font-extrabold tracking-tight text-gray-900">
				Driver Dashboard
			</h1>
			<p class="mt-1 font-sans text-sm text-gray-500">
				Toggle online to receive trips and manage your status.
			</p>
		</div>
		<a
			href="/"
			class="inline-flex items-center rounded-xl border border-gray-300 bg-white px-4 py-2 font-sans text-sm font-semibold text-gray-700 shadow-sm hover:bg-gray-50"
		>
			Back to Home
		</a>
	</div>

	{#if !$authStore}
		<div class="rounded-2xl border border-yellow-200 bg-yellow-50 p-6 text-center shadow-sm">
			<svg
				class="mx-auto h-12 w-12 text-yellow-500"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
				/>
			</svg>
			<h2 class="mt-4 font-sans text-lg font-bold text-yellow-800">Authentication Required</h2>
			<p class="mt-2 font-sans text-sm text-yellow-700">
				Please register or log in first to use the Driver dashboard.
			</p>
			<div class="mt-4">
				<a
					href="/register"
					class="inline-flex items-center rounded-xl bg-yellow-600 px-4 py-2 font-sans text-sm font-semibold text-white shadow-sm hover:bg-yellow-500"
				>
					Register / Login
				</a>
			</div>
		</div>
	{:else if $authStore.role !== 'driver'}
		<div class="rounded-2xl border border-red-200 bg-red-50 p-6 text-center shadow-sm">
			<svg
				class="mx-auto h-12 w-12 text-red-500"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
				/>
			</svg>
			<h2 class="mt-4 font-sans text-lg font-bold text-red-800">Access Denied</h2>
			<p class="mt-2 font-sans text-sm text-red-700">
				This dashboard is only accessible to drivers. Your current role is "{$authStore.role}".
			</p>
			<div class="mt-4">
				<a
					href="/dashboard/rider"
					class="inline-flex items-center rounded-xl bg-red-600 px-4 py-2 font-sans text-sm font-semibold text-white shadow-sm hover:bg-red-500"
				>
					Go to Rider Dashboard
				</a>
			</div>
		</div>
	{:else}
		<DriverControls driverId={$authStore.id} />
	{/if}
</div>
