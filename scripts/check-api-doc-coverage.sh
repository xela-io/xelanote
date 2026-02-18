#!/usr/bin/env bash
# Verifies that API routes defined in backend/internal/api/routes*.go
# are documented as method headings in docs/api.md.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DOC="$ROOT/docs/api.md"
ROUTES_GLOB="$ROOT/backend/internal/api/routes*.go"
BASELINE="$ROOT/scripts/api-doc-coverage-baseline.txt"

if [ ! -f "$DOC" ]; then
  echo "FAIL: docs/api.md not found"
  exit 1
fi

if ! compgen -G "$ROUTES_GLOB" > /dev/null; then
  echo "FAIL: No route files found at backend/internal/api/routes*.go"
  exit 1
fi

ACTUAL=$(mktemp)
DOCUMENTED=$(mktemp)
MISSING_FILE=$(mktemp)
trap 'rm -f "$ACTUAL" "$DOCUMENTED" "$MISSING_FILE"' EXIT

perl -ne '
sub strip_trailing_slash {
  my ($p) = @_;
  $p =~ s#/$## if $p ne "/";
  return $p;
}
sub normalize_path {
  my ($p) = @_;
  $p =~ s#/+#/#g;
  $p = "/" if $p eq "";
  return strip_trailing_slash($p);
}
sub normalize_placeholders {
  my ($p) = @_;
  $p =~ s/\{[^}]+\}/:param/g;
  $p =~ s/:[^\/\s]+/:param/g;
  return strip_trailing_slash($p);
}
sub join_path {
  my ($a, $b) = @_;
  $a = normalize_path($a);
  $b = normalize_path($b);

  $a = "" if $a eq "/";
  $b = "" if $b eq "/";

  my $out;
  if ($a eq "") {
    $out = ($b eq "") ? "/" : $b;
  } elsif ($b eq "") {
    $out = $a;
  } elsif ($a =~ m#/$# && $b =~ m#^/#) {
    $out = $a . substr($b, 1);
  } elsif ($a !~ m#/$# && $b !~ m#^/#) {
    $out = $a . "/" . $b;
  } else {
    $out = $a . $b;
  }

  return normalize_path($out);
}

BEGIN {
  our @route_stack = ();
  our @route_depth = ();
  our $depth = 0;
}

my $line = $_;

if ($line =~ /\.((?:Get|Post|Put|Patch|Delete))\("([^"]+)"/) {
  my ($method, $suffix) = (uc($1), $2);
  my $full = "";
  for my $prefix (@route_stack) {
    $full = join_path($full, $prefix);
  }
  $full = join_path($full, $suffix);

  if ($full ne "/health" && $full ne "/captcha") {
    if ($full !~ m#^/api(?:/|$)#) {
      $full = join_path("/api", $full);
    }
    $full = normalize_placeholders($full);
    print "$method $full\n";
  }
}

my $pending_route;
if ($line =~ /r\.Route\("([^"]+)"/) {
  $pending_route = $1;
}

my $opens = () = ($line =~ /\{/g);
my $closes = () = ($line =~ /\}/g);
$depth += ($opens - $closes);

if (defined $pending_route) {
  push @route_stack, $pending_route;
  push @route_depth, $depth;
}

while (@route_stack && $depth < $route_depth[-1]) {
  pop @route_stack;
  pop @route_depth;
}
' $ROUTES_GLOB | sort -u > "$ACTUAL"

perl -ne '
if (/^###+\s+(GET|POST|PUT|PATCH|DELETE)\s+([^\s]+)/) {
  my ($method, $path) = ($1, $2);
  $path =~ s/\{[^}]+\}/:param/g;
  $path =~ s/:[^\/\s]+/:param/g;
  $path =~ s#/$## if $path ne "/";
  print "$method $path\n";
}
' "$DOC" | sort -u > "$DOCUMENTED"

MISSING=$(comm -23 "$ACTUAL" "$DOCUMENTED" || true)
printf "%s\n" "$MISSING" | sed '/^$/d' | sort -u > "$MISSING_FILE"

if [ "${1:-}" = "--update-baseline" ]; then
  cp "$MISSING_FILE" "$BASELINE"
  COUNT=$(wc -l < "$BASELINE" | tr -d ' ')
  echo "Updated API doc coverage baseline with $COUNT undocumented route(s)."
  exit 0
fi

if [ ! -f "$BASELINE" ]; then
  echo "FAIL: Baseline file not found: $BASELINE"
  echo "Create it with:"
  echo "  bash scripts/check-api-doc-coverage.sh --update-baseline"
  exit 1
fi

if ! diff -u "$BASELINE" "$MISSING_FILE" > /dev/null 2>&1; then
  echo "FAIL: API doc coverage drift detected."
  echo ""
  echo "Differences (- = baseline undocumented, + = current undocumented):"
  diff -u "$BASELINE" "$MISSING_FILE" || true
  echo ""
  echo "Document newly missing routes in docs/api.md."
  echo "If this change is intentional, update the baseline:"
  echo "  bash scripts/check-api-doc-coverage.sh --update-baseline"
  exit 1
fi

TOTAL=$(wc -l < "$ACTUAL" | tr -d ' ')
MISSING_COUNT=$(wc -l < "$MISSING_FILE" | tr -d ' ')
echo "OK: API doc coverage check passed (registered: $TOTAL, baseline-undocumented: $MISSING_COUNT)."
