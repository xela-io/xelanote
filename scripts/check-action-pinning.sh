#!/usr/bin/env bash
set -euo pipefail

bad_refs=0

while IFS= read -r workflow; do
  lineno=0
  while IFS= read -r line; do
    lineno=$((lineno + 1))

    # Match lines like: uses: owner/repo@ref
    if [[ "$line" =~ uses:[[:space:]]*([^[:space:]]+)@([^[:space:]]+) ]]; then
      action="${BASH_REMATCH[1]}"
      ref="${BASH_REMATCH[2]}"

      # Reject mutable branch refs for any action
      if [[ "$ref" == "main" || "$ref" == "master" ]]; then
        echo "FAIL: Mutable branch ref: ${workflow}:${lineno}: ${action}@${ref}"
        bad_refs=1
        continue
      fi

      # SEC-004: Third-party actions (not actions/*) must be SHA-pinned
      if [[ "$action" != actions/* ]] && [[ ! "$ref" =~ ^[0-9a-f]{40}$ ]]; then
        echo "FAIL: Third-party action not SHA-pinned: ${workflow}:${lineno}: ${action}@${ref}"
        bad_refs=1
      fi
    fi
  done < "$workflow"
done < <(find .github/workflows -type f \( -name '*.yml' -o -name '*.yaml' \))

if [[ $bad_refs -ne 0 ]]; then
  echo "Action pinning check failed. Pin third-party actions to commit SHAs (actions/* may use version tags)."
  exit 1
fi

echo "Action pinning check passed."
