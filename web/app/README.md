# Haillion Web App

The user-facing web application for the Haillion private-hire platform. It is a
[SvelteKit](https://svelte.dev/docs/kit) + TypeScript app styled with
[Tailwind CSS](https://tailwindcss.com/), built and run with
[Bun](https://bun.com/) (never npm/yarn/pnpm). In production it is served behind
the API gateway at `/`, talking to the backend services under `/api/*`.

## What's here

- `src/routes/` — pages: the landing page (`/`), `register`, and the
  `dashboard/rider` and `dashboard/driver` views.
- `src/lib/api/` — typed clients for the identity, matching, trip, billing, and
  review endpoints.
- `src/lib/components/` — UI building blocks (fare estimate, nearby list and
  map, payment flow, rating form, trip status, driver controls).
- `src/lib/stores/` — Svelte stores, including auth state.

Live driver/rider maps are rendered with [Leaflet](https://leafletjs.com/), and
the app is packaged with `svelte-adapter-bun`.

## Developing

Install dependencies with Bun, then start the dev server:

```sh
bun install --frozen-lockfile
bun run dev

# or open the app in a new browser tab
bun run dev -- --open
```

The dev server runs at `http://localhost:5173`. The backend it calls is expected
on the gateway at `http://localhost:8080` — see the repository root `README.md`
for how to bring the platform up locally with Skaffold.

## Building

Create a production build and preview it:

```sh
bun run build
bun run preview
```

## Checks

```sh
bun run test    # unit and component tests (Vitest)
bun run check   # type-check with svelte-check
bun run lint    # Prettier + ESLint
bun run format  # apply Prettier formatting
```
