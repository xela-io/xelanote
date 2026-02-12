#!/usr/bin/env bash
# Checks that api/ -> db/ layer violations match the known baseline.
# New violations cause failure. Removing a violation without updating
# the baseline also fails (keeps the ratchet tight).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BACKEND="$ROOT/backend"
BASELINE="$ROOT/scripts/layer-violation-baseline.txt"

if [ ! -f "$BASELINE" ]; then
  echo "FAIL: Baseline file not found: $BASELINE"
  exit 1
fi

# Find all non-test .go files in api/ that import internal/db
ACTUAL=$(mktemp)
trap 'rm -f "$ACTUAL"' EXIT

for f in "$BACKEND"/internal/api/*.go; do
  [ -f "$f" ] || continue
  case "$f" in *_test.go) continue;; esac
  if grep -q '"github.com/xela-io/xelanote/internal/db' "$f"; then
    # Output relative path from backend/
    echo "${f#"$BACKEND"/}"
  fi
done | sort > "$ACTUAL"

# Compare against baseline
if diff -u "$BASELINE" "$ACTUAL" > /dev/null 2>&1; then
  COUNT=$(wc -l < "$BASELINE" | tr -d ' ')
  echo "OK: Layer violations match baseline ($COUNT files)."
  exit 0
fi

echo "FAIL: Layer violations differ from baseline."
echo ""
echo "Differences (- = baseline, + = actual):"
diff -u "$BASELINE" "$ACTUAL" || true
echo ""
echo "If you REMOVED a violation, update the baseline:"
echo "  grep -rl '\"github.com/xela-io/xelanote/internal/db' backend/internal/api/*.go \\"
echo "    | grep -v '_test\\.go\$' | sed 's|^backend/||' | sort > scripts/layer-violation-baseline.txt"
echo ""
echo "If you ADDED a new violation, refactor to use the service layer instead."
exit 1
