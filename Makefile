# Makefile — install the developer tool dependencies this template expects.
#
# `make tools` installs everything the agents and CI reference (see AGENTS.md
# and .github/workflows/security.yml): the Go toolchain helpers, the linter,
# and the security scanners. A few tools (ripgrep, fd, gh) come from your OS
# package manager — run `make system-tools` to print the commands.
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
	@echo "  macOS (Homebrew):  brew install ripgrep fd gh"
	@echo "  Debian/Ubuntu:     sudo apt-get update && sudo apt-get install -y ripgrep fd-find gh"
	@echo "  Fedora:            sudo dnf install -y ripgrep fd-find gh"
	@echo "  Arch:              sudo pacman -S ripgrep fd github-cli"
	@echo ""
	@echo "Note: on Debian/Ubuntu the 'fd' binary is named 'fdfind'."

.PHONY: check
check: ## Print the version of each installed tool
	@for t in go gofmt goimports gopls golangci-lint gosec govulncheck checkov hadolint rg fd gh; do \
		if ! command -v $$t >/dev/null 2>&1; then \
			printf "  %-14s MISSING\n" "$$t"; \
		elif [ "$$t" = "go" ] || [ "$$t" = "gofmt" ]; then \
			printf "  %-14s %s\n" "$$t" "$$($(GO) version 2>&1 | head -n1)"; \
		else \
			printf "  %-14s %s\n" "$$t" "$$($$t --version 2>&1 | head -n1)"; \
		fi; \
	done
