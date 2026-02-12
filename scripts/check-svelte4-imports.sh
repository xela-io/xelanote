#!/usr/bin/env bash
# Blocks forbidden Svelte 4 store imports.
# Allowed: `import { get } from 'svelte/store'` (Svelte 5 compatible).
# Blocked: writable, readable, derived, or namespace imports.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$ROOT/frontend/src"

if [ ! -d "$SRC" ]; then
  echo "SKIP: frontend/src not found."
  exit 0
fi

VIOLATIONS=$(mktemp)
trap 'rm -f "$VIOLATIONS"' EXIT

# Find all imports from 'svelte/store' or "svelte/store" in .ts/.svelte/.js files
find "$SRC" -type f \( -name '*.ts' -o -name '*.svelte' -o -name '*.js' \) \
  -exec grep -Hn "from ['\"]svelte/store['\"]" {} + 2>/dev/null \
  | while IFS= read -r line; do
    # Allow `{ get }` only imports
    if echo "$line" | grep -qE 'import[[:space:]]+\{[[:space:]]*get[[:space:]]*\}[[:space:]]+from'; then
      continue
    fi
    echo "$line"
  done > "$VIOLATIONS"

if [ -s "$VIOLATIONS" ]; then
  COUNT=$(wc -l < "$VIOLATIONS" | tr -d ' ')
  echo "FAIL: Found $COUNT forbidden svelte/store import(s)."
  echo "Only \`import { get } from 'svelte/store'\` is allowed (Svelte 5 compatible)."
  echo ""
  cat "$VIOLATIONS"
  exit 1
fi

echo "OK: No forbidden svelte/store imports found."
