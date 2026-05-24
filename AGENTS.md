# Project: <name>

Go microservices monorepo. Services live under `services/<name>/`, shared libs under `pkg/`, internal-only code under `internal/`. One go.mod at the root (workspace mode).

The Go services are **headless**: they expose gRPC and HTTP+JSON APIs and render no HTML. User-facing UIs live under `web/<app>/` and are built with a separate toolchain (Bun + SvelteKit) by the frontend agent. See the **Frontend** section below and `.opencode/agent/frontend.md`. The rules in this file are Go-specific unless a section says otherwise.

## Tooling
- Format: `gofmt -s -w` and `goimports -w` before any commit.
- Lint: `golangci-lint run ./...` must pass.
- Test: `go test ./<changed-package>/...` first; `go test ./...` before PR.
- LSP: gopls is available. **Use it for symbol lookup instead of grep where possible** (`workspace/symbol`, `textDocument/definition`, `textDocument/references`).
- Search: `rg` (ripgrep) for content, `fd` for paths. Never `find` or `grep -r`.
- GitHub: `gh` CLI is authenticated. Use `gh issue` and `gh pr` for all repo interactions.

## Context discipline (read this every session)

This repo is too large to fit in any model's context. Follow these rules without exception:

1. **Start with the repo map**, not source. Read `.opencode/maps/repo-map.md` first. It is the navigation aid.
2. **Pull files on demand.** Never `cat` a directory. Use `gopls workspace/symbol` or `rg -l` to locate, then read only the spans you need.
3. **Prefer symbol lookups over text search** for anything Go-specific (types, functions, interfaces). gopls respects scope and types; ripgrep doesn't.
4. **Externalise state.** Decisions, partial findings, and TODOs go in the GitHub issue comments — not your context. Re-read the issue at the start of each turn.
5. **Bounded sub-tasks.** If a task touches more than ~5 files or ~3 packages, stop and ask the planner to split it.
6. **Fresh context per issue.** Do not carry state between issues in your head. The issue body and its comments are the source of truth.

## Implementation discipline (applies to any agent that writes code)

Anti-patterns that produce bad PRs even when scope and tests are correct.

- **Surface assumptions before acting.** If a task has multiple plausible interpretations, post a comment listing them and stop — don't pick silently. One extra round-trip costs less than redoing the work.
- **Minimum code that satisfies the acceptance criteria.** No config flags, no interfaces with one implementation, no error paths for impossible states, no "might be useful later" helpers. If a 200-line solution could be 50, rewrite it.
- **Surgical edits only.** Touch only what the issue requires. Do not "improve" adjacent code, comments, or formatting. Do not refactor unrelated code that you happen to notice. If you spot unrelated dead code or a real bug, post a comment on the issue — do not fix it in this PR.
- **Match existing structure even when you'd write it differently.** gofmt and golangci-lint handle surface style; you handle structural mimicry — the same patterns for error returns, the same shape for table-driven tests, the same naming for receivers.
- **Orphans you create, you clean up.** If your changes leave an import, variable, or function unused, remove it. If something was already unused before you touched the file, leave it and mention it.
- **Every changed line traces to an acceptance criterion.** If you can't justify a line by pointing to a checkbox in the issue, delete it.

## Code conventions
- Errors: wrap with `fmt.Errorf("...: %w", err)`. No `errors.New` for wrapped errors.
- Logging: `slog` only. No `log` package, no `fmt.Println` in non-main packages.
- Tests: table-driven, `t.Run(name, ...)`. Use `testify/require` for fatal asserts, `testify/assert` for non-fatal.
- Context: every public function that does I/O takes `ctx context.Context` as first arg.
- Interfaces: declared at the consumer, not the producer. Keep them small.
- No global state. No `init()` for anything other than `flag` or `prometheus` registration.

## Dependencies

Prefer the Go standard library. Add a third-party dependency only when the alternative is significant in-repo complexity. The bar is: "implementing this correctly ourselves is materially harder than vetting and updating someone else's code." If you're not sure, default to stdlib.

**Third-party is the right call for:**
- Database drivers (`pgx`, `database/sql` drivers — you are not writing a Postgres wire protocol).
- gRPC and protobuf (`google.golang.org/grpc`, `google.golang.org/protobuf`).
- Cloud provider SDKs (GCP, AWS — no stdlib equivalent).
- Cryptography not in `crypto/*` (e.g. `golang.org/x/crypto/...`).
- Observability SDKs (OpenTelemetry).
- Migration tooling (`golang-migrate`, `goose`).

**Use stdlib instead of:**
- HTTP routing — `net/http` with `ServeMux` (Go 1.22+) handles path params and method matching. No `chi`, `gin`, `echo` unless the service genuinely needs middleware ecosystems we'd otherwise reimplement.
- Configuration — `flag` and `os.Getenv` cover most services. No `viper`.
- Logging — `log/slog` only (already in Code conventions). No `zap`, `zerolog`, `logrus`.
- Errors — `fmt.Errorf` with `%w` plus `errors.Is/As`. No `pkg/errors`.
- JSON — `encoding/json` unless a benchmark in this repo proves it's the bottleneck.
- Validation — write a function. No `go-playground/validator` unless validation is genuinely declarative across many types.
- Assertions in general code — no `samber/lo`, no `funk`. Write the loop.

**Grandfathered exceptions (allowed without justification):**
- `testify/require` and `testify/assert` for tests (per Code conventions).
- `google/uuid` if UUIDs are needed — do not add alternatives.

**Adding a new third-party dependency:**
- The PR description must answer: what stdlib approach did you consider, and why is it materially worse? "More convenient" is not sufficient.
- Run `go mod graph | rg <new-pkg>` and note the transitive deps it pulls in. A small convenience that pulls 40 transitive packages is usually a net loss; call it out explicitly.
- Pin to a specific version. Updates go through Renovate, not ad hoc bumps.
- New direct deps are reviewed by a human, not auto-approved by the reviewer agent.

## Pre-PR proof gates

Every PR must include, in its description, the actual output of the following commands. "I ran it locally" is not acceptable — the literal output is the evidence. CI re-runs all of these and is the authoritative gate, but the agent pastes its own run so failures are caught one round-trip earlier.

**Always:**
- `go build ./...`
- `go vet ./...`
- `go test -race ./<changed-packages>/...`
- `golangci-lint run ./<changed-packages>/...`
- `gosec -quiet ./<changed-packages>/...`
- `govulncheck ./...`

**Conditionally, when the PR touches the relevant files:**
- Terraform changes (`**.tf`, `**.tfvars`) → `checkov -d <changed-dir> --framework terraform`
- Dockerfile changes → `checkov -f <Dockerfile> --framework dockerfile` and `hadolint <Dockerfile>`
- Kubernetes manifests (`**/k8s/**.yaml`) → `checkov -d <changed-dir> --framework kubernetes`
- Generated code (`*.pb.go`, `*_mock.go`, sqlc output) → `go generate ./...` and confirm `git diff` is empty

If any of these fail, the coder does not open the PR. It posts the failing output as an issue comment, re-labels `status:blocked-replan`, and ends its turn.

## Avoiding hallucinations

Agents fabricate import paths and function signatures more often than is comfortable. Defences:

- **No hand-edited `go.mod` or `go.sum`.** New imports go through `go get <pkg>@<version>` explicitly. The PR description names every new direct dep.
- **Every function call must be resolvable via gopls before you write it.** Use `textDocument/definition` or `workspace/symbol`. If gopls can't find it, you imagined it.
- **Especially careful with:** `slices`, `maps`, `cmp`, `log/slog` (stdlib APIs that have shifted between Go versions); cloud SDKs (GCP and AWS rename packages and change auth flows between major versions); anything you "remember" rather than just looked up.
- **Version-pin sensitivity.** The repo's Go version (`go.mod`'s `go` directive) is the source of truth. Do not use stdlib features newer than that version. Check with `go doc <pkg>.<symbol>` if uncertain.

## Test discipline

Coverage numbers are gameable and agents game them aggressively. Specify what to test, not how much.

- **Every error branch has a test that exercises it.** A test that only checks the happy path is incomplete.
- **Assertions check specific values, not just absence of error.** `require.NoError(t, err)` followed by nothing is meaningless. The next line must assert on the returned value.
- **No `time.Sleep` in tests.** Use `testing/synctest` (Go 1.24+) or inject a fake clock. Sleep-based tests are flaky and waste CI minutes.
- **No `t.Skip` without a referenced issue number.** A skipped test is a hidden failure; it must trace to a tracked TODO.
- **Mock at boundaries you own.** Mock the database driver via your repository interface, not `*sql.DB` directly. Never mock types from third-party packages — your test then asserts on your mock, not on real behaviour.
- **New tests must fail before the implementation and pass after.** Run the test against the unmodified branch first; if it passes, it isn't testing what you think.
- **Table-driven tests use `t.Run(tc.name, ...)`** so individual cases are addressable from the CLI.

## PR descriptions

The PR description is read by the reviewer agent and by humans. It must be factual, not promotional, and use this structure:
```

## What

<One-paragraph factual description of the change. No “cleaner”, “better”, “improved”.>

## Why

Closes #<issue-number>.

## How

<2-4 sentences on the approach. Reference specific decisions made and where assumptions in the issue’s Handover notes were applied.>

## Proof

<Paste the literal output of each command in the Pre-PR proof gates section. Use collapsible details blocks for long output.>

## Dependencies

<List any new direct dependencies and the justification per the Dependencies section. “None” if no new deps.>

## Caveats

<Anything the coder is uncertain about; anything the issue implied but this PR did NOT do; anything a reviewer should look at extra carefully. “None” only if genuinely none.>

```
No marketing language. No "this makes the code cleaner". State what changed and what was verified.

## Monorepo rules
- A service may import from `pkg/` and its own `internal/`. It may **not** import from another service's `internal/` or from another service directly.
- Cross-service contracts live in `pkg/proto/` (protobuf) or `pkg/api/` (typed clients).
- Database migrations live in `services/<name>/migrations/`. Never edit historic migrations.
- A `web/<app>/` frontend may only reach the Go side over the network (its public APIs) or via a generated TypeScript client. It must never import Go source, and Go code must never import from `web/`.

## Frontend (`web/`)

Go services here are headless. Anything user-facing — dashboards, admin UIs, customer apps — lives under `web/<app>/` and is owned by the frontend agent (`.opencode/agent/frontend.md`). The discipline mirrors the Go side: minimal dependencies, surgical changes, every line traceable to an acceptance criterion, and literal proof-gate output in the PR.

### Toolset (non-negotiable)
- **Bun for everything.** `bun install`, `bun run <script>`, `bun test`, `bunx`. **Never `npm`, `yarn`, or `pnpm`** — no `package-lock.json` or `yarn.lock`; the committed lockfile is `bun.lock`. CI installs with `bun install --frozen-lockfile`.
- **SvelteKit** (Svelte 5, runes) with **TypeScript in strict mode**. No React/Vue/Angular. No plain-JS app code; an `any` requires a comment justifying it.
- **Build/dev:** Vite via SvelteKit (`bun run dev`, `bun run build`). No custom Webpack/Rollup config unless SvelteKit can't express the need.
- **Styling:** scoped styles inside `.svelte` components, with one shared design-token layer for variables. No runtime CSS-in-JS.
- **State:** Svelte runes and stores. No Redux/MobX/Zustand.
- **Data:** SvelteKit `load` functions calling the Go APIs. API types come from the generated client (`pkg/api` / OpenAPI- or proto-generated TS) — do not hand-write types that duplicate the Go contract.
- **Lint/format/typecheck:** `svelte-check` (types + templates), `eslint`, `prettier` — all run through Bun scripts.
- **Tests:** `vitest` + `@testing-library/svelte` for units/components; Playwright for end-to-end.

### Dependencies (same bar as Go)

Prefer SvelteKit and platform built-ins. Add an npm dependency only when implementing it correctly ourselves is materially harder than vetting someone else's code. "More convenient" is not sufficient. Pin exact versions (no `^`/`~`); updates go through Renovate. A new direct dependency is justified in the PR description and reviewed by a human, not the reviewer agent.

### Pre-PR proof gates (frontend)

For any PR that touches `web/`, paste the literal output of these (run from the app dir):
- `bun install --frozen-lockfile`
- `bun run check`   (svelte-check / type + template check)
- `bun run lint`
- `bun run test`
- `bun run build`

CI re-runs these via a path-filtered workflow and is the authoritative gate. If any fail, the frontend agent does not open the PR — it posts the failing output as an issue comment, re-labels `status:blocked-replan`, and ends its turn.

## Handover protocol

All work is tracked as GitHub issues. See `.opencode/agent/planner.md`, `.opencode/agent/coder.md`, and `.opencode/agent/frontend.md` for role-specific rules. The issue is the single source of truth between agents.
