#!/usr/bin/env bash
#
# mode.sh — switch the project between online and GODARK (fully offline) mode.
#
#   godark   Switch agents to local Gemma models and force network tooling
#            offline. Run AFTER the online prep in `make godark`.
#   online   Switch agents back to the Gemini models and re-enable the network.
#   status   Print the current mode and each agent's active model.
#
# Online (the committed default) uses cloud models:
#   planner, reviewer  -> google/gemini-3.1-pro    (deep reasoning)
#   coder,   frontend  -> google/gemini-3.5-flash  (fast implementation)
# GODARK swaps in the local, runnable-offline models:
#   planner, reviewer  -> google/gemma-4-31B-it     (dense)
#   coder,   frontend  -> google/gemma-4-26B-A4B-it (MoE)
#
# The switch rewrites the `model:` field in .opencode/agent/*.md, sets a
# gitignored mode marker (.opencode/cache/mode) that other scripts honour, and
# toggles GOTOOLCHAIN so Go never tries to download a toolchain while dark.
#
# NOTE: godark modifies tracked agent files locally. Run `make online` to
# restore the Gemini defaults before committing.

set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

AGENT_DIR=".opencode/agent"
MODE_FILE=".opencode/cache/mode"

# agent -> "online_model|offline_model"
models_for() {
  case "$1" in
    planner|reviewer) echo "google/gemini-3.1-pro|google/gemma-4-31B-it" ;;
    coder|frontend)   echo "google/gemini-3.5-flash|google/gemma-4-26B-A4B-it" ;;
    *)                echo "" ;;
  esac
}

set_model() { # file model
  awk -v m="$2" '!d && /^model:[[:space:]]/ {print "model: " m; d=1; next} {print}' "$1" \
    > "$1.tmp" && mv "$1.tmp" "$1"
}

apply() { # mode: godark|online
  local mode="$1" f name map
  for f in "$AGENT_DIR"/*.md; do
    [ -e "$f" ] || continue
    name="$(basename "$f" .md)"
    map="$(models_for "$name")"; [ -n "$map" ] || continue
    if [ "$mode" = godark ]; then set_model "$f" "${map##*|}"; else set_model "$f" "${map%%|*}"; fi
  done
}

case "${1:-}" in
  godark)
    apply godark
    mkdir -p "$(dirname "$MODE_FILE")"; echo dark > "$MODE_FILE"
    command -v go >/dev/null 2>&1 && go env -w GOTOOLCHAIN=local 2>/dev/null || true
    echo "GODARK: local Gemma models active; network tooling forced offline." >&2
    ;;
  online)
    apply online
    rm -f "$MODE_FILE"
    command -v go >/dev/null 2>&1 && go env -u GOTOOLCHAIN 2>/dev/null || true
    echo "ONLINE: Gemini models active (planner/reviewer=gemini-3.1-pro, coder/frontend=gemini-3.5-flash)." >&2
    ;;
  status)
    if [ -f "$MODE_FILE" ] && [ "$(cat "$MODE_FILE" 2>/dev/null)" = dark ]; then
      echo "mode: GODARK (offline)"
    else
      echo "mode: online"
    fi
    for f in "$AGENT_DIR"/*.md; do
      [ -e "$f" ] || continue
      printf "  %-10s %s\n" "$(basename "$f" .md)" "$(grep -m1 '^model:' "$f" | sed 's/^model:[[:space:]]*//')"
    done
    ;;
  *)
    echo "usage: mode.sh {godark|online|status}" >&2; exit 1 ;;
esac
