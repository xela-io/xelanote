#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CERT_DIR="$ROOT_DIR/.certs"

DEV_HOST="dev.xelanote.local"
PWA_HOST="pwa.xelanote.local"

echo "Local phone HTTPS setup for xelanote"
echo

if ! command -v mkcert >/dev/null 2>&1; then
  echo "mkcert not found."
  echo "Install (macOS): brew install mkcert"
  echo "Then run: mkcert -install"
  exit 1
fi

mkdir -p "$CERT_DIR"

echo "Generating certificates in $CERT_DIR ..."
mkcert -cert-file "$CERT_DIR/$DEV_HOST.pem" -key-file "$CERT_DIR/$DEV_HOST-key.pem" "$DEV_HOST"
mkcert -cert-file "$CERT_DIR/$PWA_HOST.pem" -key-file "$CERT_DIR/$PWA_HOST-key.pem" "$PWA_HOST"

cat <<EOF

Done.

Next steps:
1. Add hostnames on your machine (/etc/hosts):
   127.0.0.1 $DEV_HOST
   127.0.0.1 $PWA_HOST

2. Start backend (Air):
   make dev

3. Start frontend:
   make phone-frontend-dev      # HMR (UI work)
   make phone-frontend-preview  # build + preview (PWA/iPhone standalone tests)

4. Start Caddy (HTTPS on :8443, no sudo needed):
   caddy run --config $ROOT_DIR/Caddyfile.local

5. Open on iPhone (same Wi-Fi):
   https://<YOUR-LAN-IP>    (if using tunnel/proxy with LAN hostname)
   or configure LAN DNS for:
   https://$DEV_HOST:8443 / https://$PWA_HOST:8443

Important:
- iPhone must trust the local CA if you use mkcert-generated certs directly.
- For easiest HTTPS on iPhone, use a tunnel to the Caddy endpoint.
EOF
