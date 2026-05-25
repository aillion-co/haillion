import { render, screen } from '@testing-library/svelte';
import Page from './+page.svelte';
import { describe, it, expect, vi } from 'vitest';

vi.mock('$app/paths', () => ({
	resolve: (path: string) => path
}));

describe('+page.svelte', () => {
	it('renders home page nav links correctly', () => {
		render(Page);

		const riderLink = screen.getByRole('link', { name: /request a trip/i });
		const driverLink = screen.getByRole('link', { name: /driver dashboard/i });

		expect(riderLink.getAttribute('href')).toBe('/dashboard/rider');
		expect(driverLink.getAttribute('href')).toBe('/dashboard/driver');
	});
});
