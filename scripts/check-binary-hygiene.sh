#!/usr/bin/env bash
set -euo pipefail

violations=0

while IFS=$'\t' read -r meta file; do
  mode="$(awk '{print $1}' <<< "$meta")"
  if [[ "$mode" != "100755" ]]; then
    continue
  fi

  # Allow executable scripts.
  case "$file" in
    *.sh|*.bash|*.zsh|*.py|*.pl|*.rb)
      continue
      ;;
  esac

  # Explicit denylist for known binary-like paths.
  case "$file" in
    backend/cmd/server/server|bin/*|*.exe|*.dll|*.so|*.dylib|*.bin|*.out)
      echo "Tracked binary artifact forbidden: $file"
      violations=1
      ;;
  esac
done < <(git ls-files -s)

if [[ $violations -ne 0 ]]; then
  echo "Binary hygiene check failed."
  exit 1
fi

echo "Binary hygiene check passed."
