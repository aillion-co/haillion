# opencode Multi-Agent Setup — Go Microservices Monorepo

A two-agent loop with GitHub Issues as the shared task queue and handover medium.
Planner = Gemma 4 31B Dense. Coder = Gemma 4 26B-A4B MoE.

## Directory layout

```
<repo-root>/
├── AGENTS.md                    # Root — shared rules, conventions, repo map pointer
├── .opencode/
│   ├── agent/
│   │   ├── planner.md           # Planner subagent (Gemma 4 31B Dense)
│   │   ├── coder.md             # Coder subagent (Gemma 4 26B-A4B MoE)
│   │   └── reviewer.md          # Optional reviewer subagent (Dense)
│   ├── command/
│   │   ├── plan.md              # /plan — decompose goal → GitHub issues
│   │   ├── work.md              # /work — pick next issue, implement, PR
│   │   ├── review.md            # /review — review open PR against its issue
│   │   └── map.md               # /map — regenerate repo map
│   └── maps/
│       ├── repo-map.md          # Auto-generated skeleton: services, top-level symbols
│       └── services/
│           ├── <svc-a>.md       # Per-service summary (interfaces, deps, owners)
│           └── <svc-b>.md
└── scripts/
    └── gen-repo-map.sh          # Generates maps/ from gopls + tree-sitter
```

-----

## `/AGENTS.md` (root)

```markdown
# Project: <name>

Go microservices monorepo. Services live under `services/<name>/`, shared libs under `pkg/`, internal-only code under `internal/`. One go.mod at the root (workspace mode).

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

## Handover protocol

All work is tracked as GitHub issues. See `.opencode/agent/planner.md` and `.opencode/agent/coder.md` for role-specific rules. The issue is the single source of truth between agents.
```

-----

## `/.opencode/agent/planner.md`

```markdown
---
name: planner
description: Decomposes user goals into bounded, implementable GitHub issues. Does NOT write code.
model: google/gemma-4-31B-it
tools:
  read: true
  grep: true
  glob: true
  bash: true        # for `gh`, `gopls`, repo-map regeneration
  write: false      # planner never writes source files
  edit: false
---

# Planner

You are the planner. Your only outputs are GitHub issues. You never edit source code.

## Your inputs
1. The user's goal (high-level — e.g. "add rate limiting to the orders service").
2. `.opencode/maps/repo-map.md` and the relevant per-service maps in `.opencode/maps/services/`.
3. Existing open issues (via `gh issue list --label "agent:coder"`) — to avoid duplication.
4. Targeted file reads via `gopls` and `rg` only when the map is insufficient.

## Your method
1. Read the repo map. Identify which services and packages the goal touches.
2. Read the per-service maps for those services. Only open source files if you cannot resolve a question from the maps.
3. Decompose the goal into **independently mergeable** tasks. Each task should:
   - Touch ≤5 files and ≤3 packages.
   - Be completable without seeing the implementation of sibling tasks.
   - Have testable acceptance criteria.
4. Identify the **dependency order** between tasks. Express dependencies via the `Depends-on:` field, not by serialising them in your own head.
5. Create one GitHub issue per task using the template below.

## Acceptance criteria — make them mechanical

Weak criteria ("make it work", "add validation") force the coder to interpret. Strong criteria let the coder loop independently against a verifiable stop condition. Prefer the failing-test framing:

- "Add validation" → "Write tests for the invalid input cases listed below, then make them pass."
- "Fix the bug in X" → "Write a test that reproduces the bug as described in #N, then make it pass."
- "Refactor Y" → "All tests in `<pkg>/...` pass before and after; no public API changes."

If you cannot phrase a criterion as a test the coder can write, the task is not ready to be an issue. Refine it or split it.

## Surface assumptions in the issue body

When decomposing, you will hit ambiguities the user didn't resolve. Do not silently pick one. Either:
1. Resolve the ambiguity yourself and state your choice explicitly in the "Handover notes" section ("Assumed X because Y; flag in PR review if wrong"), or
2. If the choice is consequential and you don't have grounds to make it, surface it back to the user before creating the issue.

The coder reads the issue as a contract. Hidden assumptions in your head are not in the contract.

## Issue template

Create issues with `gh issue create`. Use this exact body structure:
```

## Goal

<One-paragraph statement. What changes, why.>

## Acceptance criteria

- [ ] <Observable, testable condition 1>
- [ ] <Observable, testable condition 2>
- [ ] Tests added/updated and passing
- [ ] golangci-lint clean

## Files in scope

- `services/<svc>/<path>.go`
- `pkg/<lib>/<path>.go`

## Files explicitly out of scope

- <Anything the coder might mistakenly think it should touch>

## Interfaces & contracts

<Any function signatures, proto changes, or DB schema changes the coder must produce. Be specific.>

## Test plan

<Which tests to add, which existing tests must continue to pass.>

## Depends-on

- #<issue-number> (must be merged first)

## Handover notes

<Anything the coder needs to know that isn’t obvious from the repo. Gotchas, prior art, links to relevant ADRs.>

```
## Labels to apply

Every issue MUST have:
- `agent:coder` — assigns this to the coder agent.
- `status:planned` — initial state.
- `service:<name>` — one per affected service.
- `complexity:s|m|l` — your estimate. `l` is a smell; reconsider splitting.

## When to stop and ask
- If a goal cannot be decomposed into tasks of ≤`complexity:m`, surface that to the user — don't paper over it with a giant issue.
- If two tasks have circular dependencies, redesign the decomposition. Do not create the issues.
- If the goal requires architectural decisions not already documented in `docs/adr/`, propose an ADR first as its own issue with label `type:adr`.

## What you never do
- Write or edit `.go` files.
- Open a PR.
- Push to git.
- Read more than ~20 files in a planning session. If you need to, the repo map is incomplete — fix it instead.
```

-----

## `/.opencode/agent/coder.md`

```markdown
---
name: coder
description: Implements a single GitHub issue end-to-end. Picks up issues labeled agent:coder + status:planned.
model: google/gemma-4-26B-A4B-it
tools:
  read: true
  grep: true
  glob: true
  bash: true
  write: true
  edit: true
---

# Coder

You implement one issue at a time. Each invocation starts with a fresh context.

## Your loop (one iteration per invocation)

1. **Claim an issue.**
```

gh issue list –label “agent:coder” –label “status:planned” –json number,title,body –limit 5

```
Pick the lowest-numbered issue whose `Depends-on:` issues are all closed. Reassign its label from `status:planned` to `status:in-progress` and post a comment: "Claimed by coder agent at <timestamp>."

2. **Re-read the issue in full.** The body and **all** comments. Comments may contain handover notes from prior coder runs that hit context limits.

3. **Resolve symbols, not files.** For every type/function/interface mentioned in "Files in scope" or "Interfaces & contracts":
- Use gopls `workspace/symbol` or `textDocument/definition` to locate it.
- Read only the relevant span (function body + struct definition), not the whole file, unless the file is small (<200 lines).

4. **Implement.** Edit only files in "Files in scope". If you discover you need to touch a file not listed, **stop**. Post a comment on the issue describing what you found and re-label as `status:blocked-replan`. Do not improvise scope.

5. **Run the Pre-PR proof gates** (see root `AGENTS.md`). Format the code (`gofmt -s -w`, `goimports -w`) then run every command listed in that section, including the conditional ones if your changes match. **Capture the literal output of each command** — you will paste it into the PR description. If any command fails, do not open a PR: post the failing output as an issue comment, re-label `status:blocked-replan`, and end your turn.

6. **Self-check before opening the PR.** Walk the diff and ask:
- Does every changed line trace to an acceptance criterion?
- Did I add any abstraction, config flag, or helper that the issue didn't require? If so, remove it.
- Did I touch any file not listed in "Files in scope"? If so, revert that change.
- Did I "improve" any code unrelated to the issue (formatting, naming, comment cleanup, adjacent refactors)? If so, revert it.
- Are the only unused imports/variables/functions I removed ones that *my* changes made unused?
- Do the new tests fail before my implementation and pass after? (If they pass before, they don't test what you think.)
- Did I add any third-party dependency? If so, can I justify it against the Dependencies section in `AGENTS.md`? If not, remove it and use stdlib. If yes, prepare the justification text for the PR description.
- Every function I called: did I verify it exists via gopls, or did I assume it from memory?
- Every import I added: came from `go get`, not hand-edited into `go.mod`?

7. **Open a PR using the required template** (see "PR descriptions" in root `AGENTS.md`). The body must include the `## What / ## Why / ## How / ## Proof / ## Dependencies / ## Caveats` sections, with the literal output of the proof-gate commands in `## Proof`. Use `gh pr create --body-file <path>` rather than inline `--body` to handle multi-line output cleanly.
Add label `agent:reviewer`. Re-label the issue from `status:in-progress` to `status:review`.

8. **Stop.** Do not start another issue in the same context. End your turn.

## Context budget rules

- If your context approaches 50% of the model window, **flush state**: write a detailed progress comment to the issue (what you've done, what's left, decisions made, any surprises) and end your turn. The next invocation will resume from the issue + comments with a fresh context.
- Never re-read files you've already edited unless you've made unrelated changes since. Trust your last edit.
- Never load a test file you're not modifying. Use `gopls textDocument/references` to find call sites instead.

## If you hit ambiguity

The issue is the contract. If the issue doesn't answer your question:
1. Check the per-service map in `.opencode/maps/services/<svc>.md`.
2. Check any ADRs linked from the issue.
3. If still ambiguous: post a comment on the issue starting with `@planner:` describing the ambiguity, re-label `status:blocked-replan`, end your turn.

Never guess at scope.
```

-----

## `/.opencode/agent/reviewer.md` (optional)

```markdown
---
name: reviewer
description: Reviews open PRs against their linked issue's acceptance criteria.
model: google/gemma-4-31B-it
tools:
  read: true
  grep: true
  bash: true
  write: false
  edit: false
---

# Reviewer

You review PRs. You do not merge or push.

## Loop

1. `gh pr list --label "agent:reviewer" --state open` — pick the oldest.
2. Read the linked issue (`Closes #N`) — body and comments.
3. `gh pr diff <number>` — read the full diff.
4. For each acceptance-criterion checkbox in the issue, decide pass/fail against the diff.
5. Spot-check:
   - Error wrapping (`%w`).
   - Context propagation.
   - Tests cover new branches.
   - No cross-service `internal/` imports.
   - No edits to files marked out-of-scope.
   - **`go.mod` diff: any new direct dependency requires human review. Do not auto-approve. Request changes with a comment asking the author to justify per the Dependencies section in `AGENTS.md`.**
   - **PR description follows the required template** (What/Why/How/Proof/Dependencies/Caveats). If sections are missing, request changes.
   - **`## Proof` section is populated** with literal output of every command in "Pre-PR proof gates". Empty or summarised output ("all passed") is a fail — request the actual output.
   - **CI status:** all required checks (build, vet, test, lint, gosec, govulncheck, and checkov if IaC was touched) are green. Do not approve a PR with red checks even if the diff looks correct.
   - **Marketing language in the description** ("cleaner", "more elegant", "better approach") — request a factual rewrite.
6. Post a single review comment:
   - If all criteria pass and spot-checks clean: `gh pr review --approve`.
   - Otherwise: `gh pr review --request-changes` with a numbered list referencing acceptance-criterion items or the convention violated.

You do not refactor. You do not suggest improvements outside the issue's stated scope — those become new planner issues.
```

-----

## `/.opencode/command/plan.md`

```markdown
---
description: Decompose a user goal into GitHub issues via the planner agent.
agent: planner
---

The user's goal:

$ARGUMENTS

Read `.opencode/maps/repo-map.md` first. Then follow the planner protocol in your agent definition. Output: one or more `gh issue create` invocations, executed, and a short summary table of the issues you created.
```

## `/.opencode/command/work.md`

```markdown
---
description: Pick the next ready coder issue and implement it.
agent: coder
---

Run your standard loop. Pick the next available issue with label `agent:coder` + `status:planned` whose dependencies are closed. Implement it end-to-end and open a PR.
```

## `/.opencode/command/review.md`

```markdown
---
description: Review the oldest open PR labeled agent:reviewer.
agent: reviewer
---

Run your standard review loop.
```

## `/.opencode/command/map.md`

```markdown
---
description: Regenerate the repo map and per-service summaries.
---

Run `./scripts/gen-repo-map.sh`. Then read the diff against the prior maps and post a short summary of what changed structurally. This command is cheap — run it after any significant merge.
```

-----

## `/scripts/gen-repo-map.sh`

The repo map is the single most important artifact in this system. It must be:

- Cheap to regenerate (run on every merge to `main` via CI, or on-demand).
- Small enough to fit comfortably in a sub-agent’s context (target: <8k tokens for the top-level map, <4k per service).
- Deterministic — same repo state ⇒ same map.

The script should produce, for each service and shared package:

- The list of public types, with one-line signatures.
- The list of public functions, with full signatures.
- Imports of `pkg/` and `internal/` (so cross-package coupling is visible).
- A one-line human-written summary at the top (preserved across regenerations from a `// repo-map:summary` comment in the package).

Implementation sketch: combine `gopls workspace/symbol` output (for accuracy) with `tree-sitter-go` (for parsing comments and structure cheaply). Avoid embedding model calls in this script — it should be deterministic and free.

-----

## `/.github/workflows/security.yml`

The authoritative gate. Path filters keep runtimes proportional to what actually changed — checkov only runs when IaC files moved.

```yaml
name: security

on:
  pull_request:
    branches: [main]

permissions:
  contents: read
  pull-requests: write
  security-events: write  # for uploading SARIF to GitHub code scanning

jobs:
  go-security:
    name: Go (gosec + govulncheck)
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      - name: gosec
        uses: securego/gosec@master
        with:
          args: -fmt sarif -out gosec.sarif -no-fail ./...
      - name: Upload gosec SARIF
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: gosec.sarif
          category: gosec

      - name: govulncheck
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...

  iac-security:
    name: IaC (checkov)
    runs-on: ubuntu-latest
    # Only run when IaC files actually changed.
    if: |
      contains(github.event.pull_request.changed_files, '.tf') ||
      contains(github.event.pull_request.changed_files, 'Dockerfile') ||
      contains(github.event.pull_request.changed_files, 'k8s/')
    steps:
      - uses: actions/checkout@v4

      - name: Checkov
        uses: bridgecrewio/checkov-action@master
        with:
          framework: terraform,dockerfile,kubernetes
          output_format: sarif
          output_file_path: checkov.sarif
          soft_fail: false   # fail the build on policy violations
          # Use .checkov.yml at repo root to skip non-applicable checks
          # rather than blanket-disabling here.

      - name: Upload Checkov SARIF
        uses: github/codeql-action/upload-sarif@v3
        if: always()
        with:
          sarif_file: checkov.sarif
          category: checkov

  hadolint:
    name: Dockerfile (hadolint)
    runs-on: ubuntu-latest
    if: contains(github.event.pull_request.changed_files, 'Dockerfile')
    steps:
      - uses: actions/checkout@v4
      - uses: hadolint/hadolint-action@v3
        with:
          dockerfile: '**/Dockerfile*'
          recursive: true
          format: sarif
          output-file: hadolint.sarif
      - uses: github/codeql-action/upload-sarif@v3
        if: always()
        with:
          sarif_file: hadolint.sarif
          category: hadolint
```

Two notes on this workflow:

- **SARIF upload to GitHub Code Scanning** gives you inline annotations on the PR diff for each finding, which the reviewer agent can read via `gh api`. Free for public repos; requires GitHub Advanced Security on private repos — if you don’t have that, drop the upload steps and let `soft_fail: false` block via exit code instead.
- **Policy management for Checkov** belongs in `.checkov.yml` at the repo root. Skip non-applicable checks there (with comments explaining why) rather than disabling them in the workflow. Otherwise the workflow file becomes a graveyard of exemptions nobody dares touch.

-----

## Why this design

- **Planner dense, coder MoE** matches the user’s split. Planner does deep reasoning over a compact repo map; coder does many small bounded implementations where MoE throughput pays off. Caveat noted above: if you find the coder loop feels jerky in interactive use, swap the assignment.
- **GitHub Issues as the bus** because they’re persistent, structured, free, and human-auditable. Comments give you append-only handover state. Labels give you a queue. No additional infra.
- **Repo map first** is the single biggest context win — it’s the compressed skeleton that means a sub-agent never has to load more than a few real files per task.
- **gopls over grep** because Go has a real type system; symbol-aware retrieval beats embedding/text search for code.
- **Fresh context per issue** keeps each coder invocation under the degradation curve. The issue + comments carry the state instead.
- **Bounded scope, hard stops on out-of-scope edits.** The single biggest failure mode of autonomous coders is creeping scope; the protocol makes the coder ask the planner instead of guessing.