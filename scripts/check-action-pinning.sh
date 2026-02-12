#!/usr/bin/env bash
set -euo pipefail

bad_refs=0

while IFS= read -r workflow; do
  while IFS= read -r line; do
    if [[ "$line" =~ uses:[[:space:]]*[^[:space:]]+@(main|master)$ ]]; then
      echo "Mutable action ref forbidden: ${workflow}: ${line}"
      bad_refs=1
    fi
  done < "$workflow"
done < <(find .github/workflows -type f \( -name '*.yml' -o -name '*.yaml' \))

if [[ $bad_refs -ne 0 ]]; then
  echo "Action pinning check failed. Replace mutable refs with fixed versions or commit SHAs."
  exit 1
fi

echo "Action pinning check passed."
