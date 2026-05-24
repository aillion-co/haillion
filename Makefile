# Makefile — install the developer tool dependencies this template expects.
#
# `make tools` installs everything the agents and CI reference (see AGENTS.md
# and .github/workflows/security.yml): the Go toolchain helpers, the linter,
# and the security scanners. A few tools (ripgrep, fd, gh) come from your OS
# package manager — run `make system-tools` to print the commands.
#
# `make bootstrap` is the one command to prepare for working fully offline
# (planes, etc.): it installs the tools, warms the Go module cache, installs
# web/ deps + Playwright browsers, and populates docs/vendor/. Run it while
# online, then verify with `make offline-check`. See "Offline / disconnected
# work" in AGENTS.md for what does and does not work without a network.
#
# Go-based tools install into $(go env GOPATH)/bin; ensure that is on your PATH.
#
# Versions default to `latest`. Per AGENTS.md, pin them for reproducible
# environments by overriding on the command line, e.g.
#   make tools GOPLS_VERSION=v0.18.1 GOLANGCI_LINT_VERSION=v1.64.8

GO   ?= go
GOBIN := $(shell $(GO) env GOPATH)/bin

GOIMPORTS_VERSION     ?= latest
GOPLS_VERSION         ?= latest
GOLANGCI_LINT_VERSION ?= latest
GOSEC_VERSION         ?= latest
GOVULNCHECK_VERSION   ?= latest
HADOLINT_VERSION      ?= v2.12.0

.DEFAULT_GOAL := help

.PHONY: help
help: ## List available targets
	@grep -E '^[a-zA-Z0-9_-]+:.*## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*## "}{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: tools
tools: go-tools checkov hadolint ## Install all tool dependencies (Go tools + scanners)
	@echo ""
	@echo "Done. ripgrep, fd and gh are OS-managed — run 'make system-tools' for those."

.PHONY: bootstrap
bootstrap: tools go-cache frontend-deps docs ## Prepare for fully offline work — run this while online
	@./scripts/issues.sh pull || echo "bootstrap: issue cache skipped (gh not ready/offline) — run 'make issues-pull' later"
	@echo ""
	@echo "Offline prep complete. Verify with 'make offline-check'."
	@echo "Caveats: govulncheck needs the online vuln DB (defer to CI offline). The GitHub"
	@echo "Issues bus is cached for offline use ('make issues-sync' on reconnect), but"
	@echo "opening PRs still needs a network — see AGENTS.md 'Offline / disconnected work'."

.PHONY: issues-pull
issues-pull: ## Snapshot GitHub issues to the local offline cache (online)
	./scripts/issues.sh pull

.PHONY: issues-sync
issues-sync: ## Replay offline issue changes back to GitHub (online)
	./scripts/issues.sh sync

.PHONY: issues-status
issues-status: ## Show issue cache freshness + pending offline changes
	./scripts/issues.sh status

.PHONY: go-cache
go-cache: ## Warm the Go module cache + toolchain (no-op until go.mod exists)
	@if [ -f go.mod ]; then \
		echo "Warming Go module cache..."; \
		$(GO) mod download all && { $(GO) build ./... 2>/dev/null || true; }; \
	else \
		echo "go-cache: no go.mod yet — skipping (re-run once the module exists)"; \
	fi

.PHONY: frontend-deps
frontend-deps: ## Install web/ deps with Bun + Playwright browsers (no-op until an app exists)
	@if ! command -v bun >/dev/null 2>&1; then \
		echo "frontend-deps: bun not found — install it ('make system-tools') and re-run"; \
	elif ! ls web/*/package.json >/dev/null 2>&1; then \
		echo "frontend-deps: no web/*/package.json yet — skipping"; \
	else \
		for d in web/*/; do \
			[ -f "$$d/package.json" ] || continue; \
			echo "==> bun install ($$d)"; ( cd "$$d" && bun install ) || exit 1; \
			if grep -q '"@playwright/test"' "$$d/package.json"; then \
				echo "==> playwright install ($$d)"; ( cd "$$d" && bunx playwright install ) || exit 1; \
			fi; \
		done; \
	fi

.PHONY: offline-check
offline-check: ## Report whether this machine is ready to work offline
	@echo "Offline readiness:"
	@for t in go gopls goimports golangci-lint gosec govulncheck; do \
		if command -v $$t >/dev/null 2>&1; then printf "  [ok]   %s\n" "$$t"; else printf "  [MISS] %s — run 'make go-tools'\n" "$$t"; fi; \
	done
	@if [ -d docs/vendor ] && [ -n "$$(ls -A docs/vendor 2>/dev/null)" ]; then printf "  [ok]   docs/vendor populated\n"; else printf "  [MISS] docs/vendor — run 'make docs' online\n"; fi
	@if [ -f go.mod ]; then \
		if $(GO) mod verify >/dev/null 2>&1; then printf "  [ok]   go module cache\n"; else printf "  [MISS] go modules — run 'make go-cache' online\n"; fi; \
	else printf "  [n/a]  no go.mod yet\n"; fi
	@if ls web/*/package.json >/dev/null 2>&1; then \
		if ls -d web/*/node_modules >/dev/null 2>&1; then printf "  [ok]   web/ deps installed\n"; else printf "  [MISS] web/ deps — run 'make frontend-deps' online\n"; fi; \
	else printf "  [n/a]  no web/ app yet\n"; fi
	@if [ -f .opencode/cache/meta.json ]; then printf "  [ok]   issue cache present (see 'make issues-status')\n"; else printf "  [MISS] issue cache — run 'make issues-pull' online\n"; fi
	@pend=$$([ -f .opencode/cache/queue/journal.jsonl ] && grep -c . .opencode/cache/queue/journal.jsonl 2>/dev/null || echo 0); \
		printf "  [note] %s offline issue change(s) queued; 'make issues-sync' on reconnect\n" "$$pend"
	@printf "  [note] govulncheck needs the online vuln DB; defer to CI when offline\n"
	@printf "  [note] opening PRs needs the network; commit locally and 'gh pr create' on reconnect\n"

.PHONY: docs
docs: ## Vendor up-to-date product docs into docs/vendor/ (run where egress is open)
	./scripts/gen-docs.sh

.PHONY: map
map: ## Regenerate the repo map under .opencode/maps/
	./scripts/gen-repo-map.sh

.PHONY: go-tools
go-tools: ## Install Go tools: goimports, gopls, golangci-lint, gosec, govulncheck
	$(GO) install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)
	$(GO) install golang.org/x/tools/gopls@$(GOPLS_VERSION)
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	$(GO) install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)
	$(GO) install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

.PHONY: checkov
checkov: ## Install Checkov (IaC scanner) via pip --user
	@command -v python3 >/dev/null 2>&1 || { echo "python3 is required to install checkov"; exit 1; }
	python3 -m pip install --user --upgrade checkov

.PHONY: hadolint
hadolint: ## Install hadolint (Dockerfile linter) into GOPATH/bin
	@command -v curl >/dev/null 2>&1 || { echo "curl is required to install hadolint"; exit 1; }
	@mkdir -p "$(GOBIN)"
	@os=$$(uname -s); arch=$$(uname -m); \
		case "$$arch" in aarch64|arm64) arch=arm64 ;; x86_64|amd64) arch=x86_64 ;; esac; \
		url="https://github.com/hadolint/hadolint/releases/download/$(HADOLINT_VERSION)/hadolint-$$os-$$arch"; \
		echo "Downloading $$url"; \
		curl -fsSL "$$url" -o "$(GOBIN)/hadolint" && chmod +x "$(GOBIN)/hadolint"

.PHONY: system-tools
system-tools: ## Print install commands for OS-managed tools (ripgrep, fd, gh)
	@echo "Install these with your OS package manager:"
	@echo "  macOS (Homebrew):  brew install ripgrep fd gh jq"
	@echo "  Debian/Ubuntu:     sudo apt-get update && sudo apt-get install -y ripgrep fd-find gh jq"
	@echo "  Fedora:            sudo dnf install -y ripgrep fd-find gh jq"
	@echo "  Arch:              sudo pacman -S ripgrep fd github-cli jq"
	@echo ""
	@echo "Note: on Debian/Ubuntu the 'fd' binary is named 'fdfind'."
	@echo "Note: gh + jq are required by scripts/issues.sh (offline issue cache)."

.PHONY: check
check: ## Print the version of each installed tool
	@for t in go gofmt goimports gopls golangci-lint gosec govulncheck checkov hadolint rg fd gh jq; do \
		if ! command -v $$t >/dev/null 2>&1; then \
			printf "  %-14s MISSING\n" "$$t"; \
		elif [ "$$t" = "go" ] || [ "$$t" = "gofmt" ]; then \
			printf "  %-14s %s\n" "$$t" "$$($(GO) version 2>&1 | head -n1)"; \
		else \
			printf "  %-14s %s\n" "$$t" "$$($$t --version 2>&1 | head -n1)"; \
		fi; \
	done
