#!/usr/bin/env bash
#
# issues.sh — offline-capable interface to the GitHub Issues task bus.
#
# The agent workflow uses GitHub Issues for handover, but that bus is online.
# This tool lets a developer (or agent) read and mutate issues offline and
# reconcile on reconnect:
#
#   pull                 Snapshot issues to the local cache (online).
#   list [--label L]...  List cached issues (offline-first).
#        [--state S]
#   view <n>             Show a cached issue: body, labels, comments.
#   comment <n> -b TEXT  Add a comment.  online: post now.  offline: queue.
#           <n> -F FILE
#   relabel <n> --add L --remove L     Change labels (online now / offline queued).
#   state <n> open|closed              Open or close an issue.
#   create -F FILE --title T --label L Create an issue. offline: gets a LOCAL-n id.
#   sync [--dry-run]     Replay the offline journal against GitHub (online).
#   status               Cache freshness + pending queue summary.
#
# Reads are always served from the local cache (refresh it with `pull`), so
# behaviour is identical on a plane and at a desk. Writes go straight to GitHub
# when online, or to an append-only journal when offline; `sync` replays the
# journal — comments are append-only (always safe), label/state changes apply
# as deltas (concurrent edits by others are preserved), and created issues get
# their real number with a temp -> real id remap. Genuine conflicts are written
# to .opencode/cache/conflicts.md for human review rather than force-applied.
#
# Storage lives under .opencode/cache/ and is gitignored (per-machine).
# Connectivity: auto-detected; override with ISSUES_OFFLINE=1 or ISSUES_ONLINE=1.
#
# Dependencies: gh (authenticated), jq.

set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

CACHE=".opencode/cache"
ISSUES_DIR="$CACHE/issues"
INDEX="$ISSUES_DIR/index.json"
META="$CACHE/meta.json"
QUEUE="$CACHE/queue"
JOURNAL="$QUEUE/journal.jsonl"
IDMAP="$QUEUE/idmap.json"
SYNCED="$CACHE/synced.log"
CONFLICTS="$CACHE/conflicts.md"

ISSUE_FIELDS="number,title,body,state,labels,author,createdAt,updatedAt,comments"

die() { echo "issues: $*" >&2; exit 1; }
now() { date -u +%Y-%m-%dT%H:%M:%SZ; }
command -v jq >/dev/null 2>&1 || die "jq is required"

ensure_dirs() { mkdir -p "$ISSUES_DIR" "$QUEUE"; }

online() {
  [ -n "${ISSUES_OFFLINE:-}" ] && return 1
  command -v gh >/dev/null 2>&1 || return 1
  [ -n "${ISSUES_ONLINE:-}" ] && return 0
  timeout 8 gh api rate_limit >/dev/null 2>&1
}

cached_repo() { [ -f "$META" ] && jq -r '.repo // empty' "$META" || true; }

issue_file() { echo "$ISSUES_DIR/$1.json"; }

# Rebuild index.json from every cached issue file (real + LOCAL-*), robust to
# empty globs so it never wipes a valid index when one set is absent.
index_rebuild() {
  local files=() f
  for f in "$ISSUES_DIR"/*.json; do
    [ -e "$f" ] || continue
    [ "$f" = "$INDEX" ] && continue
    files+=("$f")
  done
  if [ ${#files[@]} -gt 0 ]; then
    jq -s 'map({number,title,state,labels,updatedAt})' "${files[@]}" > "$INDEX"
  else
    echo '[]' > "$INDEX"
  fi
}

require_cached() {
  [ -f "$(issue_file "$1")" ] || die "issue $1 not in cache — run 'issues.sh pull' (online)"
}

# --- pull --------------------------------------------------------------------
cmd_pull() {
  online || die "pull needs a network connection (and authenticated gh)"
  ensure_dirs
  local repo limit="${1:-500}"
  repo="$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null)" \
    || die "could not determine repo (is gh authenticated, in a repo?)"

  echo "Pulling issues for $repo ..." >&2
  local numbers
  numbers="$(gh issue list --repo "$repo" --state all --limit "$limit" \
    --json number -q '.[].number')" || die "gh issue list failed"

  # Preserve any locally-created (LOCAL-*) issues that haven't synced yet.
  local n
  for n in $numbers; do
    gh issue view "$n" --repo "$repo" --json "$ISSUE_FIELDS" >"$(issue_file "$n")" \
      || echo "issues: warning: failed to fetch #$n" >&2
  done

  # Rebuild index from cached issues (keeps any unsynced LOCAL-* entries too).
  index_rebuild

  jq -n --arg repo "$repo" --arg t "$(now)" --argjson count "$(echo "$numbers" | grep -c . || echo 0)" \
    '{repo:$repo, pulled_at:$t, count:$count}' > "$META"
  echo "Cached $(echo "$numbers" | grep -c . || echo 0) issues at $(now)." >&2
}

# --- list / view -------------------------------------------------------------
cmd_list() {
  local want_label=() want_state=""
  while [ $# -gt 0 ]; do
    case "$1" in
      --label) want_label+=("$2"); shift 2 ;;
      --state) want_state="$2"; shift 2 ;;
      *) die "list: unknown arg '$1'" ;;
    esac
  done
  [ -f "$INDEX" ] || die "no cache — run 'issues.sh pull' (online)"

  local filter='.'
  local l
  for l in "${want_label[@]:-}"; do
    [ -z "$l" ] && continue
    filter="$filter | map(select(.labels // [] | map(.name) | index(\"$l\")))"
  done
  if [ -n "$want_state" ]; then
    local up; up="$(echo "$want_state" | tr '[:lower:]' '[:upper:]')"
    filter="$filter | map(select((.state // \"\") == \"$up\"))"
  fi

  jq -r "$filter"' | sort_by(.number)[] |
    ["#\(.number)", "[\(.state)]", ((.labels//[]|map(.name)|join(","))), .title] | @tsv' "$INDEX" \
    | awk -F'\t' '{printf "%-8s %-9s %-32s %s\n", $1, $2, $3, $4}'
}

cmd_view() {
  local n="${1:-}"; [ -n "$n" ] || die "view: need an issue number"
  require_cached "$n"
  jq -r '
    "#\(.number)  \(.title)",
    "state: \(.state)   labels: \((.labels//[]|map(.name)|join(", ")))",
    "author: \(.author.login // "?")   updated: \(.updatedAt // "?")",
    "",
    (.body // "(no body)"),
    "",
    "── comments (\((.comments//[])|length)) ──",
    ((.comments//[])[] |
      "\n[\(.author.login // .author // "?")\(if ._queued then " · QUEUED" else "" end)] \(.createdAt // "")\n\(.body)")
  ' "$(issue_file "$n")"
}

# --- journal helpers ---------------------------------------------------------
journal_count() { [ -f "$JOURNAL" ] && grep -c . "$JOURNAL" 2>/dev/null || echo 0; }

next_journal_id() { printf 'j-%05d' "$(( $(journal_count) + 1 ))"; }

next_temp_id() {
  local max=0 n
  if [ -f "$JOURNAL" ]; then
    while IFS= read -r n; do [ "$n" -gt "$max" ] 2>/dev/null && max="$n"; done < <(
      jq -r 'select(.op=="create") | .temp | ltrimstr("LOCAL-")' "$JOURNAL" 2>/dev/null)
  fi
  echo "LOCAL-$(( max + 1 ))"
}

append_journal() { ensure_dirs; printf '%s\n' "$1" >> "$JOURNAL"; }

read_body() { # --from-args: -b TEXT | -F FILE -> stdout
  if [ "$1" = "-b" ]; then printf '%s' "$2";
  elif [ "$1" = "-F" ]; then [ -f "$2" ] || die "no such file: $2"; cat "$2";
  else die "expected -b TEXT or -F FILE"; fi
}

# --- comment -----------------------------------------------------------------
cmd_comment() {
  local n="${1:-}"; shift || true
  [ -n "$n" ] || die "comment: need an issue number"
  local body; body="$(read_body "${1:-}" "${2:-}")"
  [ -n "$body" ] || die "comment: empty body"

  if online; then
    local repo; repo="$(cached_repo)"; repo="${repo:-$(gh repo view --json nameWithOwner -q .nameWithOwner)}"
    printf '%s' "$body" | gh issue comment "$n" --repo "$repo" --body-file - \
      && echo "Commented on #$n." >&2 || die "gh issue comment failed"
    return
  fi

  require_cached "$n"
  ensure_dirs
  local id; id="$(next_journal_id)"
  printf '%s' "$body" > "$QUEUE/$id.md"
  append_journal "$(jq -nc --arg id "$id" --arg op comment --arg issue "$n" \
    --arg ts "$(now)" --arg bf "$QUEUE/$id.md" \
    '{id:$id,op:$op,issue:$issue,ts:$ts,body_file:$bf}')"
  # Optimistic local update so the agent sees its own comment this session.
  local f; f="$(issue_file "$n")"
  jq --arg b "$body" --arg t "$(now)" \
    '.comments = ((.comments//[]) + [{author:{login:"(queued)"},body:$b,createdAt:$t,_queued:true}])' \
    "$f" > "$f.tmp" && mv "$f.tmp" "$f"
  echo "Queued comment on #$n ($id). Run 'make issues-sync' when online." >&2
}

# --- relabel -----------------------------------------------------------------
cmd_relabel() {
  local n="${1:-}"; shift || true
  [ -n "$n" ] || die "relabel: need an issue number"
  local add=() rem=()
  while [ $# -gt 0 ]; do
    case "$1" in
      --add) add+=("$2"); shift 2 ;;
      --remove) rem+=("$2"); shift 2 ;;
      *) die "relabel: unknown arg '$1'" ;;
    esac
  done
  [ ${#add[@]} -eq 0 ] && [ ${#rem[@]} -eq 0 ] && die "relabel: nothing to do"
  local add_json rem_json
  add_json="$(printf '%s\n' "${add[@]:-}" | jq -R . | jq -sc 'map(select(length>0))')"
  rem_json="$(printf '%s\n' "${rem[@]:-}" | jq -R . | jq -sc 'map(select(length>0))')"

  if online; then
    local repo; repo="$(cached_repo)"; repo="${repo:-$(gh repo view --json nameWithOwner -q .nameWithOwner)}"
    local args=()
    [ "$add_json" != "[]" ] && args+=(--add-label "$(IFS=,; echo "${add[*]}")")
    [ "$rem_json" != "[]" ] && args+=(--remove-label "$(IFS=,; echo "${rem[*]}")")
    gh issue edit "$n" --repo "$repo" "${args[@]}" >/dev/null \
      && echo "Relabelled #$n." >&2 || die "gh issue edit failed"
    return
  fi

  require_cached "$n"
  append_journal "$(jq -nc --arg id "$(next_journal_id)" --arg issue "$n" --arg ts "$(now)" \
    --argjson add "$add_json" --argjson remove "$rem_json" \
    '{id:$id,op:"relabel",issue:$issue,ts:$ts,add:$add,remove:$remove}')"
  local f; f="$(issue_file "$n")"
  jq --argjson add "$add_json" --argjson rem "$rem_json" '
    .labels = ((.labels//[]) | map(select(.name as $x | ($rem|index($x))|not)))
            + ($add | map({name:.}))
    | .labels |= unique_by(.name)' "$f" > "$f.tmp" && mv "$f.tmp" "$f"
  echo "Queued relabel on #$n. Run 'make issues-sync' when online." >&2
}

# --- state -------------------------------------------------------------------
cmd_state() {
  local n="${1:-}" s="${2:-}"
  [ -n "$n" ] || die "state: need an issue number"
  case "$s" in open|closed) ;; *) die "state: expected 'open' or 'closed'" ;; esac

  if online; then
    local repo; repo="$(cached_repo)"; repo="${repo:-$(gh repo view --json nameWithOwner -q .nameWithOwner)}"
    if [ "$s" = closed ]; then gh issue close "$n" --repo "$repo" >/dev/null;
    else gh issue reopen "$n" --repo "$repo" >/dev/null; fi \
      && echo "Set #$n to $s." >&2 || die "gh state change failed"
    return
  fi

  require_cached "$n"
  append_journal "$(jq -nc --arg id "$(next_journal_id)" --arg issue "$n" --arg ts "$(now)" --arg st "$s" \
    '{id:$id,op:"state",issue:$issue,ts:$ts,state:$st}')"
  local f up; f="$(issue_file "$n")"; up="$(echo "$s" | tr '[:lower:]' '[:upper:]')"
  jq --arg s "$up" '.state=$s' "$f" > "$f.tmp" && mv "$f.tmp" "$f"
  echo "Queued state=$s on #$n. Run 'make issues-sync' when online." >&2
}

# --- create ------------------------------------------------------------------
cmd_create() {
  local title="" bodyfile="" labels=()
  while [ $# -gt 0 ]; do
    case "$1" in
      --title) title="$2"; shift 2 ;;
      -F) bodyfile="$2"; shift 2 ;;
      --label) labels+=("$2"); shift 2 ;;
      *) die "create: unknown arg '$1'" ;;
    esac
  done
  [ -n "$title" ] || die "create: --title is required"
  [ -n "$bodyfile" ] && [ -f "$bodyfile" ] || die "create: -F FILE (body) is required"
  local labels_json; labels_json="$(printf '%s\n' "${labels[@]:-}" | jq -R . | jq -sc 'map(select(length>0))')"

  if online; then
    local repo; repo="$(cached_repo)"; repo="${repo:-$(gh repo view --json nameWithOwner -q .nameWithOwner)}"
    local args=(); local l
    for l in "${labels[@]:-}"; do [ -n "$l" ] && args+=(--label "$l"); done
    local url; url="$(gh issue create --repo "$repo" --title "$title" --body-file "$bodyfile" "${args[@]}")" \
      || die "gh issue create failed"
    echo "Created ${url##*/}: $url" >&2
    return
  fi

  ensure_dirs
  local temp id; temp="$(next_temp_id)"; id="$(next_journal_id)"
  cp "$bodyfile" "$QUEUE/$id.md"
  append_journal "$(jq -nc --arg id "$id" --arg temp "$temp" --arg ts "$(now)" \
    --arg title "$title" --arg bf "$QUEUE/$id.md" --argjson labels "$labels_json" \
    '{id:$id,op:"create",temp:$temp,ts:$ts,title:$title,body_file:$bf,labels:$labels}')"
  jq -n --arg num "$temp" --arg title "$title" --arg body "$(cat "$bodyfile")" \
    --arg t "$(now)" --argjson labels "$labels_json" \
    '{number:$num,title:$title,body:$body,state:"OPEN",labels:($labels|map({name:.})),
      author:{login:"(local)"},comments:[],createdAt:$t,updatedAt:$t,_local:true}' \
    > "$(issue_file "$temp")"
  index_rebuild
  echo "Queued new issue $temp: $title. Run 'make issues-sync' when online." >&2
}

# --- sync --------------------------------------------------------------------
map_get() { [ -f "$IDMAP" ] && jq -r --arg k "$1" '.[$k] // empty' "$IDMAP" || true; }
map_put() {
  ensure_dirs; [ -f "$IDMAP" ] || echo '{}' > "$IDMAP"
  jq --arg k "$1" --arg v "$2" '.[$k]=$v' "$IDMAP" > "$IDMAP.tmp" && mv "$IDMAP.tmp" "$IDMAP"
}
resolve_issue() { # temp/real -> real number (or empty if unmapped temp)
  case "$1" in LOCAL-*) map_get "$1" ;; *) echo "$1" ;; esac
}
# Rewrite "#LOCAL-n" references in a body file to their real numbers (stdout).
rewrite_refs() {
  local content; content="$(cat "$1")"
  if [ -f "$IDMAP" ]; then
    local k v
    while IFS=$'\t' read -r k v; do
      [ -n "$k" ] || continue
      content="${content//#$k/#$v}"
    done < <(jq -r 'to_entries[] | "\(.key)\t\(.value)"' "$IDMAP")
  fi
  printf '%s' "$content"
}
conflict() { ensure_dirs; printf -- '- [%s] %s\n' "$(now)" "$1" >> "$CONFLICTS"; echo "issues: CONFLICT: $1" >&2; }

cmd_sync() {
  local dry=0; [ "${1:-}" = "--dry-run" ] && dry=1
  [ -f "$JOURNAL" ] && [ "$(journal_count)" -gt 0 ] || { echo "Nothing queued." >&2; return 0; }
  online || die "sync needs a network connection (and authenticated gh)"
  local repo; repo="$(cached_repo)"; repo="${repo:-$(gh repo view --json nameWithOwner -q .nameWithOwner)}"
  ensure_dirs; [ -f "$IDMAP" ] || echo '{}' > "$IDMAP"

  local remaining="$QUEUE/journal.remaining"; : > "$remaining"
  local applied=0 failed=0 line op id issue target

  while IFS= read -r line; do
    [ -n "$line" ] || continue
    op="$(jq -r '.op' <<<"$line")"
    id="$(jq -r '.id' <<<"$line")"
    case "$op" in
      create)
        local temp title bf; temp="$(jq -r '.temp' <<<"$line")"
        title="$(jq -r '.title' <<<"$line")"; bf="$(jq -r '.body_file' <<<"$line")"
        local args=(); while IFS= read -r l; do [ -n "$l" ] && args+=(--label "$l"); done < <(jq -r '.labels[]?' <<<"$line")
        if [ "$dry" = 1 ]; then echo "DRY create '$title' ($temp) labels=$(jq -rc '.labels' <<<"$line")"; continue; fi
        local body url num; body="$(rewrite_refs "$bf")"
        url="$(printf '%s' "$body" | gh issue create --repo "$repo" --title "$title" --body-file - "${args[@]}" 2>/dev/null)"
        if [ -n "$url" ]; then num="${url##*/}"; map_put "$temp" "$num"
          echo "created $temp -> #$num"; applied=$((applied+1))
        else conflict "create '$title' ($temp) failed"; echo "$line" >> "$remaining"; failed=$((failed+1)); fi
        ;;
      comment)
        issue="$(jq -r '.issue' <<<"$line")"; target="$(resolve_issue "$issue")"
        local bf; bf="$(jq -r '.body_file' <<<"$line")"
        if [ -z "$target" ]; then
          if [ "$dry" = 1 ]; then echo "DRY comment on $issue (resolves after create)"; continue; fi
          conflict "comment on unmapped $issue"; echo "$line" >> "$remaining"; failed=$((failed+1)); continue
        fi
        if [ "$dry" = 1 ]; then echo "DRY comment on #$target"; continue; fi
        if rewrite_refs "$bf" | gh issue comment "$target" --repo "$repo" --body-file - >/dev/null 2>&1; then
          echo "commented on #$target"; applied=$((applied+1))
        else conflict "comment on #$target failed"; echo "$line" >> "$remaining"; failed=$((failed+1)); fi
        ;;
      relabel)
        issue="$(jq -r '.issue' <<<"$line")"; target="$(resolve_issue "$issue")"
        if [ -z "$target" ]; then
          if [ "$dry" = 1 ]; then echo "DRY relabel on $issue (resolves after create)"; continue; fi
          conflict "relabel on unmapped $issue"; echo "$line" >> "$remaining"; failed=$((failed+1)); continue
        fi
        local cur add rem
        cur="$(gh issue view "$target" --repo "$repo" --json labels -q '[.labels[].name]' 2>/dev/null)"
        if [ -z "$cur" ]; then conflict "relabel #$target: issue not found upstream"; echo "$line" >> "$remaining"; failed=$((failed+1)); continue; fi
        # Delta: only add what's missing, only remove what's present (preserve others' edits).
        add="$(jq -rc --argjson cur "$cur" '[.add[]? | select(. as $x | ($cur|index($x))|not)]' <<<"$line")"
        rem="$(jq -rc --argjson cur "$cur" '[.remove[]? | select(. as $x | ($cur|index($x)))]' <<<"$line")"
        if [ "$dry" = 1 ]; then echo "DRY relabel #$target add=$add remove=$rem (current=$cur)"; continue; fi
        local eargs=()
        [ "$add" != "[]" ] && eargs+=(--add-label "$(jq -r 'join(",")' <<<"$add")")
        [ "$rem" != "[]" ] && eargs+=(--remove-label "$(jq -r 'join(",")' <<<"$rem")")
        if [ ${#eargs[@]} -eq 0 ]; then echo "relabel #$target: already satisfied"; applied=$((applied+1));
        elif gh issue edit "$target" --repo "$repo" "${eargs[@]}" >/dev/null 2>&1; then
          echo "relabelled #$target"; applied=$((applied+1))
        else conflict "relabel #$target failed"; echo "$line" >> "$remaining"; failed=$((failed+1)); fi
        ;;
      state)
        issue="$(jq -r '.issue' <<<"$line")"; target="$(resolve_issue "$issue")"
        local st; st="$(jq -r '.state' <<<"$line")"
        if [ -z "$target" ]; then
          if [ "$dry" = 1 ]; then echo "DRY state on $issue -> $st (resolves after create)"; continue; fi
          conflict "state on unmapped $issue"; echo "$line" >> "$remaining"; failed=$((failed+1)); continue
        fi
        if [ "$dry" = 1 ]; then echo "DRY state #$target -> $st"; continue; fi
        if { [ "$st" = closed ] && gh issue close "$target" --repo "$repo" >/dev/null 2>&1; } \
        || { [ "$st" = open ] && gh issue reopen "$target" --repo "$repo" >/dev/null 2>&1; }; then
          echo "set #$target -> $st"; applied=$((applied+1))
        else conflict "state #$target -> $st failed"; echo "$line" >> "$remaining"; failed=$((failed+1)); fi
        ;;
      *) conflict "unknown op '$op' ($id)"; echo "$line" >> "$remaining"; failed=$((failed+1)) ;;
    esac
  done < "$JOURNAL"

  if [ "$dry" = 1 ]; then rm -f "$remaining"; echo "Dry run only — nothing applied." >&2; return 0; fi

  # Persist what was applied; keep only failures for retry.
  { echo "# sync $(now): applied=$applied failed=$failed"; } >> "$SYNCED"
  mv "$remaining" "$JOURNAL"
  echo "Synced: $applied applied, $failed left in queue." >&2
  [ "$failed" -gt 0 ] && echo "See $CONFLICTS and retry 'make issues-sync' after resolving." >&2
  echo "Refreshing cache ..." >&2
  cmd_pull
}

# --- status ------------------------------------------------------------------
cmd_status() {
  echo "Issue cache status:"
  if [ -f "$META" ]; then
    printf "  repo:      %s\n" "$(jq -r '.repo // "?"' "$META")"
    printf "  pulled:    %s (%s cached)\n" "$(jq -r '.pulled_at // "never"' "$META")" "$(jq -r '.count // 0' "$META")"
  else
    printf "  (no cache yet — run 'make issues-pull' online)\n"
  fi
  local pending; pending="$(journal_count)"
  printf "  queued:    %s pending change(s)\n" "$pending"
  if [ "$pending" -gt 0 ]; then
    jq -rc '"    - \(.op) \(.issue // .temp // "")"' "$JOURNAL" 2>/dev/null | sort | uniq -c | sed 's/^/    /'
  fi
  if online; then printf "  network:   online\n"; else printf "  network:   OFFLINE\n"; fi
  [ -f "$CONFLICTS" ] && printf "  conflicts: see %s\n" "$CONFLICTS"
  return 0
}

# --- dispatch ----------------------------------------------------------------
cmd="${1:-}"; shift || true
case "$cmd" in
  pull)    cmd_pull "$@" ;;
  list)    cmd_list "$@" ;;
  view)    cmd_view "$@" ;;
  comment) cmd_comment "$@" ;;
  relabel) cmd_relabel "$@" ;;
  state)   cmd_state "$@" ;;
  create)  cmd_create "$@" ;;
  sync)    cmd_sync "$@" ;;
  status)  cmd_status "$@" ;;
  ""|-h|--help|help)
    sed -n '3,40p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//' ;;
  *) die "unknown command '$cmd' (try: pull list view comment relabel state create sync status)" ;;
esac
