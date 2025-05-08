#!/usr/bin/env bash
# cockcli_menu.sh – Browse, edit and sync Cockpit CMS entries via cockcli + fzf
#
# Usage:  cockcli_menu.sh <instance-alias>
#
# Requirements
#   • cockcli (from github.com/joona/cockcli) on PATH
#   • fzf ≥0.30, sha256sum, mktemp
#   • $EDITOR set (default: vim)
#
# Environment overrides
#   FORMAT   – json | yaml   (default: json)
#   TMPDIR   – directory for temporary files
#
set -euo pipefail

if [[ "$#" -lt 1 ]]; then
  echo "Usage: $0 <instance-alias>" >&2
  exit 1
fi
INSTANCE="$1"; shift
FMT="${FORMAT:-yaml}"
EDITOR_CMD="${EDITOR:-vim}"

# Helper: die with message
_die() { echo "error: $*" >&2; exit 1; }

# Helper: run cockcli with --instance immediately after the binary
cc() { cockcli --instance "$INSTANCE" "$@"; }

# 1. Pick a collection ─────────────────────────────────────
collection=$(cc list | fzf --prompt="Collections> ") || exit 0

# 2. Pick an item (ID) ─────────────────────────────────────
item_id=$(cc list "$collection" | awk '{ print $0 }' | fzf --prompt="$collection items> " | awk '{ print $1 }') || exit 0

# 3. Download the entry to a temp file (JSON or YAML) ──────
tmp=$(mktemp --tmpdir="${TMPDIR:-}" "cockpit_${item_id}.XXXXXXXXXX.${FMT}")
trap 'rm -f "$tmp"' EXIT

cc get --format "$FMT" --output "$tmp" "$collection" "$item_id"

orig_hash=$(sha256sum "$tmp" | cut -d' ' -f1)

# 4. Open in editor ─────────────────────────────────────────
"$EDITOR_CMD" "$tmp"

new_hash=$(sha256sum "$tmp" | cut -d' ' -f1)
[[ "$orig_hash" == "$new_hash" ]] && {
  echo "No changes made to $item_id"
  exit 0
}

# Show diff ────────────────────────────────────────────────
# get diff output by dry running update
cc update --dry-run --tempfile "$tmp" "$collection" 2>&1


# 5. Confirm & push update ─────────────────────────────────
# Show the diff and ask for confirmation without fzf
echo 
echo "Do you want to update $item_id? (y/n)"
read -r -n 1 -s answer
echo
if [[ "$answer" != "y" ]]; then
  echo "Update cancelled."
  exit 0
fi

if cc update "$collection" "$tmp"; then
  echo "Updated $item_id"
else
  echo "Update failed."
fi
