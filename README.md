# Multi-Agent Template for Go Microservice Monorepos

This is a **template repository**. It doesn't contain a working application — it
contains the *setup* that lets a team of AI agents build and review a Go
microservices codebase safely, with humans staying in control.

## The idea in one paragraph

Building software with AI works best when the work is broken into small,
well-defined tasks and each task is handed off cleanly. This template wires up
two AI agents — a **Planner** and a **Coder** (plus an optional **Reviewer**) —
that talk to each other through **GitHub Issues**. The Planner turns a goal like
*"add rate limiting to the orders service"* into a set of small, checkable
tasks. The Coder picks up one task at a time, writes the code, proves it works,
and opens a pull request. The Reviewer checks that pull request against the
original task. GitHub is the shared to-do list, so nothing lives only inside an
agent's memory.

## Who does what

| Agent | Job | Writes code? |
|-------|-----|--------------|
| **Planner** | Breaks a goal into small GitHub issues with clear acceptance criteria. | No |
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
├── agent/                    # One file per agent (planner, coder, frontend, reviewer)
├── command/                  # Shortcuts: /plan, /work, /frontend, /review, /map
└── maps/                     # An auto-generated "map" of the codebase
scripts/
└── gen-repo-map.sh           # Regenerates the map from the source code
.github/workflows/
└── security.yml              # Automated security checks on every pull request
```

- **`AGENTS.md`** — the shared rulebook: coding conventions, what tools to use,
  how to keep changes small, and how to prove work is correct.
- **`.opencode/agent/`** — the instructions and model assignment for each agent.
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
  reviewers don't rewrite code; planners don't write code.

## Works offline

The repo is built so you can develop with no network — on a plane, say. Run
`make bootstrap` once **while online**: it installs the tools, warms the Go and
frontend caches, pulls the reference docs, and snapshots the GitHub issues into
a local cache. After that, building, testing, linting, reading docs, and
reading/updating issues all work offline; check readiness with
`make offline-check`. Issue changes you make offline are queued and replayed
with `make issues-sync` when you reconnect. The remaining network-only steps —
opening PRs, `govulncheck`'s vulnerability database, and adding new
dependencies — are documented in `AGENTS.md`.

## How to use it

1. Copy this template into a new repository.
2. Fill in your project name in `AGENTS.md`.
3. Run `make bootstrap` (while online) to install tools and prime the caches.
4. Give the Planner a goal (`/plan add rate limiting to the orders service`).
5. Let the Coder work through the resulting issues (`/work`).
6. Review and merge the pull requests (`/review`).

The detailed, exact instructions for each agent live in the files above — this
README is just the overview.
