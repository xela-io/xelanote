#!/usr/bin/env bash
# Blocks localStorage persistence of auth tokens.
# Checks for setItem with token/auth/jwt keys and direct property assignment.
# Limitation: Only detects string-literal keys. Variable-based keys require code review.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$ROOT/frontend/src"

if [ ! -d "$SRC" ]; then
  echo "SKIP: frontend/src not found."
  exit 0
fi

VIOLATIONS=$(mktemp)
trap 'rm -f "$VIOLATIONS"' EXIT

# Search production code only (exclude test files and test directory)
find "$SRC" -type f \( -name '*.ts' -o -name '*.svelte' -o -name '*.js' \) \
  ! -name '*.test.ts' ! -name '*.spec.ts' ! -path '*/test/*' \
  | sort | while IFS= read -r file; do

  REL="${file#"$ROOT"/}"

  # Check localStorage.setItem with auth-related string-literal keys (case insensitive)
  grep -inE 'localStorage\.setItem\([[:space:]]*['"'"'"][^'"'"'"]*\b(token|auth|jwt)\b' "$file" \
    | while IFS= read -r match; do
      echo "$REL:$match"
    done || true

  # Check direct property assignment: localStorage.token = ...
  grep -nE 'localStorage\.(token|auth[Tt]oken|jwt|accessToken|refreshToken)[[:space:]]*=' "$file" \
    | while IFS= read -r match; do
      echo "$REL:$match"
    done || true

done > "$VIOLATIONS" || true

if [ -s "$VIOLATIONS" ]; then
  COUNT=$(wc -l < "$VIOLATIONS" | tr -d ' ')
  echo "FAIL: Found $COUNT localStorage auth persistence pattern(s)."
  echo "Auth tokens must not be stored in localStorage (use httpOnly cookies)."
  echo ""
  cat "$VIOLATIONS"
  exit 1
fi

echo "OK: No localStorage auth persistence found."
