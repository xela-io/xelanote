#!/usr/bin/env bash
# Checks frontend bundle size after build and fails if it exceeds the threshold.
# Run after `npm run build` in the frontend directory.
# Exit code 0 = within budget, 1 = over budget.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BUILD_DIR="$ROOT/frontend/build"

# Budget: total JS size in KB (adjust as needed)
JS_BUDGET_KB=3600

if [ ! -d "$BUILD_DIR" ]; then
  echo "SKIP: Build directory not found ($BUILD_DIR). Run 'npm run build' first."
  exit 0
fi

JS_DIR="$BUILD_DIR/_app/immutable"
if [ ! -d "$JS_DIR" ]; then
  echo "SKIP: Immutable assets directory not found ($JS_DIR)."
  exit 0
fi

# Measure total JS bundle size (gzipped approximation not needed - raw size is the signal)
JS_SIZE_BYTES=$(find "$JS_DIR" -name '*.js' -exec cat {} + | wc -c)
JS_SIZE_KB=$((JS_SIZE_BYTES / 1024))

CSS_SIZE_BYTES=$(find "$JS_DIR" -name '*.css' -exec cat {} + 2>/dev/null | wc -c)
CSS_SIZE_KB=$((CSS_SIZE_BYTES / 1024))

TOTAL_KB=$((JS_SIZE_KB + CSS_SIZE_KB))

echo "=== Frontend Bundle Size ==="
echo "  JS:    ${JS_SIZE_KB} KB"
echo "  CSS:   ${CSS_SIZE_KB} KB"
echo "  Total: ${TOTAL_KB} KB (budget: ${JS_BUDGET_KB} KB)"
echo ""

if [ "$TOTAL_KB" -gt "$JS_BUDGET_KB" ]; then
  echo "FAIL: Bundle size (${TOTAL_KB} KB) exceeds budget (${JS_BUDGET_KB} KB)."
  echo "Run 'npm run analyze' to identify large modules."
  exit 1
fi

echo "OK: Bundle size within budget."
