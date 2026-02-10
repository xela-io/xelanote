#!/usr/bin/env bash
# Checks that environment variables in Go source, docs, and .env.example are in sync.
# Exit code 0 = all synced, 1 = drift detected.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BACKEND="$ROOT/backend"
DOCS="$ROOT/docs/environment-variables.md"
ENV_EXAMPLE="$ROOT/backend/.env.example"

errors=0

# Extract env vars from Go source (os.Getenv calls)
go_vars=$(grep -roh 'os\.Getenv("[^"]*")' "$BACKEND" --include='*.go' \
  | sed 's/os\.Getenv("//;s/")//' \
  | sort -u)

# Extract env vars from docs (backtick-quoted variable names in table rows)
doc_vars=$(grep -oP '^\| `\K[A-Z_]+(?=`)' "$DOCS" | sort -u)

# Extract env vars from .env.example (lines with KEY= or # KEY=)
env_example_vars=$(grep -oP '^#?\s*\K[A-Z_]+(?==)' "$ENV_EXAMPLE" | sort -u)

echo "=== Environment Variable Sync Check ==="
echo ""
echo "Go source:  $(echo "$go_vars" | wc -l) vars"
echo "Docs:       $(echo "$doc_vars" | wc -l) vars"
echo ".env.example: $(echo "$env_example_vars" | wc -l) vars"
echo ""

# Check: vars in Go but not in docs
missing_docs=$(comm -23 <(echo "$go_vars") <(echo "$doc_vars") || true)
if [ -n "$missing_docs" ]; then
  echo "FAIL: In Go source but missing from docs/environment-variables.md:"
  echo "$missing_docs" | sed 's/^/  - /'
  errors=1
  echo ""
fi

# Check: vars in Go but not in .env.example
missing_example=$(comm -23 <(echo "$go_vars") <(echo "$env_example_vars") || true)
if [ -n "$missing_example" ]; then
  echo "FAIL: In Go source but missing from .env.example:"
  echo "$missing_example" | sed 's/^/  - /'
  errors=1
  echo ""
fi

# Check: vars in docs but not in Go source (stale docs)
stale_docs=$(comm -13 <(echo "$go_vars") <(echo "$doc_vars") || true)
if [ -n "$stale_docs" ]; then
  echo "WARN: In docs but not found in Go source (possibly stale):"
  echo "$stale_docs" | sed 's/^/  - /'
  echo ""
fi

if [ "$errors" -eq 0 ]; then
  echo "OK: All environment variables are in sync."
fi

exit "$errors"
