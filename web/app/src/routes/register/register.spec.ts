import { describe, it, expect } from 'vitest';
import fs from 'fs';
import path from 'path';

describe('Registration Page Component (Code Analysis)', () => {
	const componentPath = path.resolve(__dirname, './+page.svelte');

	it('shall exist on the filesystem', () => {
		expect(fs.existsSync(componentPath)).toBe(true);
	});

	it('shall declare onMount and mounted state rune', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');

		// Assert onMount import
		expect(content).toContain("import { onMount } from 'svelte';");

		// Assert mounted state rune
		expect(content).toContain('let mounted = $state(false);');

		// Assert onMount implementation sets mounted to true
		expect(content).toMatch(/onMount\(\s*\(\)\s*=>\s*\{\s*mounted\s*=\s*true;?\s*\}\s*\)/);
	});

	it('shall prevent submission via GET by using method="post" on form', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');

		// Assert form has method="post"
		expect(content).toContain('method="post"');
		expect(content).toMatch(/<form[^>]*onsubmit=\{handleSubmit\}[^>]*method="post"/);
	});

	it('shall disable submit button before onMount has run and enable after', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');

		// Assert button disabled attribute is tied to both mounted and loading
		expect(content).toContain('disabled={!mounted || loading}');
	});

	it('shall prevent default form submission in handleSubmit synchronously as the first statement', () => {
		const content = fs.readFileSync(componentPath, 'utf-8');

		// Assert handleSubmit is defined and calls e.preventDefault() first
		expect(content).toMatch(
			/async\s+function\s+handleSubmit\(\s*e:\s*SubmitEvent\s*\)\s*\{\s*e\.preventDefault\(\);/
		);
	});
});
