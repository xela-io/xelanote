#!/usr/bin/env bash
# ufw-cloudflare-only.sh
# Restricts HTTP/HTTPS to Cloudflare IPs only.
# Run on the Hetzner production server as root.
#
# Usage:
#   sudo bash ufw-cloudflare-only.sh          # Apply rules
#   sudo bash ufw-cloudflare-only.sh --dry-run # Show what would be done
#
# Can be scheduled via cron to pick up new Cloudflare IPs:
#   0 3 * * 0  /root/ufw-cloudflare-only.sh >> /var/log/ufw-cloudflare-update.log 2>&1

set -euo pipefail

DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

CF_V4_URL="https://www.cloudflare.com/ips-v4"
CF_V6_URL="https://www.cloudflare.com/ips-v6"

# Fallback IPs if fetch fails (last known good - 2026-02)
FALLBACK_V4=(
    173.245.48.0/20
    103.21.244.0/22
    103.22.200.0/22
    103.31.4.0/22
    141.101.64.0/18
    108.162.192.0/18
    190.93.240.0/20
    188.114.96.0/20
    197.234.240.0/22
    198.41.128.0/17
    162.158.0.0/15
    104.16.0.0/13
    104.24.0.0/14
    172.64.0.0/13
    131.0.72.0/22
)

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

run_cmd() {
    if $DRY_RUN; then
        echo "  [DRY-RUN] $*"
    else
        "$@"
    fi
}

# --- Fetch current Cloudflare IPs ---
log "Fetching Cloudflare IPv4 ranges from $CF_V4_URL ..."
CF_V4=$(curl -sf --max-time 10 "$CF_V4_URL" 2>/dev/null || true)

if [[ -z "$CF_V4" ]]; then
    log "WARNING: Could not fetch IPv4 ranges, using fallback list"
    CF_V4=$(printf '%s\n' "${FALLBACK_V4[@]}")
fi

log "Fetching Cloudflare IPv6 ranges from $CF_V6_URL ..."
CF_V6=$(curl -sf --max-time 10 "$CF_V6_URL" 2>/dev/null || true)

# Validate: must have at least 10 IPv4 ranges (sanity check)
V4_COUNT=$(echo "$CF_V4" | grep -c '/')
if [[ "$V4_COUNT" -lt 10 ]]; then
    log "ERROR: Only $V4_COUNT IPv4 ranges found. Aborting to prevent lockout."
    exit 1
fi

log "Found $V4_COUNT IPv4 ranges"
[[ -n "$CF_V6" ]] && log "Found $(echo "$CF_V6" | grep -c '/') IPv6 ranges"

# --- Remove existing HTTP/HTTPS rules ---
log "Removing existing rules for ports 80 and 443 ..."

# Delete rules by filtering ufw status (in reverse order to preserve numbering)
# We need to parse numbered rules and delete from highest to lowest
if ! $DRY_RUN; then
    # Get rule numbers for port 80 and 443, delete in reverse order
    while true; do
        RULE_NUM=$(ufw status numbered 2>/dev/null \
            | grep -E '\b(80|443)\b' \
            | head -1 \
            | grep -oP '^\[\s*\K[0-9]+' || true)
        [[ -z "$RULE_NUM" ]] && break
        echo "y" | ufw delete "$RULE_NUM"
    done
else
    echo "  [DRY-RUN] Would remove all existing port 80/443 rules"
fi

# --- Add Cloudflare-only rules ---
log "Adding IPv4 allow rules for ports 80 and 443 ..."
while IFS= read -r cidr; do
    [[ -z "$cidr" || "$cidr" =~ ^# ]] && continue
    cidr=$(echo "$cidr" | tr -d '[:space:]')
    run_cmd ufw allow from "$cidr" to any port 80 proto tcp comment "Cloudflare IPv4"
    run_cmd ufw allow from "$cidr" to any port 443 proto tcp comment "Cloudflare IPv4"
done <<< "$CF_V4"

if [[ -n "$CF_V6" ]]; then
    log "Adding IPv6 allow rules for ports 80 and 443 ..."
    while IFS= read -r cidr; do
        [[ -z "$cidr" || "$cidr" =~ ^# ]] && continue
        cidr=$(echo "$cidr" | tr -d '[:space:]')
        run_cmd ufw allow from "$cidr" to any port 80 proto tcp comment "Cloudflare IPv6"
        run_cmd ufw allow from "$cidr" to any port 443 proto tcp comment "Cloudflare IPv6"
    done <<< "$CF_V6"
fi

# --- Ensure default deny is set ---
run_cmd ufw default deny incoming

# --- Remove port 8080 if still open ---
if ufw status 2>/dev/null | grep -q '8080'; then
    log "Removing public port 8080 rule (no longer needed with Cloudflare proxy) ..."
    run_cmd ufw delete allow 8080/tcp || true
fi

log "Done. Current UFW status:"
ufw status verbose

log ""
log "IMPORTANT: Verify you can still access the server via SSH before closing this session!"
log "  Run in another terminal: ssh xelanote-prod"
