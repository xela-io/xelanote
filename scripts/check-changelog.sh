#!/usr/bin/env bash
# Ensures CHANGELOG.md is updated when code files are changed.
# Skips check for docs-only, config-only, or test-only commits.
set -euo pipefail

STAGED=$(git diff --cached --name-only --diff-filter=ACMR)

# Skip if nothing is staged
if [ -z "$STAGED" ]; then
  exit 0
fi

# Check if any code files (non-docs, non-config) are staged
CODE_CHANGED=false
while IFS= read -r file; do
  case "$file" in
    backend/internal/*|backend/cmd/*|frontend/src/*) CODE_CHANGED=true; break ;;
  esac
done <<< "$STAGED"

# If no code files changed, skip the check
if [ "$CODE_CHANGED" = false ]; then
  exit 0
fi

# Check if CHANGELOG.md is in the staged files
if echo "$STAGED" | grep -q '^CHANGELOG.md$'; then
  exit 0
fi

echo "ERROR: Code files changed but CHANGELOG.md not updated."
echo "Please add an entry to the [Unreleased] section in CHANGELOG.md."
echo ""
echo "To skip this check (e.g. for pure refactoring): LEFTHOOK=0 git commit ..."
exit 1
