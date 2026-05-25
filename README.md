# Haillion

**Haillion** is an open-source, real-time private hire platform (similar to Uber). It manages riders and drivers, matches them using real-time geospatial queries, tracks trips through a state machine, and processes demand-based surge pricing and payments.

This repository contains the full headless Go microservices backend and SvelteKit frontend (planned), structured to be built and maintained by a team of AI agents safely, with humans staying in control.

## Architecture

Haillion is composed of four core Go microservices:
- **Identity Service (`services/identity`)**: Manages rider and driver profiles and authentication.
- **Matching Service (`services/matching`)**: Tracks live driver locations and calculates ETAs, matching riders to drivers using geospatial queries (PostGIS).
- **Trip Service (`services/trip`)**: Tracks trip states (Requested, Accepted, In-Progress, Completed) using a strict state machine.
- **Billing Service (`services/billing`)**: Processes simulated payments and applies demand-based surge pricing.

## The idea in one paragraph

Building software with AI works best when the work is broken into small,
well-defined tasks and each task is handed off cleanly. This repository wires up
two AI agents — a **Boss** and a **Coder** (plus an optional **Reviewer**) —
that talk to each other through **GitHub Issues**. The Boss turns a goal like
*"add rate limiting to the orders service"* into a set of small, checkable
tasks. The Coder picks up one task at a time, writes the code, proves it works,
and opens a pull request. The Reviewer checks that pull request against the
original task. GitHub is the shared to-do list, so nothing lives only inside an
agent's memory.

## Who does what

| Agent | Job | Writes code? |
|-------|-----|--------------|
| **Boss** | Breaks a goal into small GitHub issues with clear acceptance criteria. | No |
| **Coder** | Implements one Go/backend issue end-to-end and opens a pull request. | Yes |
| **Frontend** | Implements one UI issue end-to-end (SvelteKit + Bun) and opens a pull request. | Yes |
| **Reviewer** | Checks a pull request against its issue; approves or requests changes. | No |

Each agent works with a fresh, limited view of the repo so it never gets
overwhelmed — it reads a compact **repo map** first and only opens the specific
files it needs.

## Backend and frontend

The Go services are **headless** — they expose APIs (gRPC, HTTP+JSON) and serve
no HTML. User-facing apps live separately under `web/<app>/` and are built with
**SvelteKit + TypeScript**, using **Bun** as the package manager and runner
(never npm). The Coder owns the Go side; the Frontend agent owns `web/`. The two
only meet at the API boundary, with TypeScript types generated from the Go
contract rather than hand-written twice.

## What's in this repo

```
AGENTS.md                     # The rulebook every agent follows
.opencode/
├── agents/                    # One file per agent (boss, coder, frontend, reviewer)
├── command/                  # Shortcuts: /plan, /work, /frontend, /review, /map
└── maps/                     # An auto-generated "map" of the codebase
scripts/
└── gen-repo-map.sh           # Regenerates the map from the source code
.github/workflows/
└── security.yml              # Automated security checks on every pull request
```

- **`AGENTS.md`** — the shared rulebook: coding conventions, what tools to use,
  how to keep changes small, and how to prove work is correct.
- **`.opencode/agents/`** — the instructions and model assignment for each agent.
- **`.opencode/command/`** — slash-command shortcuts that start an agent on a
  task (`/plan`, `/work`, `/review`, `/map`).
- **`.opencode/maps/`** — a short, auto-generated summary of the codebase so
  agents can navigate without reading everything.
- **`scripts/gen-repo-map.sh`** — rebuilds that map deterministically; run it
  after merges (also wired to the `/map` command).
- **`.github/workflows/security.yml`** — the automated gate that re-runs build,
  vet, security, and vulnerability checks on every pull request.

## The guidelines it enforces

The rules in `AGENTS.md` exist to keep AI-written code small, correct, and
trustworthy. The main ones:

- **Small, traceable changes.** Every line of code must map back to a specific
  requirement in the issue. No "while I'm here" refactors, no speculative
  helpers, no config knobs nobody asked for.
- **Plain standard library first.** Prefer Go's built-in tools; only add an
  outside dependency when writing it ourselves would be genuinely harder, and
  always explain why.
- **Prove it, don't claim it.** Before a pull request opens, the Coder runs the
  build, tests, linter, and security scanners and pastes the real output as
  evidence. The same checks run automatically in CI as the authoritative gate.
- **Tests that mean something.** Every error path is tested; assertions check
  real values, not just "no error happened."
- **Ask, don't guess.** If a task is ambiguous or would grow too large, the
  agent stops and asks rather than improvising.
- **Stay in your lane.** Services can't reach into each other's private code;
  reviewers don't rewrite code; bosses don't write code.

## Online and godark modes

The project runs in one of two modes:

- **Online** (default) — agents use cloud models: **Gemini 3.1 Pro** for
  planning and review, **Gemini 3.5 Flash** for coding and frontend work.
- **Godark** — fully offline, for working with no network (on a plane, say).

Run `make godark` **while still online**: it installs the tools, warms the Go
and frontend caches, pulls the reference docs, snapshots the GitHub issues, and
then switches the agents to local **Gemma** models (boss/review on
`gemma-4-31B-it`, coding/frontend on `gemma-4-26B-A4B-it`) that run without a
network. After that, building, testing, linting, reading docs, and
reading/updating issues all work offline; check the mode and readiness with
`make godark-check`. Issue changes you make while dark are queued and replayed
with `make issues-sync` when you reconnect. Run `make online` to switch back to
the Gemini models and re-enable the network.

`make godark` rewrites the `model:` fields in `.opencode/agents/*.md` to the
local models — so while dark those files show as modified. Run `make online` to
restore the committed Gemini defaults before committing. The remaining
network-only steps — opening PRs, `govulncheck`'s vulnerability database, the Go
toolchain download, and adding new dependencies — need a connection; agents are
told to plan around them in `AGENTS.md`.

## How to use it

1. Ensure your AI agent environment is configured correctly.
2. The initial architecture has been seeded into GitHub Issues via the Boss.
3. Run `make tools` to install the toolchain (online/Gemini is the default;
   run `make godark` instead when you need to go offline-ready).
4. Let the Coder work through the queued issues (`/work`).
5. Review and merge the pull requests (`/review`).

The detailed, exact instructions for each agent live in the files above — this
README is just the overview.

## Running the Platform Locally

Haillion is designed to be easily runnable locally using Docker, Kubernetes (via KIND), and Skaffold.

1. Install prerequisites: [Docker](https://docs.docker.com/get-docker/), [KIND](https://kind.sigs.k8s.io/docs/user/quick-start/), [kubectl](https://kubernetes.io/docs/tasks/tools/), and [Skaffold](https://skaffold.dev/docs/install/).
2. Start the local KIND cluster:
   ```bash
   kind create cluster --name agent-platform
   ```
3. Run Skaffold to build the images and deploy them to the cluster:
   ```bash
   skaffold run
   ```
4. The API Gateway will be automatically port-forwarded to `http://localhost:8080`.

### Running E2E Tests

The repository contains a full end-to-end integration test suite simulating a complete user journey (riders, drivers, matching, trips, billing, and reviews).

Once the platform is running locally via `skaffold run`:

1. Ensure the gateway is port-forwarded:
   ```bash
   kubectl port-forward svc/gateway -n haillion 8080:8080 > /dev/null 2>&1 &
   ```
2. Run the tests:
   ```bash
   go test -v ./test/e2e/...
   ```

### Running the Frontend UI Locally

The user-facing web application is built with SvelteKit and Bun, located under `web/app/`.

To test and develop the UI locally against your running cluster:

1. Ensure the backend platform is running and the gateway is port-forwarded:
   ```bash
   skaffold run
   kubectl port-forward svc/gateway -n haillion 8080:8080 > /dev/null 2>&1 &
   ```
2. Navigate to the frontend directory:
   ```bash
   cd web/app
   ```
3. Install the dependencies using Bun (do not use npm/yarn/pnpm):
   ```bash
   bun install --frozen-lockfile
   ```
4. Start the SvelteKit development server:
   ```bash
   bun run dev
   ```
5. Open your browser and navigate to `http://localhost:5173` to interact with the Haillion platform (register as a rider/driver, request trips, etc.).

**Frontend Testing & Checks:**
You can run the frontend verification checks from within the `web/app` directory:
- Run unit and component tests: `bun run test`
- Type-check the project: `bun run check`
- Lint the code: `bun run lint`
