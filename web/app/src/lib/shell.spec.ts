import { describe, it, expect } from 'vitest';
import fs from 'fs';
import path from 'path';

describe('Aillion Web App Shell', () => {
	it('shall have a basic navigation header with key routes', () => {
		const layoutPath = path.resolve(__dirname, '../routes/+layout.svelte');
		expect(fs.existsSync(layoutPath)).toBe(true);

		const layoutContent = fs.readFileSync(layoutPath, 'utf-8');

		// Assert navigation logo brand
		expect(layoutContent).toContain('Aillion');

		// Assert navigation links
		expect(layoutContent).toContain("href={resolvePath('/')}");
		expect(layoutContent).toContain("href={resolvePath('/rider')}");
		expect(layoutContent).toContain("href={resolvePath('/driver')}");
		expect(layoutContent).toContain("href={resolvePath('/payments')}");
	});

	it('shall produce a bun.lock and no npm/yarn/pnpm lockfiles', () => {
		const appDir = path.resolve(__dirname, '../../');

		// Check that bun.lock exists
		const bunLockPath = path.join(appDir, 'bun.lock');
		expect(fs.existsSync(bunLockPath)).toBe(true);

		// Check that other lockfiles do not exist
		const forbiddenLockfiles = ['package-lock.json', 'yarn.lock', 'pnpm-lock.yaml'];
		for (const lockfile of forbiddenLockfiles) {
			const lockfilePath = path.join(appDir, lockfile);
			expect(fs.existsSync(lockfilePath)).toBe(false);
		}
	});
});
